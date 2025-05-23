package sense

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"

	"github.com/joho/godotenv"
)

var main_argv = os.Args // allow test package to override

// FeatureEnabled returns true if the os env is truthy, or flagname is found in command line
func FeatureEnabled(envname string, flagname string) bool {
	if envname == "" && flagname == "" {
		panic("FeatureEnabled called with no args")
	}
	if envname != "" {
		if EnvBool(envname) {
			return true
		}
	}
	if flagname != "" {
		if FastParseArgsBool(flagname) {
			return true
		}
	}
	return false
}

// FeatureDisabled returns true if the os env is falsy, or flagname is found in command line
func FeatureDisabled(envname string, flagname string) bool {
	if envname == "" && flagname == "" {
		panic("FeatureDisabled called with no args")
	}
	if envname != "" {
		if EnvBoolDisabled(envname) {
			return true
		}
	}
	if flagname != "" {
		if FastParseArgsBool(flagname) {
			return true
		}
	}
	return false
}

// FastParseArgs is a quick way to check if a flag has been FOUND on the actual command line
//
// returns true if flagname is found, and the next argument
// if there is one.
//
// example:
//
//	found, value := FastParseArgs("-flagname")
//
//	if found {
//		// do something with value
//	}
//
// Completely skips first arg for tests
func FastParseArgs(flagname string) (bool, string) {
	if strings.Contains(flagname, "-") {
		panic("here, flagname should not contain -")
	}
	argc := len(main_argv)
	if argc == 0 {
		panic("no args")
	}
	argv := main_argv
	for i := 1; i < argc; i++ {
		if strings.Contains(argv[i], "=") {
			if strings.Split(argv[i], "=")[0] == "-"+flagname {
				return true, strings.Split(argv[i], "=")[1]
			}
		}
		if strings.Replace(argv[i], "-", "", 2) == flagname {
			if i+1 < argc {
				return true, argv[i+1]
			}
			return true, ""
		}
	}
	return false, ""
}

// FastParseArgsBool is a quick way to check if a bool flag has been FOUND on the actual command line
func FastParseArgsBool(flagname string) bool {
	x, next := FastParseArgs(flagname)
	if next == "" {
		return x
	}
	// skip if next is flag, check if next is "disabled"
	if !strings.HasPrefix(next, "-") && isFalsy(next) {
		fmt.Fprintf(os.Stderr, "warn: bool is falsy, right?!: %q\n", next)
		return false
	}
	return x
}

func boolString(s string, unset bool, unparsable bool) bool {
	switch strings.ToLower(s) {
	case "":
		return unset
	case "true", "yes", "1", "on", "enabled", "enable":
		return true
	case "false", "no", "0", "off", "disabled", "disable":
		return false
	default:
		if !strings.HasPrefix(s, "-") {
			fmt.Fprintf(os.Stderr, "warn: unknown bool string: %q\n", s)
		}
		return unparsable
	}

}

var _dotenvdone atomic.Bool

func DotEnv(extras ...string) error {
	if len(extras) == 0 && !_dotenvdone.CompareAndSwap(false, true) {
		return nil
	}
	// check for .env file unless -noenv is in args
	// (before flags are parsed)
	// todo: use sense package
	noenv := false
	for _, v := range os.Args {
		if strings.Contains(v, "-noenv") {
			noenv = true
		}
	}
	var err error
	if !noenv {
		err = godotenv.Load(append([]string{".env"}, extras...)...)
	} else {
		println("Skipping .env file")
	}
	return err
}

func Getenv(name string) string {
	DotEnv()
	return os.Getenv(name) // should be the only os.Getenv call.
}

var LookupEnv = osLookupEnv

func osLookupEnv(name string) (string, bool) {
	DotEnv()                  // noop if already done
	return os.LookupEnv(name) // should be the only os.LookupEnv call in the codebase to make sure a .env file is sourced before any env vars are read
}

// EnvBool returns false if empty/unset/falsy, true if otherwise non-empty
func EnvBool(name string) bool {
	x, ok := osLookupEnv(name)
	if !ok {
		return false
	}
	return boolString(x, false, true)
}

// EnvBoolDisabled returns true only if nonempty+falsy (such as "0" or "false")
//
// a bit different logic than !EnvBool
func EnvBoolDisabled(name string) bool {
	x, ok := osLookupEnv(name)
	if !ok {
		return false
	}
	return isFalsy(x)
}

func isFalsy(s string) bool {
	return !boolString(s, true, true)
}

// EnvOr returns the value of the environment variable, or the default if unset
func EnvOr(name, def string) string {
	x, ok := osLookupEnv(name)
	if !ok {
		return def
	}
	return x
}

// example usages of FeatureEnabled

// IsNoKeys the one true way
func IsNoKeys() bool {
	return FeatureEnabled("NO_KEYS", "nokeys")
}

// IsNoSign the one true way
func IsNoSign() bool {
	return FeatureEnabled("NO_SIGN", "nosign")
}

func IsNoCountdown() bool {
	return FeatureEnabled("NO_COUNTDOWN", "now")
}
