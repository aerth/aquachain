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

package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"errors"
	"io"
	"math/big"
	"os"

	"github.com/btcsuite/btcd/btcec/v2"
	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/math"
	"gitlab.com/aquachain/aquachain/rlp"
)

var (
	secp256k1_N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	secp256k1_halfN = new(big.Int).Div(secp256k1_N, big.NewInt(2))
)

// Creates an ethereum address given the bytes and the nonce
func CreateAddress(b common.Address, nonce uint64) common.Address {
	data, _ := rlp.EncodeToBytes([]interface{}{b, nonce})
	return common.BytesToAddress(Keccak256(data)[12:])
}

// // ToECDSA creates a private key with the given D value.
// func ToECDSA(d []byte) (*btcec.PrivateKey, error) {
// 	return toECDSA(d, true)
// }

// ToECDSAUnsafe blindly converts a binary blob to a private key. It should almost
// never be used unless you are sure the input is valid and want to avoid hitting
// errors due to bad origin encoding (0 prefixes cut off).
func ToECDSAUnsafe(d []byte) *btcec.PrivateKey {
	priv, _ := toECDSA(d, false)
	return priv
}

// toECDSA creates a private key with the given D value. The strict parameter
// controls whether the key's length should be enforced at the curve size or
// it can also accept legacy encodings (0 prefixes).
func toECDSA(d []byte, strict bool) (*btcec.PrivateKey, error) {
	k, _ := btcec.PrivKeyFromBytes(d)
	if k == nil {
		return nil, errors.New("invalid private key")
	}
	if strict && k.ToECDSA().Curve != S256() {
		return nil, errors.New("invalid curve")
	}
	if k.PubKey() == nil {
		return nil, errors.New("no public key")
	}
	return k, nil
}

// FromECDSA exports a private key into a binary dump.
func FromECDSA(priv *btcec.PrivateKey) []byte {
	if priv == nil {
		return nil
	}
	return math.PaddedBigBytes(priv.ToECDSA().D, priv.ToECDSA().Params().BitSize/8)
}

func ToECDSAPub(pub []byte) *btcec.PublicKey {
	if len(pub) == 0 {
		return nil
	}
	pubk, err := btcec.ParsePubKey(pub)
	if err != nil {
		return nil
	}
	return pubk
}

// FromECDSA serializes a public key to 65 bytes
func FromECDSAPub(pub *ecdsa.PublicKey) []byte {
	return elliptic.Marshal(S256(), pub.X, pub.Y)
}

func Ecdsa2Btcec(priv *ecdsa.PrivateKey) *btcec.PrivateKey {
	pk, _ := btcec.PrivKeyFromBytes(priv.D.Bytes())
	return pk
}
func Ecdsa2BtcecPub(pub *ecdsa.PublicKey) *btcec.PublicKey {
	pubk, err := btcec.ParsePubKey(FromECDSAPub(pub))
	if err != nil {
		log.Error("error parsing pub key", "err", err)
		panic(err.Error())
	}
	return pubk
}

// HexToECDSA parses a secp256k1 private key.
func HexToECDSA(hexkey string) (*ecdsa.PrivateKey, error) {
	k, err := HexToBtcec(hexkey)
	if err != nil {
		return nil, err
	}
	return k.ToECDSA(), nil
}

// HexToBtcec parses a secp256k1 private key.
func HexToBtcec(hexkey string) (*btcec.PrivateKey, error) {
	b, err := hex.DecodeString(hexkey)
	if err != nil {
		return nil, errors.New("invalid hex string")
	}
	return BytesToKey(b)

}

