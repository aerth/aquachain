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

package abi

import (
	"fmt"
	"strings"

	"gitlab.com/aquachain/aquachain/crypto"
)

// Method represents a callable given a `Name` and whether the method is a constant.
// If the method is `Const` no transaction needs to be created for this
// particular Method call. It can easily be simulated using a local VM.
// For example a `Balance()` method only needs to retrieve something
// from the storage and therefor requires no Tx to be send to the
// network. A method such as `Transact` does require a Tx and thus will
// be flagged `true`.
// Input specifies the required input parameters for this gives method.
type Method struct {
	Name    string
	Const   bool
	Inputs  Arguments
	Outputs Arguments
}

// Sig returns the methods string signature according to the ABI spec.
//
// Example
//
//	function foo(uint32 a, int b)    =    "foo(uint32,int256)"
//
// Please note that "int" is substitute for its canonical representation "int256"
func (method Method) Sig() string {
	types := make([]string, len(method.Inputs))
	i := 0
	for _, input := range method.Inputs {
		types[i] = input.Type.String()
		i++
	}
	return fmt.Sprintf("%v(%v)", method.Name, strings.Join(types, ","))
}

func (method Method) String() string {
	inputs := make([]string, len(method.Inputs))
	for i, input := range method.Inputs {
		inputs[i] = fmt.Sprintf("%v %v", input.Name, input.Type)
	}
	outputs := make([]string, len(method.Outputs))
	for i, output := range method.Outputs {
		if len(output.Name) > 0 {
			outputs[i] = fmt.Sprintf("%v ", output.Name)
		}
		outputs[i] += output.Type.String()
	}
	constant := ""
	if method.Const {
		constant = "constant "
	}
	return fmt.Sprintf("function %v(%v) %sreturns(%v)", method.Name, strings.Join(inputs, ", "), constant, strings.Join(outputs, ", "))
}

func (method Method) Id() []byte {
	return crypto.Keccak256([]byte(method.Sig()))[:4]
}
