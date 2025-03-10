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

package subcommands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/aqua/downloader"
	"gitlab.com/aquachain/aquachain/aqua/event"
	"gitlab.com/aquachain/aquachain/aquadb"
	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/core"
	"gitlab.com/aquachain/aquachain/core/state"
	"gitlab.com/aquachain/aquachain/core/types"
	"gitlab.com/aquachain/aquachain/opt/console"
	"gitlab.com/aquachain/aquachain/subcommands/aquaflags"
	"gitlab.com/aquachain/aquachain/subcommands/mainctxs"
	"gitlab.com/aquachain/aquachain/trie"
)

func echo(_ context.Context, cmd *cli.Command) error {
	fmt.Println("echo")
	log.Info("echo")
	for _, v := range cmd.Root().Flags {
		fmt.Printf("%s: %v (isset=%v)\n", v.Names()[0], v.String(), v.IsSet())
	}
	return nil
}

var (
	echoCommand = &cli.Command{
		Action:      echo,
		Name:        "echo",
		Category:    "TEST COMMANDS",
		Description: "echo flag set",
	}
	initCommand = &cli.Command{
		Action:    MigrateFlags(initGenesis),
		Name:      "init",
		Usage:     "Bootstrap and initialize a new genesis block",
		ArgsUsage: "<genesisPath>",
		Flags: []cli.Flag{
			aquaflags.DataDirFlag,
		},
		Category: "BLOCKCHAIN COMMANDS",
		Description: `
The init command initializes a new genesis block and definition for the network.
This is a destructive action and changes the network in which you will be
participating.

It expects the genesis file as argument.`,
	}
	importCommand = &cli.Command{
		Action:    MigrateFlags(importChain),
		Name:      "import",
		Usage:     "Import a blockchain file",
		ArgsUsage: "<filename> (<filename 2> ... <filename N>) ",
		Flags: []cli.Flag{
			aquaflags.DataDirFlag,
			aquaflags.CacheFlag,
			aquaflags.GCModeFlag,
			aquaflags.CacheDatabaseFlag,
			aquaflags.CacheGCFlag,
		},
		Category: "BLOCKCHAIN COMMANDS",
		Description: `
The import command imports blocks from an RLP-encoded form. The form can be one file
with several RLP-encoded blocks, or several files can be used.

If only one file is used, import error will result in failure. If several files are used,
processing will proceed even if an individual RLP-file import failure occurs.`,
	}
	exportCommand = &cli.Command{
		Action:    MigrateFlags(exportChain),
		Name:      "export",
		Usage:     "Export blockchain into file",
		ArgsUsage: "<filename> [<blockNumFirst> <blockNumLast>]",
		Flags: []cli.Flag{
			// aquaflags.DataDirFlag,
			aquaflags.CacheFlag,
		},
		Category: "BLOCKCHAIN COMMANDS",
		Description: `
Requires a first argument of the file to write to.
Optional second and third arguments control the first and
last block to write. In this mode, the file will be appended
if already existing.`,
	}
	copydbCommand = &cli.Command{
		Action:    MigrateFlags(copyDb),
		Name:      "copydb",
		Usage:     "Create a local chain from a target chaindata folder",
		ArgsUsage: "<sourceChaindataDir>",
		Flags: []cli.Flag{
			// aquaflags.DataDirFlag,
			aquaflags.CacheFlag,
			aquaflags.SyncModeFlag,
			aquaflags.FakePoWFlag,
			// aquaflags.ChainFlag,
		},
		Category: "BLOCKCHAIN COMMANDS",
		Description: `
The first argument must be the directory containing the blockchain to download from`,
	}
	removedbCommand = &cli.Command{
		Action:    MigrateFlags(removeDB),
		Name:      "removedb",
		Usage:     "Remove blockchain and state databases",
		ArgsUsage: " ",
		Flags:     []cli.Flag{
			// aquaflags.DataDirFlag,
			// aquaflags.ChainFlag,
		},
		Category: "BLOCKCHAIN COMMANDS",
		Description: `
Remove blockchain and state databases`,
	}
	dumpCommand = &cli.Command{
		Action:    MigrateFlags(dump),
		Name:      "dump",
		Usage:     "Dump a specific block from storage",
		ArgsUsage: "[<blockHash> | <blockNum>]...",
		Flags: []cli.Flag{
			aquaflags.DataDirFlag,
			aquaflags.CacheFlag,
		},
		Category: "BLOCKCHAIN COMMANDS",
		Description: `
The arguments are interpreted as block numbers or hashes.
Use "aquachain dump 0" to dump the genesis block.`,
	}
)

