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

// aquachain is the official command-line client for Aquachain.
package main

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"runtime"
	"sort"
	"time"

	logpkg "log"

	cli "github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/common/alerts"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/metrics"
	"gitlab.com/aquachain/aquachain/common/sense"
	"gitlab.com/aquachain/aquachain/internal/debug"
	"gitlab.com/aquachain/aquachain/opt/console"
	"gitlab.com/aquachain/aquachain/p2p/discover"
	"gitlab.com/aquachain/aquachain/params"
	"gitlab.com/aquachain/aquachain/subcommands"
	"gitlab.com/aquachain/aquachain/subcommands/aquaflags"
	"gitlab.com/aquachain/aquachain/subcommands/mainctxs"
)

const (
	clientIdentifier = "aquachain" // Client identifier to advertise over the network
)

var (
	// Git SHA1 commit hash and timestamp of the release (set via linker flags)
	gitCommit, buildDate, gitTag string
)
var this_app *cli.Command

func init() {
	// main package initialize the buildinfo with values from Makefile/-X linker flags
	subcommands.SetBuildInfo(gitCommit, buildDate, gitTag, clientIdentifier)
}

var helpCommand = &cli.Command{
	Name:  "help",
	Usage: "show help",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		cli.ShowAppHelp(cmd)
		os.Exit(1)
		return nil
	},
	UsageText: "aquachain help",
}

// setupMain ... for this main package only
func setupMain() *cli.Command {
	if !sense.EnvBool("HELP2") {
		subcommands.InitHelp()
	}
	defaults := subcommands.NewApp(clientIdentifier, gitCommit, "the Aquachain command line interface")
	this_app = &cli.Command{
		Name:                       defaults.Name,
		Usage:                      defaults.Usage,
		Version:                    defaults.Version,
		EnableShellCompletion:      defaults.EnableShellCompletion,
		ShellCompletionCommandName: defaults.ShellCompletionCommandName,
		Suggest:                    defaults.Suggest,
		Flags: append([]cli.Flag{
			aquaflags.NoEnvFlag,
			aquaflags.DoitNowFlag,
			aquaflags.ConfigFileFlag,
			aquaflags.ChainFlag,
			aquaflags.GCModeFlag,
		}, debug.Flags...),
		SuggestCommandFunc: func(commands []*cli.Command, provided string) string {
			s := cli.SuggestCommand(commands, provided)
			// log.Info("running SuggestCommand", "commands", commands, "provided", provided, "suggesting", s)
			if s == provided {
				return s
			}

			println("did you mean:", s)
			os.Exit(1)
			return s
		},
		Before:         beforeFunc,
		After:          afterFunc,
		DefaultCommand: "consoledefault",
		Commands: append([]*cli.Command{
			helpCommand,
			consoledefault,
		}, subcommands.Subcommands()...),
		HideHelpCommand: true,
		HideVersion:     true,
		Copyright:       "Copyright 2018-2025 The Aquachain Authors",
	}
	{ // add and sort flags
		app := this_app
		// app.Flags = append(app.Flags, debug.Flags...)
		app.Flags = append(app.Flags, aquaflags.NodeFlags...)
		app.Flags = append(app.Flags, aquaflags.RPCFlags...)
		app.Flags = append(app.Flags, aquaflags.ConsoleFlags...)
		sort.Sort((cli.FlagsByName)(this_app.Flags))
	}
	return this_app
}

var consoledefault = &cli.Command{
	Name:  "consoledefault",
	Usage: "Start full interactive console",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		x := subcommands.SubcommandByName("console")
		if x.Root() == nil {
			return fmt.Errorf("woops")
		}
		args := append([]string{"console"}, os.Args[1:]...) // prepend 'console' subcommand for cli parse
		args2 := cmd.Args().Slice()
		log.Info("running consoledefault", "args", args, "args2", args2, "osargs", os.Args)
		return x.Run(ctx, args) // no subcommand given so we know all the args are flags :)
	},
	Flags: subcommands.SubcommandByName("console").Flags,
}

