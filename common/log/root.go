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

package log

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/go-stack/stack"

	"gitlab.com/aquachain/aquachain/common/sense"
)

var (
	StderrHandler         = newRootHandler()
	root          *logger = newRoot(StderrHandler)
	// old: StdoutHandler = StreamHandler(os.Stdout, LogfmtFormat())
	// old: StderrHandler = StreamHandler(os.Stderr, LogfmtFormat())
)

// func init() {
// 	root.SetHandler(CallerFileHandler(StderrHandler))
// }

// New returns a new logger with the given context.
// New is a convenient alias for Root().New
func New(ctx ...interface{}) LoggerI {
	logger := root.New(ctx...)
	if debuglog {
		root.Warn("New Logger Created", append([]any{"log_creator", stack.Caller(1)}, ctx...)...)
	}
	return logger
}

func SetRootHandler(h Handler) {
	if root == nil {
		root = newRoot(h)
		return
	}
	root.SetHandler(h)
}

func SetRoot(x *logger) {
	root = x
	Info("root logger set", "log", fmt.Sprintf("%T", root))
}

// Root returns the root logger
func Root() *logger {
	return root
}

// The following functions bypass the exported logger methods (logger.Debug,
// etc.) to keep the call depth the same for all paths to logger.write so
// runtime.Caller(2) always refers to the call site in client code.

// Trace is a convenient alias for Root().Trace
func Trace(msg string, ctx ...interface{}) {
	Root().write(msg, LvlTrace, ctx)
}

// Debug is a convenient alias for Root().Debug
func Debug(msg string, ctx ...interface{}) {
	Root().write(msg, LvlDebug, ctx)
}

// atomic map TODO: *does this func copy the atomic.Value ?
var slowmap atomic.Value = func() atomic.Value {
	m := make(map[string]time.Time)
	var a atomic.Value
	a.Store(m)
	return a
}()

var oktime = time.Second * 5

// DebugSlow is flaky
func DebugSlow(msg string, ctx ...any) {
	caller := stack.Caller(1).Frame().Function
	m, _ := slowmap.Load().(map[string]time.Time)
	got, ok := m[caller]
	timesince := time.Since(got)
	if ok && timesince < 0 {
		return
	}
	if !ok || timesince > oktime {
		Root().write(msg, LvlDebug, append(ctx, "slowcaller", caller))
		m[caller] = time.Now() // ok = true
		slowmap.Store(m)
		Root().write(msg, LvlDebug, ctx)
	}
}

// Info is a convenient alias for Root().Info
func Info(msg string, ctx ...interface{}) {
	Root().write(msg, LvlInfo, ctx)
}

// Warn is a convenient alias for Root().Warn
func Warn(msg string, ctx ...interface{}) {
	Root().write(msg, LvlWarn, ctx)
}

func Noop(msg string, ctx ...interface{}) {
	// cool
}

// Error is a convenient alias for Root().Error
func Error(msg string, ctx ...interface{}) {
	Root().write(msg, LvlError, ctx)
}

// Crit is a convenient alias for Root().Crit
func Crit(msg string, ctx ...interface{}) {
	cancelcausefunc(fmt.Errorf("%s", msg)) // just incase it wasnt already cancelled
	if root != nil {
		root.write(msg, LvlCrit, ctx)
	} else {
		println("fatal: ", msg)
	}
	if sense.EnvBool("DEBUG") {
		println("stack trace requsted (DEBUG=1)...")
		debug.PrintStack()
		time.Sleep(time.Second)
	}
	os.Exit(1)
}

func GracefulShutdownf(causef string, args ...any) {
	for i, a := range args {
		if err, ok := a.(error); ok {
			args[i] = TranslateFatalError(err)
		}
	}
	GracefulShutdown(Errorf(causef, args...))
}

// store mainctx here

var mainctx context.Context
var alreadyhandled bool = true // sane default for use as a library
var cancelcausefunc context.CancelCauseFunc = func(cause error) {
	Root().Warn("main shutdown function not registered, not exiting", "cause", cause)
}

// RegisterCancelCause registers a CancelCause func to be called when a fatal error is logged (GracefulShutdown, Crit, etc)
func RegisterCancelCause(ctx context.Context, f context.CancelCauseFunc) {
	if alreadyhandled {
		panic("already registered")
	}
	mainctx = ctx
	cancelcausefunc = f
}

func HandleSignals() {
	alreadyhandled = false // we are in cmd/aquachain etc using mainctxs.CancelCause and log.GracefulShutdown is real
}

func TranslateFatalError(err error) error {
	if mainctx != nil && context.Cause(mainctx) == context.Canceled && err == context.Canceled {
		err = fmt.Errorf("received interrupt signal")
	}
	return err
}

// GracefulShutdown (when configured) returns immediately and initiates a graceful shutdown of the
// entire stack (chain, rpc, p2p, etc) and logs the cause of the shutdown.
//
// After 10 seconds the process should panic
func GracefulShutdown(cause error) {
	if root != nil {
		root.write("graceful shutdown initiated", LvlCrit, []any{"cause", cause, "caller1", Caller(1), "caller2", Caller(2)})
	} else {
		println("fatal: ", cause.Error())
		if sense.EnvBool("DEBUG") {
			println("stack trace requsted (DEBUG=1)...")
			debug.PrintStack()
		}
	}
	if alreadyhandled || testloghandler != nil { // or if testing
		Warn("graceful shutdown requested", "cause", cause)
		return
	}
	cancelcausefunc(cause) // this should shutdown the stack
	go notifyGracefulShutdown(cause)
}

func notifyGracefulShutdown(cause error) {
	for i := 10; i > 0; i-- {
		time.Sleep(time.Second) // program should quit before this first sleep
		root.write("graceful shutdown initiated", LvlCrit, []any{"cause", cause, "seconds", i})
	}
	// panic big
	debug.SetTraceback("all")
	panic(cause)
}

var Caller = stack.Caller

// Errorf can be swapped for a caller-aware version (TODO)
var Errorf = fmt.Errorf
