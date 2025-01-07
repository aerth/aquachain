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
	"log"
	"math/big"
	"os"

	"github.com/btcsuite/btcd/btcec/v2"
	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/math"
	"gitlab.com/aquachain/aquachain/crypto/sha3"
	"gitlab.com/aquachain/aquachain/rlp"
	"golang.org/x/crypto/argon2"
)

var (
	secp256k1_N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	secp256k1_halfN = new(big.Int).Div(secp256k1_N, big.NewInt(2))
)

const (
	argonThreads uint8  = 1
	argonTime    uint32 = 1
)

const KnownVersion = 4

// VersionHash switch version, returns digest bytes, v is not hashed.
func VersionHash(v byte, data ...[]byte) []byte {
	switch v {
	//	case 0:
	//		return Keccak256(data...)
	case 1:
		return Keccak256(data...)
	case 2:
		return Argon2idA(data...)
	case 3:
		return Argon2idB(data...)
	case 4:
		return Argon2idC(data...)
	default:
		panic("invalid block version")
	}
}

// Argon2id calculates and returns the Argon2id hash of the input data, using 1kb mem
func Argon2idA(data ...[]byte) []byte {
	//fmt.Printf(".")
	buf := &bytes.Buffer{}
	for i := range data {
		buf.Write(data[i])
	}
	return argon2.IDKey(buf.Bytes(), nil, argonTime, 1, argonThreads, common.HashLength)
}

// Argon2id calculates and returns the Argon2id hash of the input data, using 16kb mem
func Argon2idB(data ...[]byte) []byte {
	//fmt.Printf(".")
	buf := &bytes.Buffer{}
	for i := range data {
		buf.Write(data[i])
	}
	return argon2.IDKey(buf.Bytes(), nil, argonTime, 16, argonThreads, common.HashLength)
}

// Argon2id calculates and returns the Argon2id hash of the input data, using 32kb
func Argon2idC(data ...[]byte) []byte {
	//fmt.Printf(".")
	buf := &bytes.Buffer{}
	for i := range data {
		buf.Write(data[i])
	}
	return argon2.IDKey(buf.Bytes(), nil, argonTime, 32, argonThreads, common.HashLength)
}

// Argon2id calculates and returns the Argon2id hash of the input data.
func Argon2idAHash(data ...[]byte) (h common.Hash) {
	return common.BytesToHash(Argon2idA(data...))
}

// Argon2id calculates and returns the Argon2id hash of the input data.
func Argon2idBHash(data ...[]byte) (h common.Hash) {
	return common.BytesToHash(Argon2idB(data...))
}

// Argon2id calculates and returns the Argon2id hash of the input data.
func Argon2idCHash(data ...[]byte) (h common.Hash) {
	return common.BytesToHash(Argon2idC(data...))
}

// Keccak256 calculates and returns the Keccak256 hash of the input data.
func Keccak256(data ...[]byte) []byte {
	//fmt.Printf("o")
	d := sha3.NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}

// Keccak256Hash calculates and returns the Keccak256 hash of the input data,
// converting it to an internal Hash data structure.
func Keccak256Hash(data ...[]byte) (h common.Hash) {
	//fmt.Printf("x")
	d := sha3.NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	d.Sum(h[:0])
	return h
}

// Keccak512 calculates and returns the Keccak512 hash of the input data.
func Keccak512(data ...[]byte) []byte {
	d := sha3.NewKeccak512()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}

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

func FromECDSAPub(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(S256(), pub.X, pub.Y)
}
func Ecdsa2Btcec(priv *ecdsa.PrivateKey) *btcec.PrivateKey {
	pk, _ := btcec.PrivKeyFromBytes(priv.D.Bytes())
	return pk
}
func Ecdsa2BtcecPub(pub *ecdsa.PublicKey) *btcec.PublicKey {
	pubk, _ := btcec.ParsePubKey(FromECDSAPub(pub))
	return pubk
}

// HexToECDSA parses a secp256k1 private key.
func HexToECDSA(hexkey string) (*btcec.PrivateKey, error) {
	b, err := hex.DecodeString(hexkey)
	if err != nil {
		return nil, errors.New("invalid hex string")
	}
	return BytesToKey(b)

}

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

	log.Printf("key: %02x", priv.Serialize())
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

func GenerateKey() (*btcec.PrivateKey, error) {
	return btcec.NewPrivateKey()
}

// ValidateSignatureValues verifies whether the signature values are valid with
// the given chain rules. The v value is assumed to be either 0 or 1.
func ValidateSignatureValues(v byte, r, s *big.Int, homestead bool) bool {
	if r.Cmp(common.Big1) < 0 || s.Cmp(common.Big1) < 0 {
		log.Printf("negative sig")
		return false
	}
	// reject upper range of s values (ECDSA malleability)
	// see discussion in secp256k1/libsecp256k1/include/secp256k1.h
	if homestead && s.Cmp(secp256k1_halfN) > 0 {
		log.Printf("mall sig")
		return false
	}
	if v != 0 && v != 1 {
		log.Printf("invalid v: %d", v)
		return false
	}
	// Frontier: allow s to be in full N range
	return r.Cmp(secp256k1_N) < 0 && s.Cmp(secp256k1_N) < 0 // && (v == 0 || v == 1)
}

func PubkeyToAddress(p *btcec.PublicKey) common.Address {
	pubBytes := p.SerializeUncompressed()
	return common.BytesToAddress(Keccak256(pubBytes[1:])[12:])
}