// initGenesis will initialise the given JSON format genesis file and writes it as
// the zero'd block (i.e. genesis) or will fail hard if it can't succeed.
func initGenesis(ctx context.Context, cmd *cli.Command) error {
	// Make sure we have a valid genesis JSON
	genesisPath := cmd.Args().First()
	if len(genesisPath) == 0 {
		Fatalf("Must supply path to genesis JSON file")
	}
	file, err := os.Open(genesisPath)
	if err != nil {
		Fatalf("Failed to read genesis file: %v", err)
	}
	defer file.Close()

	genesis := new(core.Genesis)
	if err := json.NewDecoder(file).Decode(genesis); err != nil {
		Fatalf("invalid genesis file: %v", err)
	}

	if genesis.Config == nil || genesis.Config.ChainId == nil {
		Fatalf("invalid genesis file: no chainid")
	}

	// Open an initialise db
	stack := MakeFullNode(ctx, cmd)
	for _, name := range []string{"chaindata"} {
		chaindb, err := stack.OpenDatabase(name, 0, 0)
		if err != nil {
			Fatalf("Failed to open database: %v", err)
		}
		_, hash, err := core.SetupGenesisBlock(chaindb, genesis)
		if err != nil {
			Fatalf("Failed to write genesis block: %v", err)
		}
		log.Info("Successfully wrote genesis state", "database", name, "hash", hash)
	}
	return nil
}

func importChain(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 1 {
		Fatalf("This command requires an argument.")
	}
	stack := MakeFullNode(ctx, cmd)
	chain, chainDb := MakeChain(cmd, stack)
	defer chainDb.Close()

	// Start periodically gathering memory profiles
	var peakMemAlloc, peakMemSys uint64
	go func() {
		stats := new(runtime.MemStats)
		for {
			runtime.ReadMemStats(stats)
			if atomic.LoadUint64(&peakMemAlloc) < stats.Alloc {
				atomic.StoreUint64(&peakMemAlloc, stats.Alloc)
			}
			if atomic.LoadUint64(&peakMemSys) < stats.Sys {
				atomic.StoreUint64(&peakMemSys, stats.Sys)
			}
			time.Sleep(5 * time.Second)
		}
	}()
	// Import the chain
	start := time.Now()
	exitcode := 0

	if cmd.Args().Len() == 1 {
		if err := ImportChain(chain, cmd.Args().First()); err != nil {
			log.Error("Import error", "err", err)
			exitcode = 111
		}
	} else {
		for _, arg := range cmd.Args().Slice() {
			if err := ImportChain(chain, arg); err != nil {
				log.Error("Import error", "file", arg, "err", err)
			}
		}
	}
	chain.Stop()
	fmt.Printf("Import done in %v.\n\n", time.Since(start))

	// Output pre-compaction stats mostly to see the import trashing
	db := chainDb.(*aquadb.LDBDatabase)

	stats, err := db.LDB().GetProperty("leveldb.stats")
	if err != nil {
		Fatalf("Failed to read database stats: %v", err)
	}
	fmt.Println(stats)
	fmt.Printf("Trie cache misses:  %d\n", trie.CacheMisses())
	fmt.Printf("Trie cache unloads: %d\n\n", trie.CacheUnloads())

	// Print the memory statistics used by the importing
	mem := new(runtime.MemStats)
	runtime.ReadMemStats(mem)

	fmt.Printf("Object memory: %.3f MB current, %.3f MB peak\n", float64(mem.Alloc)/1024/1024, float64(atomic.LoadUint64(&peakMemAlloc))/1024/1024)
	fmt.Printf("System memory: %.3f MB current, %.3f MB peak\n", float64(mem.Sys)/1024/1024, float64(atomic.LoadUint64(&peakMemSys))/1024/1024)
	fmt.Printf("Allocations:   %.3f million\n", float64(mem.Mallocs)/1000000)
	fmt.Printf("GC pause:      %v\n\n", time.Duration(mem.PauseTotalNs))

	if cmd.IsSet(aquaflags.NoCompactionFlag.Name) {
		return nil
	}

	// Compact the entire database to more accurately measure disk io and print the stats
	start = time.Now()
	fmt.Println("Compacting entire database...")
	if err = db.LDB().CompactRange(util.Range{}); err != nil {
		Fatalf("Compaction failed: %v", err)
	}
	fmt.Printf("Compaction done in %v.\n\n", time.Since(start))

	stats, err = db.LDB().GetProperty("leveldb.stats")
	if err != nil {
		Fatalf("Failed to read database stats: %v", err)
	}
	fmt.Println(stats)

	if exitcode != 0 {
		log.Error("Import failed", "exitcode", exitcode)
		os.Exit(exitcode)
	}
	return nil
}

