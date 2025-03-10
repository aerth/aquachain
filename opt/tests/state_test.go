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

package tests

import (
	"bytes"
	"fmt"
	"testing"

	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/sense"
	"gitlab.com/aquachain/aquachain/core/vm"
)

func init() {
	log.ResetForTesting()
}
func TestState(t *testing.T) {
	// if sense.Getenv("DEBUG") == "1" {
	// 	log.SetRootHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	// 	log.PrintOrigins(true)
	// }
	// t.Parallel()

	st := new(testMatcher)
	// Long tests:
	// st.skipShortMode(`^stQuadraticComplexityTest/`)
	// Broken tests:
	// st.skipLoad(`^stTransactionTest/OverflowGasRequire\.json`) // gasLimit > 256 bits
	// st.skipLoad(`^stTransactionTest/zeroSigTransa[^/]*\.json`) // EIP-86 is not supported yet
	// Expected failures:
	// st.fails(`^stRevertTest/RevertPrecompiledTouch\.json/EIP158`, "bug in test")
	// st.fails(`^stRevertTest/RevertPrefoundEmptyOOG\.json/EIP158`, "bug in test")
	// st.fails(`^stRevertTest/RevertPrecompiledTouch\.json/Byzantium`, "bug in test")
	// st.fails(`^stRevertTest/RevertPrefoundEmptyOOG\.json/Byzantium`, "bug in test")
	// st.fails(`^stRandom2/randomStatetest64[45]\.json/(EIP150|Frontier|Homestead)/.*`, "known bug #15119")
	// st.fails(`^stCreateTest/TransactionCollisionToEmpty\.json/EIP158/2`, "known bug ")
	// st.fails(`^stCreateTest/TransactionCollisionToEmpty\.json/EIP158/3`, "known bug ")
	// st.fails(`^stCreateTest/TransactionCollisionToEmpty\.json/Byzantium/2`, "known bug ")
	// st.fails(`^stCreateTest/TransactionCollisionToEmpty\.json/Byzantium/3`, "known bug ")

	st.walk(t, stateTestDir, func(t *testing.T, name string, test *StateTest) {
		for _, subtest := range test.Subtests() {
			subtest := subtest
			key := fmt.Sprintf("%s/%d", subtest.Fork, subtest.Index)
			name := name + "/" + key
			t.Run(key, func(t *testing.T) {
				if subtest.Fork == "Constantinople" {
					t.Skip("constantinople not supported yet")
				}
				withTrace(t, test.gasLimit(subtest), func(vmconfig vm.Config) error {
					_, err := test.Run(subtest, vmconfig)
					return st.checkFailure(t, name, err)
				})
			})
		}
	})
}

// Transactions with gasLimit above this value will not get a VM trace on failure.
const traceErrorLimit = 400000

func withTrace(t *testing.T, gasLimit uint64, test func(vm.Config) error) {
	err := test(vm.Config{})
	if err == nil {
		return
	}
	skiplog := sense.Getenv("NOTRACE") == "1"

	if gasLimit > traceErrorLimit {
		t.Log("gas limit too high for EVM trace")
		return
	}
	t.Logf("re-running with EVM trace: err=%v", err)
	tracer := vm.NewStructLogger(nil)
	err2 := test(vm.Config{Debug: true, Tracer: tracer})
	if (err != nil) != (err2 != nil) {
		t.Errorf("err != nil: %v, err2 != nil: %v", err, err2)
		return
	}
	if err.Error() != err2.Error() {
		t.Errorf("different error for second run: %v", err2)
	}
	buf := new(bytes.Buffer)
	vm.WriteTrace(buf, tracer.StructLogs())
	if buf.Len() == 0 {
		t.Log("no EVM operation logs generated")
	} else if skiplog {
		t.Logf("EVM operation log: %s", shorten(buf.String(), 100))
	} else {
		t.Log("EVM operation log:\n" + buf.String())
	}
	t.Logf("EVM output: 0x%x", tracer.Output())
	t.Logf("EVM error: %v", tracer.Error())
}

func shorten(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
