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

// aquastrat command is an unoptimized stratum miner for aquachain
package main

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	cli "github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/consensus/aquahash"
	"gitlab.com/aquachain/aquachain/consensus/lightvalid"
	"gitlab.com/aquachain/aquachain/core/types"
	"gitlab.com/aquachain/aquachain/internal/debug"
	"gitlab.com/aquachain/aquachain/node"
	"gitlab.com/aquachain/aquachain/opt/aquaclient"
	"gitlab.com/aquachain/aquachain/params"
	"gitlab.com/aquachain/aquachain/rlp"
	rpc "gitlab.com/aquachain/aquachain/rpc/rpcclient"
	"gitlab.com/aquachain/aquachain/subcommands"
)

var gitCommit = ""

// valid block #1 using -testnet2
var header1 = &types.Header{
	Difficulty: big.NewInt(4096),
	Extra:      []byte{0xd4, 0x83, 0x01, 0x07, 0x04, 0x89, 0x61, 0x71, 0x75, 0x61, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x85, 0x6c, 0x69, 0x6e, 0x75, 0x78},
	GasLimit:   4704588,
	GasUsed:    0,
	// Hash: "0x73851a4d607acd8341cf415beeed9c8b8c803e1e835cb45080f6af7a2127e807",
	Coinbase:    common.HexToAddress("0xcf8e5ba37426404bef34c3ca4fa2d2ed9be41e58"),
	MixDigest:   common.Hash{},
	Nonce:       types.BlockNonce{0x70, 0xc2, 0xdd, 0x45, 0xa3, 0x10, 0x17, 0x35},
	Number:      big.NewInt(1),
	ParentHash:  common.HexToHash("0xde434983d3ada19cd43c44d8ad5511bad01ed12b3cc9a99b1717449a245120df"),
	ReceiptHash: common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
	UncleHash:   common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
	Root:        common.HexToHash("0x194b1927f77b77161b58fed1184990d8f7b345fabf8ef8706ee865a844f73bc3"),
	Time:        big.NewInt(1536181711),
	TxHash:      common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
	Version:     2,
}

var big1 = big.NewInt(1)
var Config *params.ChainConfig = params.Testnet2ChainConfig

func main() {
	ctx := context.Background()

	var (
		app = subcommands.NewApp("aquastrat", gitCommit, "usage")
	)
	app.Action = loopit
	_ = filepath.Join
	app.Flags = append(debug.Flags, []cli.Flag{
		&cli.StringFlag{
			Value:    filepath.Join(node.DefaultDatadirByChain(Config), "aquachain.ipc"),
			Name:     "rpc",
			Usage:    "path or url to rpc",
			Category: "AQUASTRAT",
		},
		&cli.StringFlag{
			Value:    "",
			Name:     "coinbase",
			Usage:    "address for mining rewards",
			Category: "AQUASTRAT",
		},
	}...)

	if err := app.Run(ctx, os.Args); err != nil {
		fmt.Println("fatal:", err)
	}
}

