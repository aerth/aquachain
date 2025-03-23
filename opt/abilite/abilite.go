// abilite is lightweight helper package for working with the Aquachain ABI.
package abilite

import (
	"gitlab.com/aquachain/aquachain/crypto"
)

// MethodSigHash returns the 4-byte hash of the method signature.
//
// The sig must be correct, this function does no input checks.
//
// Example:
//
//	hash := MethodSigHash("transfer(address,uint256)")
func MethodSigHash(sig string) []byte {
	return crypto.Keccak256([]byte(sig))[:4]
}
