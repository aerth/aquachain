package log

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gitlab.com/aquachain/aquachain/common/sense"
)

var NoSync = !sense.EnvBoolDisabled("NO_LOGSYNC")

var PrintfDefaultLevel = LvlInfo

var bootTime = time.Now() // TODO dedupe

func TimeSinceBoot() time.Duration {
	return time.Since(bootTime)
}

func (l *logger) Printf(msg string, stuff ...any) {
	msg = fmt.Sprintf(msg, stuff...)
	l.writeskip(0, msg, PrintfDefaultLevel, []any{"todo", "oldlog"}) // add todo to log to know we should migrate it
}
func (l *logger) Infof(msg string, stuff ...any) {
	msg = fmt.Sprintf(msg, stuff...)
	l.writeskip(0, msg, LvlInfo, nil)
}
func (l *logger) Warnf(msg string, stuff ...any) {
	msg = fmt.Sprintf(msg, stuff...)
	l.writeskip(0, msg, LvlWarn, nil)
}

func Printf(msg string, stuff ...any) {
	root.Printf(msg, stuff...)
}
func Infof(msg string, stuff ...any) {
	msg = strings.TrimSuffix(msg, "\n")
	msg = fmt.Sprintf(msg, stuff...)
	root.writeskip(0, msg, LvlInfo, nil)
}
func Warnf(msg string, stuff ...any) {
	msg = strings.TrimSuffix(msg, "\n")
	msg = fmt.Sprintf(msg, stuff...)
	root.writeskip(0, msg, LvlWarn, nil)
}

var testloghandler Handler

// for test packages to call in init
func ResetForTesting() {
	if testloghandler != nil {
		return
	}
	lvl := LvlWarn
	envlvl := sense.GetFirstEnv("TESTLOGLVL", "LOGLEVEL", "LOGLVL")
	if x := envlvl; x != "" && x != "0" { // so TESTLOGLVL=0 is the same as not setting it (0=crit, which is silent)
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
	lvl, err := ParseLevel(s)
	if err != nil {
		panic(err.Error())
	}
	return lvl
}

//		switch s {
//		case "":
//			return LvlInfo
//		case "trace", "5", "6", "7", "8", "9":
//			return LvlTrace
//		case "debug", "4":
//			return LvlDebug
//		case "info", "3":
//			return LvlInfo
//		case "warn", "2":
//			return LvlWarn
//		case "error", "1":
//			return LvlError
//		case "crit", "critical", "0":
//			return LvlCrit // actual silent level until a fatal error occurs
//		default: // bad value
//			panic("bad TESTLOGLVL: " + s)
//		}
//	}
func ParseLevel(s string) (Lvl, error) {
	println("parsing log level", s)
	switch s {
	case "":
		return LvlInfo, fmt.Errorf("empty log level")
	case "trace", "5", "6", "7", "8", "9":
		return LvlTrace, nil
	case "debug", "4":
		return LvlDebug, nil
	case "info", "3":
		return LvlInfo, nil
	case "warn", "2":
		return LvlWarn, nil
	case "error", "1":
		return LvlError, nil
	case "crit", "critical", "0":
		return LvlCrit, nil // actual silent level until a fatal error occurs
	default: // bad value
		return LvlInfo, fmt.Errorf("could not parse log level: %s", s)
	}
}

func newRoot(handler Handler) *logger {
	x := &logger{[]interface{}{}, new(swapHandler)}
	x.SetHandler(handler)
	return x
}

var is_testing bool

func GetUnparsedLevelFromEnv() string {
	lvl := sense.Getenv("LOGLEVEL")
	if lvl == "" {
		lvl = sense.Getenv("TESTLOGLVL")
	}
	if lvl == "" {
		lvl = sense.Getenv("LOGLVL")
	}
	if lvl == "" {
		if is_testing {
			return "warn"
		}
		return "info"
	}
	return lvl
}

func GetLevelFromEnv() Lvl {
	lvl, err := ParseLevel(GetUnparsedLevelFromEnv())
	if err != nil {
		panic(err.Error())
	}
	return lvl
}

func newRootHandler() Handler {
	x := CallerFileHandler(StreamHandler(os.Stderr, TerminalFormat(true)))
	if sense.FeatureEnabled("JSONLOG", "jsonlog") {
		return CallerFileHandler(StreamHandler(os.Stderr, JsonFormatEx(false, true)))
	}
	lvl := GetLevelFromEnv()
	x = LvlFilterHandler(lvl, x)
	return x
}
