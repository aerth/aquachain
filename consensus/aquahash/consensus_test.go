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

package aquahash

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/math"
	"gitlab.com/aquachain/aquachain/core/types"
	"gitlab.com/aquachain/aquachain/params"
)

type diffTest struct {
	ParentTimestamp    uint64
	ParentDifficulty   *big.Int
	CurrentTimestamp   uint64
	CurrentBlocknumber *big.Int
	CurrentDifficulty  *big.Int
}

func (d *diffTest) UnmarshalJSON(b []byte) (err error) {
	var ext struct {
		ParentTimestamp    string `json:"parent_timestamp"`
		ParentDifficulty   string `json:"parent_difficulty"`
		CurrentTimestamp   string `json:"current_timestamp"`
		CurrentBlocknumber string `json:"current_blocknumber"`
		CurrentDifficulty  string `json:"current_difficulty"`
	}
	if err := json.Unmarshal(b, &ext); err != nil {
		return err
	}

	d.ParentTimestamp = math.MustParseUint64(ext.ParentTimestamp)
	d.ParentDifficulty = math.MustParseBig256(ext.ParentDifficulty)
	d.CurrentTimestamp = math.MustParseUint64(ext.CurrentTimestamp)
	d.CurrentBlocknumber = math.MustParseBig256(ext.CurrentBlocknumber)
	d.CurrentDifficulty = math.MustParseBig256(ext.CurrentDifficulty)

	return nil
}

func TestCalcDifficulty(t *testing.T) {
	tests := make(map[string]diffTest)
	file, err := os.Open(filepath.Join("..", "..", "tests", "testdata", "BasicTests", "difficulty.json"))
	if err == nil {
		defer file.Close()
		err = json.NewDecoder(file).Decode(&tests)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(tests) == 0 {
		tests["below-min"] = diffTest{
			ParentTimestamp:    0,
			ParentDifficulty:   big.NewInt(131072),
			CurrentTimestamp:   240,
			CurrentBlocknumber: big.NewInt(1),
			CurrentDifficulty:  params.MinimumDifficultyHF5,
		}
		tests["below-min-2"] = diffTest{
			ParentTimestamp:    0,
			ParentDifficulty:   big.NewInt(131072),
			CurrentTimestamp:   240,
			CurrentBlocknumber: big.NewInt(2),
			CurrentDifficulty:  params.MinimumDifficultyHF5,
		}
		tests["go up"] = diffTest{
			ParentTimestamp:    0,
			ParentDifficulty:   big.NewInt(46039386), // "MinimumDifficultyHF5"
			CurrentTimestamp:   90,
			CurrentBlocknumber: big.NewInt(1),
			CurrentDifficulty:  big.NewInt(46399068),
		}
		tests["go up"] = diffTest{
			ParentTimestamp:    0,
			ParentDifficulty:   big.NewInt(46039386), // "MinimumDifficultyHF5"
			CurrentTimestamp:   120,
			CurrentBlocknumber: big.NewInt(1),
			CurrentDifficulty:  big.NewInt(46399068), // go up
		}
		tests["go up"] = diffTest{
			ParentTimestamp:    0,
			ParentDifficulty:   big.NewInt(46039386), // "MinimumDifficultyHF5"
			CurrentTimestamp:   140,
			CurrentBlocknumber: big.NewInt(1),
			CurrentDifficulty:  big.NewInt(46399068), // go up
		}
		tests["go up"] = diffTest{
			ParentTimestamp:    0,
			ParentDifficulty:   big.NewInt(46039386), // "MinimumDifficultyHF5"
			CurrentTimestamp:   params.DurationLimitHF6.Uint64() - 1,
			CurrentBlocknumber: big.NewInt(1),
			CurrentDifficulty:  big.NewInt(46399068), // go up
		}
		tests["go up again"] = diffTest{
			ParentTimestamp:    0,
			ParentDifficulty:   big.NewInt(46399068), // from up
			CurrentTimestamp:   params.DurationLimitHF6.Uint64() - 1,
			CurrentBlocknumber: big.NewInt(1),
			CurrentDifficulty:  big.NewInt(46761560), // go up again
		}
		tests["stay same"] = diffTest{
			ParentTimestamp:    0,
			ParentDifficulty:   big.NewInt(46039386), // "MinimumDifficultyHF5"
			CurrentTimestamp:   params.DurationLimitHF6.Uint64() + 1,
			CurrentBlocknumber: big.NewInt(1),
			CurrentDifficulty:  big.NewInt(46039386), // stay same
		}
		tests["go down ok"] = diffTest{
			ParentTimestamp:    0,
			ParentDifficulty:   big.NewInt(46761560), // from up again
			CurrentTimestamp:   params.DurationLimitHF6.Uint64() + 1,
			CurrentBlocknumber: big.NewInt(1),
			CurrentDifficulty:  big.NewInt(46396236), // not same as from up
		}
		// json.NewEncoder(os.Stdout).Encode(tests)
	}
	config := &params.ChainConfig{HomesteadBlock: big.NewInt(0), ChainId: big.NewInt(1337), HF: params.AllAquahashProtocolChanges.HF}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			diff := CalcDifficulty(config, test.CurrentTimestamp, &types.Header{
				Number:     new(big.Int).Sub(test.CurrentBlocknumber, big.NewInt(1)),
				Time:       new(big.Int).SetUint64(test.ParentTimestamp),
				Difficulty: test.ParentDifficulty,
			}, nil)
			if diff.Cmp(test.CurrentDifficulty) != 0 {
				t.Error(name, "failed. Expected", test.CurrentDifficulty, "and calculated", diff)
			}
		})
	}
}

