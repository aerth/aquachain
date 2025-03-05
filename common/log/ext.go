package log

import (
	"fmt"
	"os"

	"gitlab.com/aquachain/aquachain/common/sense"
)

var NoSync = !sense.EnvBoolDisabled("NO_LOGSYNC")

func init() {
	go func() {
		Warn("log nosync:", "noSync", NoSync)
	}()
}

var PrintfDefaultLevel = LvlInfo

func (l *logger) Printf(msg string, stuff ...any) {
	msg = fmt.Sprintf(msg, stuff...)
	l.write(msg, PrintfDefaultLevel, []any{"todo", "oldlog"}) // add todo to log to know we should migrate it
}

func Printf(msg string, stuff ...any) {
	root.Printf(msg, stuff...)
}

var testloghandler Handler

// for test packages to call in init
func ResetForTesting() {
	if testloghandler != nil {
		return
	}
	lvl := LvlWarn
	if x := os.Getenv("TESTLOGLVL"); x != "" && x != "0" { // so TESTLOGLVL=0 is the same as not setting it (0=crit, which is silent)
		Info("setting custom TESTLOGLVL log level", "loglevel", x)
		lvl = MustParseLevel(x)
	} else {
		Info("tests are using default TESTLOGLVL log level", "loglevel", lvl)
	}
	testloghandler = LvlFilterHandler(lvl, StreamHandler(os.Stderr, TerminalFormat(true)))
	Root().SetHandler(testloghandler)
	Warn("new testloghandler", "loglevel", lvl, "nosync", NoSync)
}

func MustParseLevel(s string) Lvl {
	switch s {
	case "trace", "5", "6", "7", "8", "9":
		return LvlTrace
	case "debug", "4":
		return LvlDebug
	case "info", "3":
		return LvlInfo
	case "warn", "2":
		return LvlWarn
	case "error", "1":
		return LvlError
	case "crit", "critical", "0":
		return LvlCrit // actual silent level until a fatal error occurs
	default: // bad value
		panic("bad TESTLOGLVL: " + s)
	}
}
