package main

import (
	"context"
	"errors"
	"fmt"

	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/subcommands"
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
		fmt.Printf("command2 executed: someflag=%q debug=%v\n", cmd.String("someflag"), cmd.Bool("debug"))
		return nil
	},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "someflag",
			Usage: "some flag for command2",
		},
	},
}

func main() {
	app := &cli.Command{
		Name:     "newtester",
		Usage:    "newtester is a command line tool for testing",
		Flags:    subcommands.GlobalFlags,
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

	e := app.Run(ctx, os.Args)
	err := context.Cause(ctx)
	if e != nil && e != errMainQuit && e != err {
		fmt.Fprintf(os.Stderr, "error1: %v\n", e)
	}
	if err != nil && err != errMainQuit {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
	os.Exit(1)
}