func TestNormalUncleReward2(t *testing.T) {
	x := new(big.Int).Set(normalUncleReward)
	log.Debug("normalUncleReward", "x", x)
	if x.String() != "1062500000000000000" {
		t.Errorf("normalUncleReward is not 1.0625 coin, got %s", x.String())
		return
	}
	if x.Cmp(maxRewardShould2) != 0 {
		t.Errorf("normalUncleReward is not %s, got %s", maxRewardShould2.String(), x.String())
		return
	}
	log.Warn("normalUncleReward", "x", x)
	tcs := []struct {
		hn  int64
		una []int64
	}{
		{100_000, []int64{100_002}},
		{100_000, []int64{100_003}},
		{100_000, []int64{100_004}},
		{100_000, []int64{100_005}},
		{100_006, []int64{100_000}},
		{100_007, []int64{100_000}},
		{100_008, []int64{100_000}},
		{100_000, []int64{100_002, 100_003}},
		{100_000, []int64{100_003, 100_004}},
		{100_000, []int64{100_004, 100_005}},
		{100_000, []int64{100_005, 100_006}},
		{100_006, []int64{100_000, 100_001}},
		{100_007, []int64{100_000, 100_001}},
		{100_008, []int64{100_000, 100_001}},
		{100_000, []int64{100_002, 100_003, 100_004}},
		{100_000, []int64{100_003, 100_004, 100_005}},
		{100_000, []int64{100_004, 100_005, 100_006}},
		{100_000, []int64{100_005, 100_006, 100_007}},
		{100_006, []int64{100_000, 100_001, 100_002, 100_003, 100_004, 100_005}},
	}

	for _, tc := range tcs {
		got := getNormalUncleReward(tc.una, tc.hn)
		log.Debug("normalUncleReward", "got", got, "hn", tc.hn, "unas", tc.una)
		if len(tc.una) == 1 && got.Cmp(maxRewardShould1) != 0 {
			t.Error("got", got, "hn", tc.hn, "unas", len(tc.una))
			continue
		}
		if len(tc.una) == 2 && got.Cmp(maxRewardShould2) > 0 {
			t.Error("got", got, "hn", tc.hn, "unas", len(tc.una))
			continue
		}
		// test preview algorithm is correct if there are (somehow) more uncles than maxUncles
		if len(tc.una) > maxUncles && got.Cmp(maxRewardShould2) < 0 {
			t.Error("got", got, "hn", tc.hn, "unas", len(tc.una))
			continue
		}
	}
}

var maxRewardShould1 = new(big.Int).SetUint64(1_031_250_000_000_000_000) // if maxUncles=1

var maxRewardShould2 = new(big.Int).SetUint64(1_062_500_000_000_000_000) // maxUncles=2
