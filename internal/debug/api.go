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
package debug

import (
	"errors"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-colorable"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/log/term"
	"gitlab.com/aquachain/aquachain/common/sense"
)

// Handler is the global debugging handler.
var Handler = new(HandlerT)

// HandlerT implements the debugging API.
// Do not create values of this type, use the one
// in the Handler variable instead.
type HandlerT struct {
	mu        sync.Mutex
	cpuW      io.WriteCloser
	cpuFile   string
	traceW    io.WriteCloser
	traceFile string
}

const (
	VmoduleGood  = "p2p/discover=3,aqua/*=4,consensus/*=9,core/*=9,rpc/*=9,node/*=9,opt/*=9,p2p/discover/udp.go=0"
	VmoduleGreat = "p2p/discover=3,aqua/*=9,consensus/*=9,core/*=9,rpc/*=9,node/*=9,opt/*=9"
)

var glogger *log.GlogHandler

// set global logger
func SetGlogger(l *log.GlogHandler) {
	if glogger == l {
		log.Warn("glogger already set!!!")
		return
	}
	glogger = l
	log.SetRootHandler(l)
}

func Initglogger(callerinfo bool, verbosityLvl64 int64, alwayscolor, isjson bool) *log.GlogHandler {
	if glogger != nil {
		return glogger
	}
	isjson = isjson && sense.Getenv("JSONLOG") != "0" && sense.Getenv("JSONLOG") != "off" // allow override false from init, even if -jsonlog is set
	verbosityLvl := log.Lvl(verbosityLvl64)
	if verbosityLvl == 0 {
		verbosityLvl = log.LvlInfo
	}
	if verbosityLvl < 0 {
		verbosityLvl = 0

	}
	usecolor := alwayscolor || sense.Getenv("COLOR") == "1" || (term.IsTty(os.Stderr.Fd()) && sense.Getenv("TERM") != "dumb")
	output := io.Writer(os.Stderr)
	if usecolor {
		output = colorable.NewColorableStderr()
	}
	var form log.Format
	if isjson {
		form = log.JsonFormat()
	} else {
		form = log.TerminalFormat(usecolor)
	}
	h := log.StreamHandler(output, form)
	if isjson && callerinfo {
		h = log.CallerFileHandler(h)
	} else if callerinfo {
		log.PrintOrigins(true) // show line numbers
	}
	x := log.NewGlogHandler(h)
	x.Verbosity(verbosityLvl)
	// go log.Trace("new glogger", "verbosity", verbosityLvl, "color", alwayscolor, "json", isjson, "caller2", log.Caller(2))
	return x
}

// Verbosity sets the log verbosity ceiling. The verbosity of individual packages
// and source files can be raised using Vmodule.
func (*HandlerT) Verbosity(level int) {
	glogger.Verbosity(log.Lvl(level))
}

func wrapVmodule(pattern string) string {
	if pattern == "good" {
		pattern = VmoduleGood
	} else if pattern == "great" {
		pattern = VmoduleGreat
	}
	return pattern
}

// Vmodule sets the log verbosity pattern. See package log for details on the
// pattern syntax.
func (*HandlerT) Vmodule(pattern string) error {
	return glogger.Vmodule(wrapVmodule(pattern))
}

// BacktraceAt sets the log backtrace location. See package log for details on
// the pattern syntax.
func (*HandlerT) BacktraceAt(location string) error {
	return glogger.BacktraceAt(location)
}

// MemStats returns detailed runtime memory statistics.
func (*HandlerT) MemStats() *runtime.MemStats {
	s := new(runtime.MemStats)
	runtime.ReadMemStats(s)
	return s
}

// GcStats returns GC statistics.
func (*HandlerT) GcStats() *debug.GCStats {
	s := new(debug.GCStats)
	debug.ReadGCStats(s)
	return s
}

// CpuProfile turns on CPU profiling for nsec seconds and writes
// profile data to file.
func (h *HandlerT) CpuProfile(file string, nsec uint) error {
	if err := h.StartCPUProfile(file); err != nil {
		return err
	}
	time.Sleep(time.Duration(nsec) * time.Second)
	h.StopCPUProfile()
	return nil
}

// StartCPUProfile turns on CPU profiling, writing to the given file.
func (h *HandlerT) StartCPUProfile(file string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cpuW != nil {
		return errors.New("CPU profiling already in progress")
	}
	f, err := os.Create(expandHome(file))
	if err != nil {
		return err
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		return err
	}
	h.cpuW = f
	h.cpuFile = file
	log.Info("CPU profiling started", "dump", h.cpuFile)
	return nil
}

// StopCPUProfile stops an ongoing CPU profile.
func (h *HandlerT) StopCPUProfile() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	pprof.StopCPUProfile()
	if h.cpuW == nil {
		return errors.New("CPU profiling not in progress")
	}
	log.Info("Done writing CPU profile", "dump", h.cpuFile)
	h.cpuW.Close()
	h.cpuW = nil
	h.cpuFile = ""
	return nil
}

// GoTrace turns on tracing for nsec seconds and writes
// trace data to file.
func (h *HandlerT) GoTrace(file string, nsec uint) error {
	if err := h.StartGoTrace(file); err != nil {
		return err
	}
	time.Sleep(time.Duration(nsec) * time.Second)
	h.StopGoTrace()
	return nil
}

// BlockProfile turns on CPU profiling for nsec seconds and writes
// profile data to file. It uses a profile rate of 1 for most accurate
// information. If a different rate is desired, set the rate
// and write the profile manually.
func (*HandlerT) BlockProfile(file string, nsec uint) error {
	runtime.SetBlockProfileRate(1)
	time.Sleep(time.Duration(nsec) * time.Second)
	defer runtime.SetBlockProfileRate(0)
	return writeProfile("block", file)
}

// SetBlockProfileRate sets the rate of goroutine block profile data collection.
// rate 0 disables block profiling.
func (*HandlerT) SetBlockProfileRate(rate int) {
	runtime.SetBlockProfileRate(rate)
}

// WriteBlockProfile writes a goroutine blocking profile to the given file.
func (*HandlerT) WriteBlockProfile(file string) error {
	return writeProfile("block", file)
}

// WriteMemProfile writes an allocation profile to the given file.
// Note that the profiling rate cannot be set through the API,
// it must be set on the command line.
func (*HandlerT) WriteMemProfile(file string) error {
	return writeProfile("heap", file)
}

// Stacks returns a printed representation of the stacks of all goroutines.
func (*HandlerT) Stacks() string {
	buf := make([]byte, 1024*1024)
	buf = buf[:runtime.Stack(buf, true)]
	return string(buf)
}

// FreeOSMemory returns unused memory to the OS.
func (*HandlerT) FreeOSMemory() {
	debug.FreeOSMemory()
}

// SetGCPercent sets the garbage collection target percentage. It returns the previous
// setting. A negative value disables GC.
func (*HandlerT) SetGCPercent(v int) int {
	return debug.SetGCPercent(v)
}

func writeProfile(name, file string) error {
	p := pprof.Lookup(name)
	log.Info("Writing profile records", "count", p.Count(), "type", name, "dump", file)
	f, err := os.Create(expandHome(file))
	if err != nil {
		return err
	}
	defer f.Close()
	return p.WriteTo(f, 0)
}

// expands home directory in file paths.
// ~someuser/tmp will not be expanded.
func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		home := sense.Getenv("HOME")
		if home == "" {
			if usr, err := user.Current(); err == nil {
				home = usr.HomeDir
			}
		}
		if home != "" {
			p = home + p[1:]
		}
	}
	return filepath.Clean(p)
}
