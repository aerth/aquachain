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

// bootnode runs a bootstrap node for the AquaChain Discovery Protocol.
package main

import (
	"crypto/ecdsa"
	"flag"
	"fmt"
	"net"
	"os"

	"gitlab.com/aquachain/aquachain/cmd/utils"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/crypto"
	"gitlab.com/aquachain/aquachain/p2p/discover"
	"gitlab.com/aquachain/aquachain/p2p/nat"
	"gitlab.com/aquachain/aquachain/p2p/netutil"
)

func main() {
	var (
		listenAddr  = flag.String("addr", ":21000", "listen address")
		genKey      = flag.String("genkey", "", "generate a node key")
		writeAddr   = flag.Bool("writeaddress", false, "write out the node's pubkey hash and quit")
		nodeKeyFile = flag.String("nodekey", "", "private key filename")
		nodeKeyHex  = flag.String("nodekeyhex", "", "private key as hex (for testing)")
		natdesc     = flag.String("nat", "none", "port mapping mechanism (any|none|upnp|pmp|extip:<IP>)")
		netrestrict = flag.String("netrestrict", "", "restrict network communication to the given IP networks (CIDR masks)")
		runv5       = flag.Bool("v5", false, "run a v5 topic discovery bootnode")
		verbosity   = flag.Int("verbosity", int(log.LvlInfo), "log verbosity (0-9)")
		vmodule     = flag.String("vmodule", "", "log verbosity pattern")
		chainid     = flag.Uint64("chainid", 61717561, "chain id for p2p communication (set to 3 for ethereum-like)")
		nodeKey     *ecdsa.PrivateKey
		err         error
	)
	flag.Parse()

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.Lvl(*verbosity))
	glogger.Vmodule(*vmodule)
	log.Root().SetHandler(glogger)

	natm, err := nat.Parse(*natdesc)
	if err != nil {
		utils.Fatalf("-nat: %v", err)
	}
	switch {
	case *genKey != "":
		nodeKey, err = crypto.GenerateKey()
		if err != nil {
			utils.Fatalf("could not generate key: %v", err)
		}
		if err = crypto.SaveECDSA(*genKey, nodeKey); err != nil {
			utils.Fatalf("%v", err)
		}
		return
	case *nodeKeyFile == "" && *nodeKeyHex == "":
		utils.Fatalf("Use -nodekey or -nodekeyhex to specify a private key")
	case *nodeKeyFile != "" && *nodeKeyHex != "":
		utils.Fatalf("Options -nodekey and -nodekeyhex are mutually exclusive")
	case *nodeKeyFile != "":
		if nodeKey, err = crypto.LoadECDSA(*nodeKeyFile); err != nil {
			utils.Fatalf("-nodekey: %v", err)
		}
	case *nodeKeyHex != "":
		if nodeKey, err = crypto.HexToECDSA(*nodeKeyHex); err != nil {
			utils.Fatalf("-nodekeyhex: %v", err)
		}
	}

	if *writeAddr {
		fmt.Printf("%v\n", discover.PubkeyID(&nodeKey.PublicKey))
		os.Exit(0)
	}

	var restrictList *netutil.Netlist
	if *netrestrict != "" {
		restrictList, err = netutil.ParseNetlist(*netrestrict)
		if err != nil {
			utils.Fatalf("-netrestrict: %v", err)
		}
	}

	addr, err := net.ResolveUDPAddr("udp", *listenAddr)
	if err != nil {
		utils.Fatalf("-ResolveUDPAddr: %v", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		utils.Fatalf("-ListenUDP: %v", err)
	}

	realaddr := conn.LocalAddr().(*net.UDPAddr)
	if natm != nil {
		if !realaddr.IP.IsLoopback() {
			go nat.Map(natm, nil, "udp", realaddr.Port, realaddr.Port, "aquachain discovery")
		}
		// TODO: react to external IP changes over time.
		if ext, err := natm.ExternalIP(); err == nil {
			realaddr = &net.UDPAddr{IP: ext, Port: realaddr.Port}
		}
	}

	if *runv5 {
		utils.Fatalf("%s", "discover v5 support dropped")
	} else {
		cfg := discover.Config{
			PrivateKey:   nodeKey,
			AnnounceAddr: realaddr,
			NetRestrict:  restrictList,
			ChainId:      *chainid,
		}
		if _, err := discover.ListenUDP(conn, cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	select {}
}
