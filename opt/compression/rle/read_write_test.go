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

package rle

import (
	"bytes"
	"testing"
)

func TestCompressionRle(t *testing.T) {
	suite := &CompressionRleSuite{}
	suite.TestDecompressSimple(t)
}

type CompressionRleSuite struct{}

func (s *CompressionRleSuite) TestDecompressSimple(t *testing.T) {
	exp := []byte{0xc5, 0xd2, 0x46, 0x1, 0x86, 0xf7, 0x23, 0x3c, 0x92, 0x7e, 0x7d, 0xb2, 0xdc, 0xc7, 0x3, 0xc0, 0xe5, 0x0, 0xb6, 0x53, 0xca, 0x82, 0x27, 0x3b, 0x7b, 0xfa, 0xd8, 0x4, 0x5d, 0x85, 0xa4, 0x70}
	res, err := Decompress([]byte{token, 0xfd})
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	if !bytes.Equal(res, exp) {
		t.Fatalf("Expected %v, got %v", exp, res)
	}

	exp = []byte{0x56, 0xe8, 0x1f, 0x17, 0x1b, 0xcc, 0x55, 0xa6, 0xff, 0x83, 0x45, 0xe6, 0x92, 0xc0, 0xf8, 0x6e, 0x5b, 0x48, 0xe0, 0x1b, 0x99, 0x6c, 0xad, 0xc0, 0x1, 0x62, 0x2f, 0xb5, 0xe3, 0x63, 0xb4, 0x21}
	res, err = Decompress([]byte{token, 0xfe})
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	if !bytes.Equal(res, exp) {
		t.Fatalf("Expected %v, got %v", exp, res)
	}

	res, err = Decompress([]byte{token, 0xff})
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	if !bytes.Equal(res, []byte{token}) {
		t.Fatalf("Expected %v, got %v", []byte{token}, res)
	}

	res, err = Decompress([]byte{token, 12})
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	if !bytes.Equal(res, make([]byte, 10)) {
		t.Fatalf("Expected %v, got %v", make([]byte, 10), res)
	}

}
