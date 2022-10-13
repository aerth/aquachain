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

package main

import (
	"crypto/rand"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"gitlab.com/aquachain/aquachain/params"
)

const (
	ipcAPIs      = "admin:1.0 debug:1.0 aqua:1.0 miner:1.0 net:1.0 personal:1.0 rpc:1.0 shh:1.0 txpool:1.0 web3:1.0"
	httpAPIs     = "aqua:1.0 net:1.0 rpc:1.0 web3:1.0"
	testCoinbase = "0x000000000000000000000000000000000000dEaD"
)

// Tests that a node embedded within a console can be started up properly and
// then terminated by closing the input stream.
func TestConsoleWelcome(t *testing.T) {
	t.Skip()

	// Start a aquachain console, make sure it's cleaned up and terminate the console
	aquachain := runAquachain(t,
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none",
		"--aquabase", testCoinbase, "--shh",
		"console")

	// Gather all the infos the welcome message needs to contain
	aquachain.SetTemplateFunc("goos", func() string { return runtime.GOOS })
	aquachain.SetTemplateFunc("goarch", func() string { return runtime.GOARCH })
	aquachain.SetTemplateFunc("gover", runtime.Version)
	aquachain.SetTemplateFunc("gethver", func() string { return params.Version })
	aquachain.SetTemplateFunc("niltime", func() string { return time.Unix(0, 0).Format(time.RFC1123) })
	aquachain.SetTemplateFunc("apis", func() string { return ipcAPIs })

	// Verify the actual welcome message to the required template
	aquachain.Expect(`
Welcome to the Aquachain JavaScript console!

instance: Aquachain/v{{gethver}}/{{goos}}-{{goarch}}/{{gover}}
coinbase: {{.Aquabase}}
at block: 0 ({{niltime}})
 datadir: {{.Datadir}}
 modules: {{apis}}

> {{.InputLine "exit"}}
`)
	aquachain.ExpectExit()
}

// Tests that a console can be attached to a running node via various means.
func TestIPCAttachWelcome(t *testing.T) {
	t.Skip()
	// Configure the instance for IPC attachement
	var ipc string
	if runtime.GOOS == "windows" {
		ipc = `\\.\pipe\aquachain` + strconv.Itoa(trulyRandInt(100000, 999999))
	} else {
		ws := tmpdir(t)
		defer os.RemoveAll(ws)
		ipc = filepath.Join(ws, "aquachain.ipc")
	}
	// Note: we need --shh because testAttachWelcome checks for default
	// list of ipc modules and shh is included there.
	aquachain := runAquachain(t,
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none",
		"--aquabase", testCoinbase, "--shh", "--ipcpath", ipc)

	time.Sleep(2 * time.Second) // Simple way to wait for the RPC endpoint to open
	testAttachWelcome(t, aquachain, "ipc:"+ipc, ipcAPIs)

	aquachain.Interrupt()
	aquachain.ExpectExit()
}

func TestHTTPAttachWelcome(t *testing.T) {
	t.Skip()
	port := strconv.Itoa(trulyRandInt(1024, 65536)) // Yeah, sometimes this will fail, sorry :P
	aquachain := runAquachain(t,
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none",
		"--aquabase", testCoinbase, "--rpc", "--rpcport", port)

	time.Sleep(2 * time.Second) // Simple way to wait for the RPC endpoint to open
	testAttachWelcome(t, aquachain, "http://localhost:"+port, httpAPIs)

	aquachain.Interrupt()
	aquachain.ExpectExit()
}

func TestWSAttachWelcome(t *testing.T) {
	t.Skip()
	port := strconv.Itoa(trulyRandInt(1024, 65536)) // Yeah, sometimes this will fail, sorry :P

	aquachain := runAquachain(t,
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none",
		"--aquabase", testCoinbase, "--ws", "--wsport", port)

	time.Sleep(2 * time.Second) // Simple way to wait for the RPC endpoint to open
	testAttachWelcome(t, aquachain, "ws://localhost:"+port, httpAPIs)

	aquachain.Interrupt()
	aquachain.ExpectExit()
}

func testAttachWelcome(t *testing.T, aquachain *testgeth, endpoint, apis string) {
	// Attach to a running aquachain note and terminate immediately
	attach := runAquachain(t, "attach", endpoint)
	defer attach.ExpectExit()
	attach.CloseStdin()

	// Gather all the infos the welcome message needs to contain
	attach.SetTemplateFunc("goos", func() string { return runtime.GOOS })
	attach.SetTemplateFunc("goarch", func() string { return runtime.GOARCH })
	attach.SetTemplateFunc("gover", runtime.Version)
	attach.SetTemplateFunc("gethver", func() string { return params.Version })
	attach.SetTemplateFunc("aquabase", func() string { return aquachain.Aquabase })
	attach.SetTemplateFunc("niltime", func() string { return time.Unix(0, 0).Format(time.RFC1123) })
	attach.SetTemplateFunc("ipc", func() bool { return strings.HasPrefix(endpoint, "ipc") })
	attach.SetTemplateFunc("datadir", func() string { return aquachain.Datadir })
	attach.SetTemplateFunc("apis", func() string { return apis })

	// Verify the actual welcome message to the required template
	attach.Expect(`
Welcome to the Aquachain JavaScript console!

instance: Aquachain/v{{gethver}}/{{goos}}-{{goarch}}/{{gover}}
coinbase: {{aquabase}}
at block: 0 ({{niltime}}){{if ipc}}
 datadir: {{datadir}}{{end}}
 modules: {{apis}}

> {{.InputLine "exit" }}
`)
	attach.ExpectExit()
}

// trulyRandInt generates a crypto random integer used by the console tests to
// not clash network ports with other tests running cocurrently.
func trulyRandInt(lo, hi int) int {
	num, _ := rand.Int(rand.Reader, big.NewInt(int64(hi-lo)))
	return int(num.Int64()) + lo
}
