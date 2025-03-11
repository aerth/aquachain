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

// Package debug interfaces Go runtime debugging facilities.
// This package is mostly glue code making these facilities available
// through the CLI and RPC subsystem. If you want to use them from Go code,
// use package runtime instead.
package log

import (
	"io"
	"os"

	"github.com/mattn/go-colorable"
	// "gitlab.com/aquachain/aquachain/common/log"

	"gitlab.com/aquachain/aquachain/common/log/term"
	"gitlab.com/aquachain/aquachain/common/sense"
)

const (
	VmoduleGood  = "p2p/discover=3,aqua/*=4,consensus/*=9,core/*=9,rpc/*=9,node/*=9,opt/*=9,p2p/discover/udp.go=0"
	VmoduleGreat = "p2p/discover=3,aqua/*=9,consensus/*=9,core/*=9,rpc/*=9,node/*=9,opt/*=9"
)

var glogger *GlogHandler

// set global logger
func SetGlogger(l *GlogHandler) {
	if glogger == l {
		Warn("glogger already set!!!")
		return
	}
	glogger = l
	SetRootHandler(l)
}

func GetGlogger() *GlogHandler {
	return glogger
}

func Initglogger(callerinfo bool, verbosityLvl64 int64, alwayscolor, isjson bool) *GlogHandler {
	if glogger != nil {
		return glogger
	}
	isjson = isjson && sense.Getenv("JSONLOG") != "0" && sense.Getenv("JSONLOG") != "off" // allow override false from init, even if -jsonlog is set
	verbosityLvl := Lvl(verbosityLvl64)
	if verbosityLvl == 0 {
		verbosityLvl = LvlInfo
	}
	if verbosityLvl < 0 {
		verbosityLvl = 0

	}
	usecolor := alwayscolor || sense.Getenv("COLOR") == "1" || (term.IsTty(os.Stderr.Fd()) && sense.Getenv("TERM") != "dumb")
	output := io.Writer(os.Stderr)
	if usecolor {
		output = colorable.NewColorableStderr()
	}
	var form Format
	if isjson {
		form = JsonFormat()
	} else {
		form = TerminalFormat(usecolor)
	}
	h := StreamHandler(output, form)
	if isjson && callerinfo {
		h = CallerFileHandler(h)
	} else if callerinfo {
		PrintOrigins(true) // show line numbers
	}
	x := NewGlogHandler(h)
	x.Verbosity(verbosityLvl)
	// go Trace("new glogger", "verbosity", verbosityLvl, "color", alwayscolor, "json", isjson, "caller2", Caller(2))
	return x
}
