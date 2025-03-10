// Copyright 2018 The aquachain Authors
// This file is part of aquachain.
//
// aquachain is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// aquachain is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with aquachain. If not, see <http://www.gnu.org/licenses/>.

// Package utils contains internal helper functions for aquachain commands.
package subcommands

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gitlab.com/aquachain/aquachain/aqua"
	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/core"
	"gitlab.com/aquachain/aquachain/core/types"
	"gitlab.com/aquachain/aquachain/node"
	"gitlab.com/aquachain/aquachain/rlp"
)

const (
	importBatchSize = aqua.ImportBatchSize
)

var start_time = time.Now().UTC()

func GetStartTime() time.Time {
	return start_time
}

func StartNode(ctx context.Context, stack *node.Node) chan struct{} {
	ch := make(chan struct{})
	go func() {
		defer close(ch)
		if err := stack.Start(ctx); err != nil {
			log.GracefulShutdownf("Error starting protocol stack: %+v", log.TranslateFatalError(err))
			time.Sleep(time.Second * 3) // so caller can wait for close(ch)
		}
	}()
	return ch
	// go func() {
	// 	log.Info("node.Node waiting for interrupt")
	// 	sigc := make(chan os.Signal, 1)
	// 	reason := ""
	// 	<-ctx.Done()
	// 	reason = context.Cause(ctx).Error()
	// 	// for force-panic on multiple interrupts
	// 	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	// 	defer signal.Stop(sigc)
	// 	go alerts.Warnf("Got %s, shutting down...", reason) // might not make it
	// 	log.Info("Got interrupt, shutting down...", "reason", reason)
	// 	go stack.Stop()

	// 	for i := 10; i > 0; i-- {
	// 		<-sigc // blocks, something should os.Exit(1) in the background
	// 		if i > 1 {
	// 			log.Warn("Already shutting down, interrupt more to panic.", "times", i-1)
	// 		}
	// 	}

	// 	time.Sleep(time.Second) // TODO remove this

	// 	debug.Exit() // ensure trace and CPU profile data is flushed.
	// 	debug.LoudPanic("boom")
	// }()
}

func ImportChain(chain *core.BlockChain, fn string) error {
	// Watch for Ctrl-C while the import is running.
	// If a signal is received, the import will stop at the next batch.
	interrupt := make(chan os.Signal, 1)
	stop := make(chan struct{})
	log.Info("Press ctrl+c to stop import")
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(interrupt)
	defer close(interrupt)
	go func() {
		if _, ok := <-interrupt; ok {
			log.Info("Interrupted during import, stopping at next batch")
		}
		close(stop)
	}()
	checkInterrupt := func() bool {
		select {
		case <-stop:
			return true
		default:
			return false
		}
	}

	log.Info("Importing blockchain", "file", fn)
	fh, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer fh.Close()

	var reader io.Reader = fh
	if strings.HasSuffix(fn, ".gz") {
		if reader, err = gzip.NewReader(reader); err != nil {
			return err
		}
	}
	stream := rlp.NewStream(reader, 0)

	// Run actual the import.
	blocks := make(types.Blocks, importBatchSize)
	n := 0
	for batch := 0; ; batch++ {
		// Load a batch of RLP blocks.
		if checkInterrupt() {
			return fmt.Errorf("interrupted")
		}
		i := 0
		for ; i < importBatchSize; i++ {
			var b types.Block
			if err := stream.Decode(&b); err == io.EOF {
				break
			} else if err != nil {
				return fmt.Errorf("at block %d: %v", n, err)
			}
			// don't import first block
			if b.NumberU64() == 0 {
				i--
				continue
			}
			blocks[i] = &b
			n++
		}
		if i == 0 {
			break
		}
		// Import the batch.
		if checkInterrupt() {
			return fmt.Errorf("interrupted")
		}
		missing := missingBlocks(chain, blocks[:i])
		if len(missing) == 0 {
			log.Info("Skipping batch as all blocks present", "size", len(blocks), "batch", batch, "first", blocks[0].Hash(), "last", blocks[i-1].Hash(), "blocknumber", blocks[i-1].Number(), "date", common.FormatTimestamp(blocks[i-1].Time().Uint64()))
			continue
		}
		if _, err := chain.InsertChain(missing); err != nil {
			return fmt.Errorf("%v", err)
		}
	}
	return nil
}

func missingBlocks(chain *core.BlockChain, blocks []*types.Block) []*types.Block {
	head := chain.CurrentBlock()
	for i, block := range blocks {
		// If we're behind the chain head, only check block, state is available at head
		if head.NumberU64() > block.NumberU64() {
			if !chain.HasBlock(block.SetVersion(chain.Config().GetBlockVersion(block.Number())), block.NumberU64()) {
				return blocks[i:]
			}
			continue
		}

		// If we're above the chain head, state availability is a must
		if !chain.HasBlockAndState(block.SetVersion(chain.Config().GetBlockVersion(block.Number())), block.NumberU64()) {
			return blocks[i:]
		}
	}
	return nil
}

func ExportChain(blockchain *core.BlockChain, fn string) error {
	log.Info("Exporting blockchain", "file", fn)
	fh, err := os.OpenFile(fn, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer fh.Close()

	var writer io.Writer = fh
	if strings.HasSuffix(fn, ".gz") {
		writer = gzip.NewWriter(writer)
		defer writer.(*gzip.Writer).Close()
	}

	if err := blockchain.Export(writer); err != nil {
		return err
	}
	log.Info("Exported blockchain", "file", fn)

	return nil
}

func ExportAppendChain(blockchain *core.BlockChain, fn string, first uint64, last uint64) error {
	log.Info("Exporting blockchain", "file", fn)
	// TODO verify mode perms
	fh, err := os.OpenFile(fn, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer fh.Close()

	var writer io.Writer = fh
	if strings.HasSuffix(fn, ".gz") {
		writer = gzip.NewWriter(writer)
		defer writer.(*gzip.Writer).Close()
	}

	if err := blockchain.ExportN(writer, first, last); err != nil {
		return err
	}
	log.Info("Exported blockchain to", "file", fn)
	return nil
}