func exportChain(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 1 {
		Fatalf("This command requires an argument.")
	}
	stack := MakeFullNode(ctx, cmd)
	chain, _ := MakeChain(cmd, stack)
	start := time.Now()

	var err error
	fp := cmd.Args().First()
	if cmd.Args().Len() < 3 {
		err = ExportChain(chain, fp)
	} else {
		// This can be improved to allow for numbers larger than 9223372036854775807
		first, ferr := strconv.ParseInt(cmd.Args().Get(1), 10, 64)
		last, lerr := strconv.ParseInt(cmd.Args().Get(2), 10, 64)
		if ferr != nil || lerr != nil {
			Fatalf("Export error in parsing parameters: block number not an integer\n")
		}
		if first < 0 || last < 0 {
			Fatalf("Export error: block number must be greater than 0\n")
		}
		err = ExportAppendChain(chain, fp, uint64(first), uint64(last))
	}

	if err != nil {
		Fatalf("Export error: %v\n", err)
	}
	fmt.Printf("Export done in %v\n", time.Since(start))
	return nil
}

func copyDb(ctx context.Context, cmd *cli.Command) error {
	// Ensure we have a source chain directory to copy
	if cmd.Args().Len() != 1 {
		Fatalf("Source chaindata directory path argument missing")
	}
	// Initialize a new chain for the running node to sync into
	stack := MakeFullNode(ctx, cmd)
	chain, chainDb := MakeChain(cmd, stack)

	var syncmode downloader.SyncMode

	err := syncmode.UnmarshalText([]byte(cmd.String(aquaflags.SyncModeFlag.Name)))
	if err != nil {
		Fatalf("%v", err)
	}
	dl := downloader.New(syncmode, chainDb, new(event.TypeMux), chain, nil, nil)

	// Create a source peer to satisfy downloader requests from
	db, err := aquadb.NewLDBDatabase(cmd.Args().First(), int(cmd.Int(aquaflags.CacheFlag.Name)), 256)
	if err != nil {
		return err
	}
	hc, err := core.NewHeaderChain(ctx, db, chain.Config(), chain.Engine(), func() bool { return false })
	if err != nil {
		return err
	}
	peer := downloader.NewFakePeer("local", db, hc, dl)
	if err = dl.RegisterPeer("local", 63, peer); err != nil {
		return err
	}
	// Synchronise with the simulated peer
	start := time.Now()

	currentHeader := hc.CurrentHeader()
	if err = dl.Synchronise("local", currentHeader.Hash(), hc.GetTd(currentHeader.Hash(), currentHeader.Number.Uint64()), syncmode); err != nil {
		return err
	}
	for dl.Synchronising() {
		time.Sleep(10 * time.Millisecond)
	}
	fmt.Printf("Database copy done in %v\n", time.Since(start))

	// Compact the entire database to remove any sync overhead
	start = time.Now()
	fmt.Println("Compacting entire database...")
	if err = chainDb.(*aquadb.LDBDatabase).LDB().CompactRange(util.Range{}); err != nil {
		Fatalf("Compaction failed: %v", err)
	}
	fmt.Printf("Compaction done in %v.\n\n", time.Since(start))

	return nil
}

