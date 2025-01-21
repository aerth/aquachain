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

package node

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"gitlab.com/aquachain/aquachain/p2p"
	"gitlab.com/aquachain/aquachain/p2p/nat"
)

const (
	DefaultHTTPHost = "127.0.0.1" // Default host interface for the HTTP RPC server
	DefaultHTTPPort = 8543        // Default TCP port for the HTTP RPC server
	DefaultWSHost   = "127.0.0.1" // Default host interface for the websocket RPC server
	DefaultWSPort   = 8544        // Default TCP port for the websocket RPC server
)

// DefaultConfig contains reasonable default settings.
var DefaultConfig = Config{
	DataDir:     DefaultDataDir(),
	HTTPPort:    DefaultHTTPPort,
	HTTPModules: []string{"aqua", "eth", "net", "web3"},
	WSPort:      DefaultWSPort,
	WSModules:   []string{"aqua", "eth", "net", "web3"},
	P2P: p2p.Config{
		ListenAddr: "0.0.0.0:21303", // ipv4 only
		MaxPeers:   20,
		NAT:        nat.Any(),
	},
}

// DefaultDataDir is the default data directory to use for the databases and other
// persistence requirements.
func DefaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "Aquachain")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "Aquachain")
		} else {
			return filepath.Join(home, ".aquachain")
		}
	}
	// As we cannot guess a stable location, return empty and handle later
	return ""
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
