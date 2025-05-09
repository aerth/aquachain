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

package build

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"gitlab.com/aquachain/aquachain/common/sense"
)

var (
	// These flags override values in build env.
	GitCommitFlag   = flag.String("git-commit", "", `Overrides git commit hash embedded into executables`)
	GitBranchFlag   = flag.String("git-branch", "", `Overrides git branch being built`)
	GitTagFlag      = flag.String("git-tag", "", `Overrides git tag being built`)
	BuildnumFlag    = flag.String("buildnum", "", `Overrides CI build number`)
	PullRequestFlag = flag.Bool("pull-request", false, `Overrides pull request status of the build`)
	CronJobFlag     = flag.Bool("cron-job", false, `Overrides cron job status of the build`)
	StaticFlag      = flag.Bool("static", false, `Use static linking`)
	MuslFlag        = flag.Bool("musl", false, `Use musl c library`)
	RaceFlag        = flag.Bool("race", false, `Use race detector (slow runtime!)`)
	UseUSBFlag      = flag.Bool("usb", false, `Use usb (trezor/ledger)`)
)

// Environment contains metadata provided by the build environment.
type Environment struct {
	Name                string // name of the environment
	Repo                string // name of GitHub repo
	Commit, Branch, Tag string // Git info
	Buildnum            string
	IsPullRequest       bool
	IsCronJob           bool
	Config              map[string]bool
}

func (env Environment) String() string {
	return fmt.Sprintf("%s env (commit:%s branch:%s tag:%s buildnum:%s pr:%t config:%v)",
		env.Name, env.Commit, env.Branch, env.Tag, env.Buildnum, env.IsPullRequest, env.Config)
}

// Env returns metadata about the current CI environment, falling back to LocalEnv
// if not running on CI.
func Env() Environment {
	switch {
	case sense.Getenv("CI") == "true" && sense.Getenv("TRAVIS") == "true":
		return Environment{
			Name:          "travis",
			Repo:          sense.Getenv("TRAVIS_REPO_SLUG"),
			Commit:        sense.Getenv("TRAVIS_COMMIT"),
			Branch:        sense.Getenv("TRAVIS_BRANCH"),
			Tag:           sense.Getenv("TRAVIS_TAG"),
			Buildnum:      sense.Getenv("TRAVIS_BUILD_NUMBER"),
			IsPullRequest: sense.Getenv("TRAVIS_PULL_REQUEST") != "false",
			IsCronJob:     sense.Getenv("TRAVIS_EVENT_TYPE") == "cron",
			Config:        map[string]bool{},
		}
	default:
		return LocalEnv()
	}
}

// LocalEnv returns build environment metadata gathered from git.
func LocalEnv() Environment {
	env := applyEnvFlags(Environment{Name: "local", Repo: "aquachain/aquachain"})

	head := readGitFile("HEAD")
	if splits := strings.Split(head, " "); len(splits) == 2 {
		head = splits[1]
	} else {
		return env
	}
	if env.Commit == "" {
		env.Commit = readGitFile(head)
	}
	if env.Branch == "" {
		if head != "HEAD" {
			env.Branch = strings.TrimPrefix(head, "refs/heads/")
		}
	}
	if info, err := os.Stat(".git/objects"); err == nil && info.IsDir() && env.Tag == "" {
		env.Tag = firstLine(RunGit("tag", "-l", "--points-at", "HEAD"))
	}
	return env
}

func firstLine(s string) string {
	return strings.Split(s, "\n")[0]
}

func applyEnvFlags(env Environment) Environment {
	if !flag.Parsed() {
		panic("you need to call flag.Parse before Env or LocalEnv")
	}
	if *GitCommitFlag != "" {
		env.Commit = *GitCommitFlag
	}
	if *GitBranchFlag != "" {
		env.Branch = *GitBranchFlag
	}
	if *GitTagFlag != "" {
		env.Tag = *GitTagFlag
	}
	if *BuildnumFlag != "" {
		env.Buildnum = *BuildnumFlag
	}
	if *PullRequestFlag {
		env.IsPullRequest = true
	}
	if *CronJobFlag {
		env.IsCronJob = true
	}

	if env.Config == nil {
		env.Config = map[string]bool{}
	}

	if *StaticFlag {
		env.Config["static"] = *StaticFlag
	}

	if *MuslFlag {
		env.Config["musl"] = *MuslFlag
	}

	if *RaceFlag {
		env.Config["race"] = *RaceFlag // uh-oh
	}

	if *UseUSBFlag {
		env.Config["usb"] = *UseUSBFlag
	}

	return env
}