func removeDB(ctx context.Context, cmd *cli.Command) error {
	stack, _ := MakeConfigNode(ctx, cmd, gitCommit, clientIdentifier, mainctxs.MainCancelCause())

	name := "chaindata"
	// Ensure the database exists in the first place
	logger := log.New("database", name)

	dbdir := stack.ResolvePath(name)

	if dbdir2, err := os.Readlink(dbdir); err == nil {
		dbdir = dbdir2 // resolve symlink
	}

	// Confirm removal and execute
	fmt.Println(dbdir)
	confirm, err := console.Stdin.PromptConfirm("Remove this database?")
	switch {
	case err != nil:
		Fatalf("%v", err)
	case !confirm:
		logger.Warn("Database deletion aborted")
	default:
		start := time.Now()
		// if err := os.RemoveAll(dbdir); err != nil {
		// 	Fatalf("Failed to remove database: %v", err)
		// 	return nil
		// }
		if err := removeExtensions(dbdir, []string{".ldb", "LOG", ".bak", ".log", ""}, []string{"MANIFEST-", "CURRENT"}); err != nil {
			Fatalf("Failed to remove database: %v", err)
			return nil
		}
		logger.Info("Database successfully deleted", "elapsed", common.PrettyDuration(time.Since(start)))
	}

	return nil
}

func removeExtensions(dir string, ext []string, prefix []string) error {
	removeFunc := func(path string) error {
		log.Trace("removedb removing file", "path", path)
		return os.Remove(path)
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
Files:
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		for _, e := range ext {
			if strings.HasSuffix(file.Name(), e) {
				if err := removeFunc(dir + "/" + file.Name()); err != nil {
					return err
				}
				continue Files
			}
		}
		for _, prefix := range prefix {
			if strings.HasPrefix(file.Name(), prefix) {
				if err := removeFunc(dir + "/" + file.Name()); err != nil {
					return err
				}
				continue Files
			}
		}
		log.Info("Skipping", "file", file.Name())
	}
	return nil
}

func dump(ctx context.Context, cmd *cli.Command) error {
	stack := MakeFullNode(ctx, cmd)
	chain, chainDb := MakeChain(cmd, stack)
	for _, arg := range cmd.Args().Slice() {
		var block *types.Block
		if hashish(arg) {
			block = chain.GetBlockByHash(common.HexToHash(arg))
		} else {
			num, _ := strconv.Atoi(arg)
			block = chain.GetBlockByNumber(uint64(num))
		}
		if block == nil {
			fmt.Println("{}")
			Fatalf("block not found")
		} else {
			state, err := state.New(block.Root(), state.NewDatabase(chainDb))
			if err != nil {
				Fatalf("could not create new state: %v", err)
			}
			fmt.Printf("%s\n", state.Dump())
		}
	}
	chainDb.Close()
	return nil
}

// hashish returns true for strings that look like hashes.
func hashish(x string) bool {
	_, err := strconv.Atoi(x)
	return err != nil
}
