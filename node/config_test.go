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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gitlab.com/aquachain/aquachain/crypto"
	"gitlab.com/aquachain/aquachain/p2p"
)

var testp2p = &p2p.Config{ChainId: 10101}

// Tests that datadirs can be successfully created, be them manually configured
// ones or automatically generated temporary ones.
func TestDatadirCreation(t *testing.T) {
	// Create a temporary data dir and check that it can be used by a node
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("failed to create manual data dir: %v", err)
	}
	defer os.RemoveAll(dir)
	ctx := t.Context()
	closemain := func(error) {}
	if _, err := New(&Config{Name: "tester", DataDir: dir, P2P: testp2p, Context: ctx, CloseMain: closemain}); err != nil {
		t.Fatalf("failed to create stack with existing datadir: %v", err)
	}
	// Generate a long non-existing datadir path and check that it gets created by a node
	dir = filepath.Join(dir, "a", "b", "c", "d", "e", "f")
	if _, err := New(&Config{Name: "tester", DataDir: dir, P2P: testp2p, Context: ctx, CloseMain: closemain}); err != nil {
		t.Fatalf("failed to create stack with creatable datadir: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("freshly created datadir not accessible: %v", err)
	}
	// Verify that an impossible datadir fails creation
	file, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("failed to create temporary file: %v", err)
	}
	defer os.Remove(file.Name())

	dir = filepath.Join(file.Name(), "invalid/path")
	if _, err := New(&Config{Name: "tester", DataDir: dir, P2P: testp2p, Context: ctx, CloseMain: closemain}); err == nil {
		t.Fatalf("protocol stack created with an invalid datadir")
	}
}

// Tests that IPC paths are correctly resolved to valid endpoints of different
// platforms. Test 2 os for the randomized temporary path in the rare case of empty ("") datadir.
func TestIPCPathResolution(t *testing.T) {
	ephwant := "aquachain-*"
	var tests = []struct {
		DataDir  string
		IPCPath  string
		Windows  bool
		Endpoint string
	}{
		{"", "", false, ""},
		{"data", "", false, ""},
		{"", "aquachain.ipc", false, filepath.Join(os.TempDir(), ephwant, "aquachain.ipc")},
		{"data", "aquachain.ipc", false, "data/aquachain.ipc"},
		{"data", "./aquachain.ipc", false, "./aquachain.ipc"},
		{"data", "/aquachain.ipc", false, "/aquachain.ipc"},
		{"", "", true, ``},
		{"data", "", true, ``},
		{"", "aquachain.ipc", true, `\\.\pipe\aquachain.ipc`},
		{"data", "aquachain.ipc", true, `\\.\pipe\aquachain.ipc`},
		{"data", `\\.\pipe\aquachain.ipc`, true, `\\.\pipe\aquachain.ipc`},
	}
	for i, test := range tests {
		// Only run when platform/test match
		if (runtime.GOOS == "windows") == test.Windows {
			endpoint := (&Config{DataDir: test.DataDir, IPCPath: test.IPCPath, P2P: testp2p}).IPCEndpoint()
			istmp := strings.Contains(test.Endpoint, ephwant)
			if (endpoint != test.Endpoint) && (!istmp) {
				t.Errorf("test %d: IPC endpoint mismatchA: have %q, want %q", i, endpoint, test.Endpoint)
			}
			if istmp {
				var gotrand int
				if _, err := fmt.Sscanf(endpoint, strings.Replace(test.Endpoint, "*", "%d", 1), &gotrand); err != nil {
					t.Errorf("test %d: IPC endpoint mismatchB: have %q, want %q", i, endpoint, test.Endpoint)
				}
				fmt.Fprintf(os.Stderr, "test %d: IPC endpoint: %q gotrand=%d\n", i, endpoint, gotrand)
				if endpoint != strings.Replace(test.Endpoint, "*", fmt.Sprintf("%d", gotrand), 1) {
					t.Errorf("test %d: IPC endpoint mismatchC: have %q, want %q", i, endpoint, test.Endpoint)
				}

			}
		}
	}
}

// Tests that node keys can be correctly created, persisted, loaded and/or made
// ephemeral.
func TestNodeKeyPersistency(t *testing.T) {
	// Create a temporary folder and make sure no key is present
	dir, err := ioutil.TempDir("", "node-test")
	if err != nil {
		t.Fatalf("failed to create temporary data directory: %v", err)
	}
	defer os.RemoveAll(dir)

	keyfile := filepath.Join(dir, "unit-test", datadirPrivateKey)

	// Configure a node with a preset key and ensure it's not persisted
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate one-shot node key: %v", err)
	}
	config := &Config{Name: "unit-test", DataDir: dir, P2P: &p2p.Config{PrivateKey: key, ChainId: 10101}}
	config.NodeKey()
	if _, err := os.Stat(filepath.Join(keyfile)); err == nil {
		t.Fatalf("one-shot node key persisted to data directory")
	}

	// Configure a node with no preset key and ensure it is persisted this time
	config = &Config{Name: "unit-test", DataDir: dir, P2P: testp2p}
	config.NodeKey()
	if _, err := os.Stat(keyfile); err != nil {
		t.Fatalf("node key not persisted to data directory: %v", err)
	}
	if _, err = crypto.LoadECDSA(keyfile); err != nil {
		t.Fatalf("failed to load freshly persisted node key: %v", err)
	}
	blob1, err := ioutil.ReadFile(keyfile)
	if err != nil {
		t.Fatalf("failed to read freshly persisted node key: %v", err)
	}

	// Configure a new node and ensure the previously persisted key is loaded
	config = &Config{Name: "unit-test", DataDir: dir}
	config.NodeKey()
	blob2, err := ioutil.ReadFile(filepath.Join(keyfile))
	if err != nil {
		t.Fatalf("failed to read previously persisted node key: %v", err)
	}
	if !bytes.Equal(blob1, blob2) {
		t.Fatalf("persisted node key mismatch: have %x, want %x", blob2, blob1)
	}

	// Configure ephemeral node and ensure no key is dumped locally
	config = &Config{Name: "unit-test", DataDir: ""}
	config.NodeKey()
	if _, err := os.Stat(filepath.Join(".", "unit-test", datadirPrivateKey)); err == nil {
		t.Fatalf("ephemeral node key persisted to disk")
	}
}
