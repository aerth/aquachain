package main

import (
	"context"
	"errors"
	"fmt"

	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/internal/debug"
	"gitlab.com/aquachain/aquachain/subcommands/aquaflags"
)

var errMainQuit = errors.New("run success")

var appconfig = struct {
	chain string
}{
	chain: "testnet3",
}
var command1 = &cli.Command{
	Name:  "command1",
	Usage: "command1 is a test command",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		fmt.Println("command1 executed")
		return nil
	},
}
var command2 = &cli.Command{
	Name:  "command2",
	Usage: "command2 is a test command",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		fmt.Println("command2 executed")
		return nil
	},
}

var GlobalFlags = append([]cli.Flag{
	aquaflags.ChainFlag,
	aquaflags.DoitNowFlag,
	aquaflags.ConfigFileFlag,
	aquaflags.DataDirFlag},
	debug.LogFlags...,
)

func main() {
	app := &cli.Command{
		Name:  "newtester",
		Usage: "newtester is a command line tool for testing",
		Flags: GlobalFlags,
		// Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
		// 	// appconfig.chain = cmd.String(aquaflags.ChainFlag.Name)
		// 	return ctx, nil
		// },
		Commands: []*cli.Command{command1, command2},
	}

	ctx, cancelcause := context.WithCancelCause(context.Background())
	defer cancelcause(errMainQuit)

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
		sig := <-ch
		cancelcause(fmt.Errorf("received signal %s", sig))
	}()

	app.Run(ctx, os.Args)

	err := context.Cause(ctx)
	if err != nil && err != errMainQuit {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
