package abilite

import (
	"fmt"
	"log"
	"os"
	"testing"
)

func TestMethodSigHash(t *testing.T) {
	tcs := []struct {
		sig  string
		hash string
	}{
		{"", "c5d24601"},
		{"transfer(address,uint256)", "a9059cbb"},
		{"approve(address,uint256)", "095ea7b3"},
		{"transferFrom(address,address,uint256)", "23b872dd"},
	}

	for _, tc := range tcs {
		t.Run(tc.sig, func(t *testing.T) {
			hash := MethodSigHash(tc.sig)
			if fmt.Sprintf("%02x", hash) != tc.hash {
				t.Errorf("expected %s but got %x", tc.hash, hash)
			}
		})
	}
}

// Example generator func
func ExampleMethodSigHash() {
	log := log.New(os.Stdout, "", 0)
	log.Printf("package aquafoo\n\n")
	log.Printf("var empty = common.Hex2Bytes(\"%02x\")", MethodSigHash(""))
	log.Printf("var transfer = common.Hex2Bytes(\"%02x\")", MethodSigHash("transfer(address,uint256)"))
	log.Printf("var approve = common.Hex2Bytes(\"%02x\")", MethodSigHash("approve(address,uint256)"))
	log.Printf("var transferFrom = common.Hex2Bytes(\"%02x\")", MethodSigHash("transferFrom(address,address,uint256)"))
	// Output:
	// package aquafoo
	//
	// var empty = common.Hex2Bytes("c5d24601")
	// var transfer = common.Hex2Bytes("a9059cbb")
	// var approve = common.Hex2Bytes("095ea7b3")
	// var transferFrom = common.Hex2Bytes("23b872dd")
}
