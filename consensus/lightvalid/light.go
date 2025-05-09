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

// The lightvalid package is a lightweight version of aquahash meant only for
// testing a nonce on a *trusted* block. It is primarily used in the
// open-aquachain-pool to validate POW (only the nonce is inputed by user)
package lightvalid

import (
	"encoding/binary"
	"errors"
	"math/big"

	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/crypto"
	"gitlab.com/aquachain/aquachain/params"
)

var NoMixDigest = common.Hash{}

// maxUint256 is a big integer representing 2^256-1
var maxUint256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))

var (
	// ErrNoVersion is returned if the block/header has no version
	ErrNoVersion = errors.New("header has no version")

	// ErrDifficultyZero is returned if difficulty is not set (use CalcDifficulty)
	ErrDifficultyZero = errors.New("difficulty is zero")

	// ErrMixDigestNonZero is returned if mix digest is nonzero. this may change in future hard forks
	ErrMixDigestNonZero = errors.New("invalid mix digest")

	// ErrPOW is returned if pow difficulty is not valid
	ErrPOW = errors.New("invalid proof of work")
)

func New() *Light {
	return &Light{}
}

type Light struct{}

type LightBlock interface {
	Difficulty() *big.Int
	HashNoNonce() common.Hash
	Nonce() uint64
	MixDigest() common.Hash
	NumberU64() uint64
	Version() params.HeaderVersion
}

// Verify checks whether the block's nonce is valid. Use VerifyWithError for the actual error.
func (l *Light) Verify(block LightBlock) bool {
	return l.VerifyWithError(block) == nil
}

// VerifyWithError returns an error if block is not valid
func (l *Light) VerifyWithError(block LightBlock) error {

	version := block.Version()
	if version < 2 || version > crypto.KnownVersion {
		return ErrNoVersion
	}

	// check difficulty is nonzero
	difficulty := block.Difficulty()
	if difficulty.Cmp(common.Big0) == 0 {
		return ErrDifficultyZero
	}

	// avoid mixdigest malleability as it's not included in a block's "hashNononce"
	if block.MixDigest() != NoMixDigest {
		return ErrMixDigestNonZero
	}

	// generate block hash
	seed := make([]byte, 40)
	copy(seed, block.HashNoNonce().Bytes())
	binary.LittleEndian.PutUint64(seed[32:], block.Nonce())
	result := crypto.VersionHash(byte(version), seed)

	// check number set from generated hash, is less than target diff
	target := new(big.Int).Div(maxUint256, difficulty)
	if new(big.Int).SetBytes(result).Cmp(target) <= 0 {
		return nil
	}
	return ErrPOW
}