// BytesToKey used by HexToBtcec, HexToECDSA, LoadECDSA
func BytesToKey(b []byte) (*btcec.PrivateKey, error) {
	priv, pub := btcec.PrivKeyFromBytes(b)
	if pub == nil {
		return nil, errors.New("invalid private key")
	}
	if !bytes.Equal(priv.Serialize(), b) {
		return nil, errors.New("invalid private key serialization")
	}
	if priv.ToECDSA().Curve != S256() {
		return nil, errors.New("invalid curve")
	}
	if priv.ToECDSA().D.Cmp(secp256k1_N) >= 0 {
		return nil, errors.New("private key is too large")
	}
	if priv.ToECDSA().D.Cmp(common.Big0) == 0 {
		return nil, errors.New("private key is 0")
	}
	if priv.ToECDSA().D.Cmp(common.Big1) == 0 {
		return nil, errors.New("private key is 1")
	}
	if !pub.IsOnCurve() {
		return nil, errors.New("public key is not on curve")
	}
	return priv, nil
}

// LoadECDSA loads a secp256k1 private key from the given file.
func LoadECDSA(file string) (*btcec.PrivateKey, error) {
	buf := make([]byte, 64)
	fd, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	if _, err := io.ReadFull(fd, buf); err != nil {
		return nil, err
	}

	key, err := hex.DecodeString(string(buf))
	if err != nil {
		return nil, err
	}
	return BytesToKey(key)
}

// SaveECDSA saves a secp256k1 private key to the given file with
// restrictive permissions. The key data is saved hex-encoded.
func SaveECDSA(file string, key *btcec.PrivateKey) error {
	k := hex.EncodeToString(FromECDSA(key))
	return os.WriteFile(file, []byte(k), 0600)
}

// func GenerateKey() (*ecdsa.PrivateKey, error) {
// 	return ecdsa.GenerateKey(S256(), rand.Reader)
// }

// GenerateKey generates a new private key and checks it is valid
func GenerateKey() (*btcec.PrivateKey, error) {
	for i := 0; i < 1000; i++ {
		k, err := btcec.NewPrivateKey()
		if err != nil {
			log.Error("failed to generate key", "err", err)
			return nil, err
		}
		ke := k.ToECDSA()
		if !ke.IsOnCurve(ke.X, ke.Y) {
			log.Info("retrying key generation (invalid curve point)")
			continue
		}
		if k.PubKey().IsOnCurve() {
			return k, nil
		}
		log.Info("retrying key generation (invalid curve point)")
	}
	log.Error("failed to generate key")
	return nil, errors.New("failed to generate key")
}

// ValidateSignatureValues verifies whether the signature values are valid with
// the given chain rules. The v value is assumed to be either 0 or 1.
func ValidateSignatureValues(v byte, r, s *big.Int, homestead bool) bool {
	if v != 0 && v != 1 {
		return false
	}
	if r.Cmp(common.Big1) < 0 || s.Cmp(common.Big1) < 0 {
		return false
	}
	// reject upper range of s values (ECDSA malleability)
	// see discussion in secp256k1/libsecp256k1/include/secp256k1.h
	if homestead && s.Cmp(secp256k1_halfN) > 0 {
		return false
	}
	// Frontier: allow s to be in full N range
	return r.Cmp(secp256k1_N) < 0 && s.Cmp(secp256k1_N) < 0 // && (v == 0 || v == 1)
}

type HasSerializeUncompressed interface {
	SerializeUncompressed() []byte
}

func PubkeyToAddress(p HasSerializeUncompressed) common.Address {
	pubBytes := p.SerializeUncompressed()
	return common.BytesToAddress(Keccak256(pubBytes[1:])[12:])
}

type EllipticCurve interface {
	IsOnCurve(x, y *big.Int) bool
	Add(x1, y1, x2, y2 *big.Int) (*big.Int, *big.Int)
	ScalarMult(Bx, By *big.Int, k []byte) (*big.Int, *big.Int)
	ScalarBaseMult(k []byte) (*big.Int, *big.Int)
}

type EllipticCurveMarshal interface {
	IsOnCurve(x, y *big.Int) bool
	Marshal(x, y *big.Int) []byte
	Unmarshal(data []byte) (*big.Int, *big.Int)
}
