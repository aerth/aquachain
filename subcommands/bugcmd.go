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

package subcommands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"runtime"
	"strings"

	"gitlab.com/aquachain/aquachain/params"

	cli "github.com/urfave/cli/v3"
)

var bugCommand = &cli.Command{
	Action:    MigrateFlags(reportBug),
	Name:      "bug",
	Usage:     "opens a window to report a bug on the aquachain repo",
	ArgsUsage: " ",
	Category:  "MISCELLANEOUS COMMANDS",
}

const issueUrl = "https://github.com/aquachain/aquachain/issues/new"

// reportBug reports a bug by opening a new URL to the aquachain GH issue
// tracker and setting default values as the issue body.
func reportBug(ctx context.Context, cmd *cli.Command) error {
	// execute template and write contents to buff
	var buff bytes.Buffer

	fmt.Fprintf(&buff, "%s", header)
	fmt.Fprintln(&buff, "Version:", params.Version)
	fmt.Fprintln(&buff, "Go Version:", runtime.Version())
	fmt.Fprintln(&buff, "OS:", runtime.GOOS)
	printOSDetails(&buff)

	// open a new GH issue
	fmt.Printf("Bug template: %s\n\n", buff.String())
	printVersion(ctx, cmd)
	fmt.Printf("\n\nPlease file a new issue at %s using the above template.\n", issueUrl)
	return nil
}

// copied from the Go source. Copyright 2017 The Go Authors
func printOSDetails(w io.Writer) {
	switch runtime.GOOS {
	case "darwin":
		printCmdOut(w, "uname -v: ", "uname", "-v")
		printCmdOut(w, "", "sw_vers")
	case "linux":
		printCmdOut(w, "uname -sr: ", "uname", "-sr")
		printCmdOut(w, "", "lsb_release", "-a")
	case "openbsd", "netbsd", "freebsd", "dragonfly":
		printCmdOut(w, "uname -v: ", "uname", "-v")
	case "solaris":
		out, err := ioutil.ReadFile("/etc/release")
		if err == nil {
			fmt.Fprintf(w, "/etc/release: %s\n", out)
		} else {
			fmt.Printf("failed to read /etc/release: %v\n", err)
		}
	}
}

// printCmdOut prints the output of running the given command.
// It ignores failures; 'go bug' is best effort.
//
// copied from the Go source. Copyright 2017 The Go Authors
func printCmdOut(w io.Writer, prefix, path string, args ...string) {
	cmd := exec.Command(path, args...)
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("%s %s: %v\n", path, strings.Join(args, " "), err)
		return
	}
	fmt.Fprintf(w, "%s%s\n", prefix, bytes.TrimSpace(out))
}

const header = `Please answer these questions before submitting your issue. Thanks!

#### What did you do?

#### What did you expect to see?

#### What did you see instead?

#### System details
`