func loopit(ctx context.Context, cmd *cli.Command) error {
	for {
		if err := runit(ctx, cmd); err != nil {
			fmt.Println(err)
			return err
		}
	}
}
func runit(ctx context.Context, cmd *cli.Command) error {
	coinbase := cmd.String("coinbase")
	if coinbase == "" || !strings.HasPrefix(coinbase, "0x") || len(coinbase) != 42 {
		return fmt.Errorf("cant mine with no -coinbase flag, or invalid: len: %v, coinbase: %q", len(coinbase), coinbase)
	}
	coinbaseAddr := common.HexToAddress(coinbase)

	rpcclient, err := getclient(ctx, cmd)
	if err != nil {
		return err
	}
	aqua := aquaclient.NewClient(rpcclient)
	block1, err := aqua.BlockByNumber(ctx, big1)
	if err != nil {
		fmt.Println("blockbynumber")
		return err
	}

	// to get genesis hash, we can't grab block zero and Hash()
	// because we dont know the chainconfig which tells us
	// the version to use for hashing.
	genesisHash := block1.ParentHash()
	switch genesisHash {
	case params.MainnetGenesisHash:
		Config = params.MainnetChainConfig
	case params.TestnetGenesisHash:
		Config = params.TestnetChainConfig
	default:
		Config = params.Testnet2ChainConfig
	}
	parent, err := aqua.BlockByNumber(ctx, nil)
	if err != nil {
		fmt.Println("blockbynumber")
		return err
	}
	var encoded []byte
	// first block is on the house (testnet2 only)
	if Config == params.Testnet2ChainConfig && parent.Number().Uint64() == 0 {
		parent.SetVersion(Config.GetBlockVersion(parent.Number()))
		block1 := types.NewBlock(header1, nil, nil, nil)
		encoded, err = rlp.EncodeToBytes(&block1)
		if err != nil {
			return err
		}
	}
	if len(encoded) != 0 {
		encoded, err = aqua.GetBlockTemplate(ctx, coinbaseAddr)
	}
	if err != nil {
		println("gbt")
		return err
	}
	var bt = new(types.Block)
	if err := rlp.DecodeBytes(encoded, bt); err != nil {
		println("getblocktemplate rlp decode error", err.Error())
		return err
	}

	// modify block
	bt.SetVersion(Config.GetBlockVersion(bt.Number()))
	fmt.Println("mining:")
	fmt.Println(bt)
	encoded, err = mine(Config, parent.Header(), bt)
	if err != nil {
		return err
	}
	if encoded == nil {
		return fmt.Errorf("failed to encoded block to rlp")
	}
	if !aqua.SubmitBlock(ctx, encoded) {
		fmt.Println("failed")
		return fmt.Errorf("failed")
	} else {
		fmt.Println("success")
	}
	return nil

}

func mine(cfg *params.ChainConfig, parent *types.Header, block *types.Block) ([]byte, error) {
	validator := lightvalid.New()
	rand.Seed(time.Now().UnixNano())
	nonce := uint64(0)
	nonce = rand.Uint64()
	hdr := block.Header()
	fmt.Println("mining algo:", hdr.Version)
	fmt.Printf("#%v, by %x\ndiff: %s\ntx: %s\n", hdr.Number, hdr.Coinbase, hdr.Difficulty, block.Transactions())
	fmt.Printf("starting from nonce: %v\n", nonce)
	second := time.Tick(10 * time.Second)
	fps := uint64(0)
	for {
		select {
		case <-second:
			// not necessary, but impossible with getwork()
			hdr.Time = big.NewInt(time.Now().Unix())
			hdr.Difficulty = aquahash.CalcDifficulty(cfg, hdr.Time.Uint64(), parent, nil)
			fmt.Printf("%s %v h/s\n", hdr.Time, fps/uint64(10))
			fps = 0
		default:
			nonce++
			fps++
			hdr.Nonce = types.EncodeNonce(nonce)
			block = block.WithSeal(hdr)
			block = types.NewBlock(hdr, block.Transactions(), block.Uncles(), []*types.Receipt{})
			if err := validator.VerifyWithError(block); err != nil {
				if err != lightvalid.ErrPOW {
					fmt.Println("error:", err)
				}
				continue
			}
			println("found nonce, encoding block", block.String())
			b, err := rlp.EncodeToBytes(&block)
			if err != nil {
				return nil, err
			}
			fmt.Println(b)
			return b, nil
		}
	}
}

func getclient(ctx context.Context, cmd *cli.Command) (*rpc.Client, error) {
	if strings.HasPrefix(cmd.String("rpc"), "http") {
		return rpc.DialHTTP(cmd.String("rpc"))
	} else {
		return rpc.DialIPC(ctx, cmd.String("rpc"))
	}
}
