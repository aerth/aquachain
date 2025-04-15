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
	"runtime"
	"sort"
	"time"

	logpkg "log"

	cli "github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/metrics"
	"gitlab.com/aquachain/aquachain/common/sense"
	"gitlab.com/aquachain/aquachain/internal/debug"
	"gitlab.com/aquachain/aquachain/opt/console"
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
		return x.Run(ctx, os.Args[1:]) // no subcommand given so we know all the args are flags :)
	},
}

// afterFunc only for this main package
func afterFunc(context.Context, *cli.Command) error {
	mainctxs.MainCancelCause()(fmt.Errorf("bye"))
	debug.Exit()
	console.Stdin.Close()
	return nil
}

// beforeFunc only for this main package
func beforeFunc(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	if err := debug.Setup(ctx, cmd); err != nil {
		return ctx, err
	}
	if x := cmd.Args().First(); x != "" && x != "daemon" || x != "console" { // is subcommand..
		return ctx, nil
	}

	// Start system runtime metrics collection
	go metrics.CollectProcessMetrics(3 * time.Second)
	if targetGasLimit := cmd.Uint(aquaflags.TargetGasLimitFlag.Name); targetGasLimit > 0 {
		params.TargetGasLimit = targetGasLimit
	}
	_, autoalertmode := sense.LookupEnv("ALERT_PLATFORM")
	if autoalertmode {
		cmd.Set(aquaflags.AlertModeFlag.Name, "true")
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
	if err := app.Run(mainctxs.Main(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: running %s failed with error %+v\n", app.Name, err)
	}
	if err := debug.WaitLoops(time.Second * 2); err != nil {
		log.Warn("waiting for loops", "err", err)
	} else if time.Since(subcommands.GetStartTime()) > time.Second*4 {
		log.Debug("graceful shutdown achieved", "subcommand", app.Name)
	}
}
