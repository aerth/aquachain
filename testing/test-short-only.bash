#!/bin/bash
# testing/test-short-only.bash - run a limited set of tests and report results
# From https://gitlab.com/aquachain/aquachain
# Copyright (c) 2018-2025 The Aquachain Authors
# Usage: testing/test-short-only.bash [go test flags]

export CGO_ENABLED=${CGO_ENABLED-0}
which go >/dev/null || {
	echo "go not found in PATH"
	exit 3
}

run_go_vet() (
	set -o pipefail
	echo "running go vet" 1>&2
	go vet -v ./... 2>&1 | tee go-vet.log
	if [ ${PIPESTATUS[0]} -ne 0 ]; then
		echo "go vet failed"
		exit 1
	fi
) || exit 1

run_short_tests() (
	set -o pipefail
	if [ $? -ne 0 ]; then
		echo "go vet failed"
		exit 1
	fi
	packagelist=${TESTPACKAGELIST-$(go list ./... 2>/dev/null | egrep -v 'p2p|fetchers|downloader|peer|simulation')}
	tmpfile=$(mktemp tmpaqua-short-tests.XXXXXX.tmp)
	echo testshorttmpfile=$tmpfile
	echo "running short tests for packages: $packagelist" 1>&2
	# trap 'rm -f $tmpfile' EXIT # TODO: uncomment this line to cleanup
	export CGO_ENABLED=${CGO_ENABLED-0}
	go test -short $@ ${packagelist} | tee -a $tmpfile 1>&2
	exitcode=${PIPESTATUS[0]}
	echo "all done testing with exit code $exitcode"
	if [ $exitcode == 0 ]; then
		echo status=OK
		exit 0
	fi
	report_issues $tmpfile $exitcode
) || exit 1

report_issues() (
	tmpfile=$1
	exitcode=$2
	echo "----------------" 1>&2
	echo "failed tests:" 1>&2
	fails=$(cat $tmpfile | egrep -- '.*FAIL.*\..*')
	echo "num_fails=$(echo "$fails" | grep 'Test' | wc -l)"
	echo "$fails" 1>&2
	echo 1>&2
	echo "----------------" 1>&2
	echo "status=FAIL"
	exit $exitcode
)

run_go_vet || exit 1

run_short_tests "$@"
exit $?
