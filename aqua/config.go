// Copyright 2018 The aquachain Authors
// This file is part of the aquachain library.
//
// The aquachain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The aquachain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the aquachain library. If not, see <http://www.gnu.org/licenses/>.

package aqua

import (
	"math/big"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	"gitlab.com/aquachain/aquachain/aqua/downloader"
	"gitlab.com/aquachain/aquachain/aqua/gasprice"
	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/alerts"
	"gitlab.com/aquachain/aquachain/common/hexutil"
	"gitlab.com/aquachain/aquachain/consensus/aquahash"
	"gitlab.com/aquachain/aquachain/core"
)

// DefaultConfig contains default settings for use on the Aquachain main net.
var DefaultConfig = &Config{
	SyncMode: downloader.FullSync,
	Aquahash: aquahash.Config{
		CacheDir:       "aquahash",
		CachesInMem:    1,
		CachesOnDisk:   0,
		DatasetsInMem:  0,
		DatasetsOnDisk: 0,
	},
	ChainId:       61717561,
	DatabaseCache: 768,
	TrieCache:     256,
	TrieTimeout:   5 * time.Minute,
	GasPrice:      big.NewInt(10000000), // 0.01 gwei
	NoPruning:     true,
	TxPool:        core.DefaultTxPoolConfig,
	GPO: gasprice.Config{
		Blocks:     20,
		Percentile: 60,
	},
}

func init() {
	home := os.Getenv("HOME")
	if home == "" {
		if user, err := user.Current(); err == nil {
			home = user.HomeDir
		}
	}
	if runtime.GOOS == "windows" {
		DefaultConfig.Aquahash.DatasetDir = filepath.Join(home, "AppData", "Aquahash")
	} else {
		DefaultConfig.Aquahash.DatasetDir = filepath.Join(home, ".aquahash")
	}
}

//go:generate gencodec -type Config -field-override ConfigMarshaling -formats toml -out gen_config.go

type Config struct {
	// The genesis block, which is inserted if the database is empty.
	// If nil, the Aquachain main net block is used.
	Genesis *core.Genesis `toml:",omitempty"`

	// Protocol options
	ChainId   uint64 // Network ID to use for selecting peers to connect to
	SyncMode  downloader.SyncMode
	NoPruning bool `toml:"NoPruning"`

	// Database options
	SkipBcVersionCheck bool `toml:"-"`
	DatabaseHandles    int  `toml:"-"`
	DatabaseCache      int
	TrieCache          int
	TrieTimeout        time.Duration

	// Mining-related options
	Aquabase     common.Address `toml:",omitempty"`
	MinerThreads int            `toml:",omitempty"`
	ExtraData    hexutil.Bytes  `toml:",omitempty"`
	GasPrice     *big.Int

	// Aquahash options
	Aquahash aquahash.Config

	// Transaction pool options
	TxPool core.TxPoolConfig

	// Gas Price Oracle options
	GPO gasprice.Config

	// Enables tracking of SHA3 preimages in the VM
	EnablePreimageRecording bool

	// Miscellaneous options

	JavascriptDirectory string `toml:"-"` // for console/attach only

	// Alert options
	Alerts      alerts.AlertConfig `toml:",omitempty"`
	p2pnodename string             `toml:"-"`
}

// ConfigMarshaling must be changed if the Config struct changes.
type ConfigMarshaling struct {
	ExtraData hexutil.Bytes
}