// afterFunc only for this main package
func afterFunc(context.Context, *cli.Command) error {
	mainctxs.MainCancelCause()(fmt.Errorf("finished")) // quit anything running in case it wasnt called
	debug.Exit()                                       // quit any running debug profiling
	console.Stdin.Close()
	time.Sleep(time.Second / 2) // just in case we got to this function by accident
	return nil
}

var mainsubcommand = ""

func isNodeFunc(subcmd string) bool {
	return subcmd == "" || subcmd == "daemon" || subcmd == "console" || subcmd == "consoledefault"
}

// beforeFunc only for this main package
func beforeFunc(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	if mainsubcommand != "" {
		return ctx, fmt.Errorf("beforeFunc called twice")
	}
	mainsubcommand = cmd.Args().First()
	if mainsubcommand == "" {
		mainsubcommand = consoledefault.Name
	}
	if cmd.Command(mainsubcommand) == nil {
		return ctx, fmt.Errorf("subcommand %s not found", mainsubcommand)
	}
	// if we are not running a node command, we dont need to do anything more here
	if !isNodeFunc(mainsubcommand) {
		log.Debug("not a node-starting subcommand", "subcommand", mainsubcommand)
		return ctx, nil
	}
	log.Debug("we are starting a node, checking runtime environment")
	if err := checkRuntimeEnvironment(); err != nil {
		log.Crit("runtime environment check failed: "+err.Error(), "subcommand", mainsubcommand)
		time.Sleep(time.Second / 2)
		return ctx, err
	}
	runtime.GOMAXPROCS(runtime.NumCPU())
	if err := debug.Setup(ctx, cmd); err != nil {
		return ctx, err
	}
	// Start system runtime metrics collection
	if metrics.Enabled = cmd.Bool(aquaflags.MetricsEnabledFlag.Name); metrics.Enabled {
		go metrics.CollectProcessMetrics(3 * time.Second)
	}
	if targetGasLimit := cmd.Uint(aquaflags.TargetGasLimitFlag.Name); targetGasLimit > 0 {
		params.TargetGasLimit = targetGasLimit
	}
	alertplatform, autoalertmode := sense.LookupEnv("ALERT_PLATFORM")
	if autoalertmode {
		log.Info("auto alert mode enabled", "platform", alertplatform)
		cmd.Set(aquaflags.AlertModeFlag.Name, "true")
	}
	if alertmode := cmd.Bool(aquaflags.AlertModeFlag.Name); alertmode {
		log.Info("alert mode enabled", "platform", alertplatform)
		alerts.ParseAlertConfig()
	}
	return ctx, nil
}

func main() {
	logpkg.SetFlags(logpkg.Lshortfile)
	if err := sense.DotEnv(); err != nil {
		println("dot env:", err.Error())
		os.Exit(1)
	}
	go func() {
		<-mainctxs.Main().Done()
		time.Sleep(time.Second * 10) // should never happen
		log.Warn("context has been done for 10 seconds and we are still running... consider sending SIGINT")
	}()
	app := setupMain()
	err := app.Run(mainctxs.Main(), os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: running %s failed with error %+v\n", app.Name, err)
	}
	fn := log.Debug
	if err != nil {
		fn = log.Error
	}
	fn("subcommand finished", "subcommand", mainsubcommand, "errored", err != nil, "error", err)
	if err := debug.WaitLoops(time.Second * 2); err != nil {
		log.Warn("waiting for loops", "err", err)
	} else if time.Since(subcommands.GetStartTime()) > time.Second*4 {
		log.Debug("graceful shutdown achieved", "subcommand", app.Name)
	}
}

func checkRuntimeEnvironment() error {
	// check working direcotry is not /
	wd, err := os.Getwd()
	if err == nil && wd == "/" {
		return fmt.Errorf("working directory is /, indicates a misconfiguration")
	}
	// check if the user is running as root
	u, err := user.Current()
	if err == nil && u.Uid == "0" {
		return fmt.Errorf("do not run as root")
	}
	// check time is not too far off
	if err := discover.CheckClockDrift(); err != nil {
		return fmt.Errorf("time check failed: %v", err)
	}

	return nil
}
