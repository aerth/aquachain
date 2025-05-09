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

package params

import (
	"fmt"
	"math/big"
	"os"

	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/toml"
)

var (
	MainnetGenesisHash  = common.HexToHash("0x381c8d2c3e3bc702533ee504d7621d510339cafd830028337a4b532ff27cd505") // Mainnet genesis hash to enforce below configs on
	TestnetGenesisHash  = common.HexToHash("0xa8773cb7d32b8f7e1b32b0c2c8b735c293b8936dd3760c15afc291a23eb0cf88") // Testnet genesis hash to enforce below configs on
	Testnet2GenesisHash = common.HexToHash("0xde434983d3ada19cd43c44d8ad5511bad01ed12b3cc9a99b1717449a245120df") // Testnet2 genesis hash to enforce below configs on
	Testnet3GenesisHash = common.HexToHash("0x05c1df1f60eedd42bdf3f002bedc4688c5bf0443771d1d30341bc5e4fe76bce8") // Testnet3 genesis hash to enforce below configs on
	EthnetGenesisHash   = common.HexToHash("0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3")
	TestingGenesisHash  = common.HexToHash("0x1556b25efd267015f318c621d79231cb43e694c92d3776254a8581cfde2f67cd")
	AllGenesisHash      = common.HexToHash("0x02f036d05929e5762b8e83ce7c104b37881922732b32fd39253efe1e7e5c2b51")
)

// KnownHF is the highest hard fork that is known by this version of Aquachain.
const KnownHF = 9

var (
	// AquachainHF is the map of hard forks (mainnet)
	AquachainHF = ForkMap{
		1: big.NewInt(3600),  // HF1 (difficulty algo) increase min difficulty to the next multiple of 2048
		2: big.NewInt(7200),  // HF2 (difficulty algo) use simple difficulty algo (240 seconds)
		3: big.NewInt(13026), // HF3 (difficulty) increase min difficulty for anticipation of gpu mining
		4: big.NewInt(21800), // HF4 (supply) remove ethereum genesis allocation
		5: big.NewInt(22800), // HF5 (POW) argonated (algo #2) (argon2id)
		6: big.NewInt(36000), // HF6 (difficulty algo) divisor increase
		7: big.NewInt(36050), // HF7 (EIP 155, 158)
		8: nil,               // HF8 (m_cost=16, diff algo, jump diff) // activated with flag
	}

	// TestnetHF is the map of hard forks (testnet public network)
	TestnetHF = ForkMap{
		1: big.NewInt(1),   // increase min difficulty to the next multiple of 2048
		2: big.NewInt(2),   // use simple difficulty algo (240 seconds)
		3: big.NewInt(3),   // increase min difficulty for anticipation of gpu mining
		4: big.NewInt(4),   // HF4
		5: big.NewInt(5),   // HF5
		6: big.NewInt(6),   // noop in testnet
		7: big.NewInt(25),  // eip 155, 158
		8: big.NewInt(650), // HF8 (m_cost=16, diff algo, jump diff)
	}

	// Testnet2HF is the map of hard forks (testnet2 private network)
	Testnet2HF = ForkMap{
		5: big.NewInt(0),
		6: big.NewInt(0),
		7: big.NewInt(0),
		8: big.NewInt(8),
		9: big.NewInt(19),
	}

	// Testnet3HF is the map of hard forks (testnet3 private network)
	Testnet3HF = ForkMap{
		// 1: big.NewInt(0),
		// 2: big.NewInt(0),
		// 3: big.NewInt(0),
		// 4: big.NewInt(0),
		5: big.NewInt(0),
		7: big.NewInt(0), // eip 155, 158
	}

	// TestHF is the map of hard forks (for testing suite)
	TestHF = ForkMap{
		1: big.NewInt(1),
		2: big.NewInt(2),
		3: big.NewInt(3),
		4: big.NewInt(4),
		5: big.NewInt(5), // argon2id
		6: big.NewInt(6),
		7: big.NewInt(7),
		// 8: big.NewInt(0),
	}

	NoHF  = (ForkMap)(nil)
	AllHF = ForkMap{
		1: big.NewInt(0),
		2: big.NewInt(0),
		3: big.NewInt(0),
		4: big.NewInt(0),
		5: big.NewInt(0), // argon2id
		6: big.NewInt(0),
		7: big.NewInt(0),
		// 8: big.NewInt(0),
	}
)

func IsMainnet(cfg *ChainConfig) bool {
	return cfg == MainnetChainConfig
}

func IsValid(cfg *ChainConfig) bool {
	return cfg != nil && cfg.ChainId != nil && (cfg.ChainId.Cmp(MainnetChainConfig.ChainId) != 0 || cfg == MainnetChainConfig)
}

var (
	// MainnetChainConfig is the chain parameters to run a node on the main network.
	MainnetChainConfig = &ChainConfig{
		ChainId:              big.NewInt(61717561),
		HomesteadBlock:       big.NewInt(0),
		EIP150Block:          big.NewInt(0),
		EIP155Block:          AquachainHF[7],
		EIP158Block:          AquachainHF[7],
		ByzantiumBlock:       AquachainHF[7],
		Aquahash:             new(AquahashConfig),
		HF:                   AquachainHF,
		DefaultPortNumber:    21303,
		DefaultBootstrapPort: 21000,
	}

	// TestnetChainConfig contains the chain parameters to run a node on the Aquachain test network.
	TestnetChainConfig = &ChainConfig{
		ChainId:              big.NewInt(617175611),
		HomesteadBlock:       big.NewInt(0),
		EIP150Block:          big.NewInt(0),
		EIP155Block:          TestnetHF[7],
		EIP158Block:          TestnetHF[7],
		ByzantiumBlock:       TestnetHF[7],
		Aquahash:             new(AquahashConfig),
		HF:                   TestnetHF,
		DefaultPortNumber:    21304,
		DefaultBootstrapPort: 21001,
	}

	// Testnet2ChainConfig contains the chain parameters to run a node on the Testnet2 test network.
	Testnet2ChainConfig = &ChainConfig{
		ChainId:              big.NewInt(617175612),
		HomesteadBlock:       big.NewInt(0),
		EIP150Block:          big.NewInt(0),
		EIP155Block:          Testnet2HF[7],
		EIP158Block:          Testnet2HF[7],
		ByzantiumBlock:       Testnet2HF[7],
		Aquahash:             new(AquahashConfig),
		HF:                   Testnet2HF,
		DefaultPortNumber:    21305,
		DefaultBootstrapPort: 21002,
	}
	// Testnet3ChainConfig contains the chain parameters to run a node on the Testnet2 test network.
	Testnet3ChainConfig = &ChainConfig{
		ChainId:        big.NewInt(617175613),
		HomesteadBlock: big.NewInt(0),
		EIP150Block:    big.NewInt(0),
		EIP155Block:    Testnet3HF[7],
		EIP158Block:    Testnet3HF[7],
		ByzantiumBlock: Testnet3HF[7],
		// Aquahash:             new(AquahashConfig),
		Clique:               &CliqueConfig{Period: 15, Epoch: 30000},
		HF:                   Testnet3HF,
		DefaultPortNumber:    21306,
		DefaultBootstrapPort: 21003,
	}

	// AllAquahashProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Aquachain core developers into the Aquahash consensus.
	//
	// This configuration is intentionally not using keyed fields to force anyone
	// adding flags to the config to also have to set these fields.
	AllAquahashProtocolChanges = &ChainConfig{big.NewInt(1337), big.NewInt(0), nil, false, big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), nil, new(AquahashConfig), nil, AllHF, 21398, 21099}
	AllCliqueProtocolChanges   = &ChainConfig{big.NewInt(1337), big.NewInt(0), nil, false, big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), nil, nil, &CliqueConfig{Period: 0, Epoch: 30000}, AllHF, 21398, 21098}

	TestChainConfig = &ChainConfig{big.NewInt(3), big.NewInt(0), nil, false, big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), nil, new(AquahashConfig), nil, TestHF, 21397, 21097}
	// TestRules       = TestChainConfig.Rules(new(big.Int))
)

// ChainConfig is the core config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig struct {
	ChainId *big.Int `json:"chainId"` // Chain id identifies the current chain and is used for replay protection

	HomesteadBlock *big.Int `json:"homesteadBlock,omitempty"` // Homestead switch block (nil = no fork, 0 = already homestead)

	// et junk to remove
	DAOForkBlock   *big.Int `json:"daoForkBlock,omitempty"`   // TheDAO hard-fork switch block (nil = no fork)
	DAOForkSupport bool     `json:"daoForkSupport,omitempty"` // Whether the nodes supports or opposes the DAO hard-fork

	// EIP150 implements the Gas price changes (https://github.com/aquanetwork/EIPs/issues/150)
	EIP150Block *big.Int    `json:"eip150Block,omitempty"` // EIP150 HF block (nil = no fork)
	EIP150Hash  common.Hash `json:"eip150Hash,omitempty"`  // EIP150 HF hash (needed for header only clients as only gas pricing changed)

	EIP155Block *big.Int `json:"eip155Block,omitempty"` // EIP155 HF block (replay protect)
	EIP158Block *big.Int `json:"eip158Block,omitempty"` // EIP158 HF block (state clearing)

	ByzantiumBlock      *big.Int `json:"byzantiumBlock,omitempty"`      // Byzantium switch block (nil = no fork, 0 = already on byzantium)
	ConstantinopleBlock *big.Int `json:"constantinopleBlock,omitempty"` // Constantinople switch block (nil = no fork, 0 = already activated)

	// Various consensus engines
	Aquahash *AquahashConfig `json:"aquahash,omitempty"`
	Clique   *CliqueConfig   `json:"clique,omitempty"`

	// HF Scheduled Maintenance Hardforks
	HF ForkMap `json:"hf,omitempty"`

	// DefaultPortNumber used by p2p package if nonzero
	DefaultPortNumber    int `json:"portNumber,omitempty"`    // eg. 21303, udp and tcp
	DefaultBootstrapPort int `json:"bootstrapPort,omitempty"` // eg. 21000, udp
}

func LoadChainConfigFile(path string) (*ChainConfig, error) {
	cfg := new(ChainConfig)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	_, err = toml.NewDecoder(f).Decode(cfg)
	f.Close()
	if err != nil {
		return nil, err
	}
	if cfg.ChainId == nil {
		return nil, fmt.Errorf("chainId is required")
	}
	name := cfg.Name()
	if name == "" || name == "mainnet" || name == "aqua" {
		return nil, fmt.Errorf("name is required")
	}
	return cfg, nil
}

func SaveChainConfig(cfg *ChainConfig, path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists")
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	err = toml.NewEncoder(f).Encode(cfg)
	return err

}

// GetGenesisVersion returns the genesis version of the chain
//
// If 0/1, a DAG is generated for ethash
func (chainConfig *ChainConfig) GetGenesisVersion() HeaderVersion {
	return chainConfig.GetBlockVersion(common.Big0)
}

func GetChainConfig(name string) *ChainConfig {
	switch name {
	default:
		log.Warn("Unknown chain config", "name", name)
		return nil
	case "aqua", "mainnet", "aquachain":
		return MainnetChainConfig
	case "testnet":
		return TestnetChainConfig
	case "testnet2":
		return Testnet2ChainConfig
	case "testnet3":
		return Testnet3ChainConfig
	case "dev":
		return AllAquahashProtocolChanges
	case "test":
		return TestChainConfig
	case "eth":
		panic("no eth net")
	}
}

func GetChainConfigByChainId(chainid *big.Int) *ChainConfig {
	for _, v := range allChainConfigs {
		if v.ChainId.Cmp(chainid) == 0 {
			return v
		}
	}
	return nil
}

func ValidChainNames() []string {
	return []string{
		MainnetChainConfig.Name(),
		TestnetChainConfig.Name(),
		Testnet2ChainConfig.Name(),
		Testnet3ChainConfig.Name(),
		// AllAquahashProtocolChanges.Name(),
		// TestChainConfig.Name(),
	}
}
func ValidChainConfigs() []*ChainConfig {
	return []*ChainConfig{
		MainnetChainConfig,
		TestnetChainConfig,
		Testnet2ChainConfig,
		Testnet3ChainConfig,
		// AllAquahashProtocolChanges,
		// TestChainConfig,
	}
}
func AllChainConfigs() []*ChainConfig {
	return []*ChainConfig{
		MainnetChainConfig,
		TestnetChainConfig,
		Testnet2ChainConfig,
		Testnet3ChainConfig,
		AllAquahashProtocolChanges,
		TestChainConfig,
	}
}

var allChainConfigs = AllChainConfigs()

func SetAllChainConfigs(cfgs []*ChainConfig) {
	log.Warn("setting all chain configs", "count", len(cfgs))
	allChainConfigs = cfgs
}

func AddChainConfig(name string, cfg *ChainConfig) {
	if GetChainConfigByChainId(cfg.ChainId) != nil {
		panic("a chain with that chainId is already defined")
	}
	if _, ok := chainNames[name]; ok {
		panic("a chain with that name is already defined")
	}
	log.Warn("adding chain config", "name", name, "chainid", cfg.ChainId.String(), "HF", cfg.HF.String())
	allChainConfigs = append(allChainConfigs, cfg)
	newChainnames := make(map[string]*ChainConfig)
	for k, v := range chainNames {
		newChainnames[k] = v
	}
	newChainnames[name] = cfg
	chainNames = newChainnames
}

// this map might be swapped when adding custom chains
var chainNames = map[string]*ChainConfig{
	"mainnet":  MainnetChainConfig,
	"testnet":  TestnetChainConfig,
	"testnet2": Testnet2ChainConfig,
	"testnet3": Testnet3ChainConfig,
	"dev":      AllAquahashProtocolChanges,
	"test":     TestChainConfig,
}

// Name returns the name of the chain config. (mainnet, testnet, etc)
//
// This is used for '-chain <name>' flag and default datadir.
func (c *ChainConfig) Name() string {
	for k, v := range chainNames {
		if v == c {
			return k
		}
	}
	return "unknown"
}

// AquahashConfig is the consensus engine configs for proof-of-work based sealing.
type AquahashConfig struct{}

// String implements the stringer interface, returning the consensus engine details.
func (c *AquahashConfig) String() string {
	return "aquahash"
}

// CliqueConfig is the consensus engine configs for proof-of-authority based sealing.
type CliqueConfig struct {
	Period      uint64 `json:"period"`      // Number of seconds between blocks to enforce
	Epoch       uint64 `json:"epoch"`       // Epoch length to reset votes and checkpoint
	StartNumber uint64 `json:"startNumber"` // Block number to start clique engine (0 = genesis)
}

// String implements the stringer interface, returning the consensus engine details.
func (c *CliqueConfig) String() string {
	return "clique"
}

// String implements the fmt.Stringer interface.
func (c *ChainConfig) String() string {
	var engine interface{}
	switch {
	case c.Aquahash != nil:
		engine = c.Aquahash
	case c.Clique != nil:
		engine = c.Clique
	default:
		engine = "unknown"
	}
	return fmt.Sprintf("{ChainID: %v EIP150: %v EIP155: %v EIP158: %v Byzantium: %v Engine: %v}",
		c.ChainId,
		c.EIP150Block,
		c.EIP155Block,
		c.EIP158Block,
		c.ByzantiumBlock,
		engine,
	)
}

// String implements the fmt.Stringer interface.
func (c *ChainConfig) StringNoChainId() string {
	return fmt.Sprintf("EIP150: %v EIP155: %v EIP158: %v Byzantium: %v",
		c.EIP150Block,
		c.EIP155Block,
		c.EIP158Block,
		c.ByzantiumBlock,
	)
}

func (c *ChainConfig) EngineName() string {
	var engine string
	switch {
	case c.Aquahash != nil:
		engine = c.Aquahash.String()
	case c.Clique != nil:
		engine = c.Clique.String()
	default:
		engine = "unknown"
	}
	return engine
}

// IsHomestead returns whether num is either equal to the homestead block or greater.
func (c *ChainConfig) IsHomestead(num *big.Int) bool {
	return isForked(c.HomesteadBlock, num)
}

// IsDAO returns whether num is either equal to the DAO fork block or greater.
func (c *ChainConfig) IsDAOFork(num *big.Int) bool {
	return isForked(c.DAOForkBlock, num)
}

func (c *ChainConfig) IsEIP150(num *big.Int) bool {
	return isForked(c.EIP150Block, num)
}

func (c *ChainConfig) IsEIP155(num *big.Int) bool {
	return isForked(c.EIP155Block, num)
}

func (c *ChainConfig) IsEIP158(num *big.Int) bool {
	return isForked(c.EIP158Block, num)
}

func (c *ChainConfig) IsByzantium(num *big.Int) bool {
	return isForked(c.ByzantiumBlock, num)
}

func (c *ChainConfig) IsConstantinople(num *big.Int) bool {
	return isForked(c.ConstantinopleBlock, num)
}

// GasTable returns the gas table corresponding to the current phase.
//
// The returned GasTable's fields shouldn't, under any circumstances, be changed.
func (c *ChainConfig) GasTable(num *big.Int) GasTable {
	if num == nil {
		return GasTableHomestead
	}
	switch {
	case c.IsHF(1, num):
		return GasTableHF1
	default:
		return GasTableHomestead
	}
}

// CheckCompatible checks whether scheduled fork transitions have been imported
// with a mismatching chain configuration.
func (c *ChainConfig) CheckCompatible(newcfg *ChainConfig, height uint64) *ConfigCompatError {
	bhead := new(big.Int).SetUint64(height)

	// Iterate checkCompatible to find the lowest conflict.
	var lasterr *ConfigCompatError
	for {
		err := c.checkCompatible(newcfg, bhead)
		if err == nil || (lasterr != nil && err.RewindTo == lasterr.RewindTo) {
			break
		}
		lasterr = err
		bhead.SetUint64(err.RewindTo)
	}
	return lasterr
}

func (c *ChainConfig) checkCompatible(newcfg *ChainConfig, head *big.Int) *ConfigCompatError {
	for i := 1; i < KnownHF; i++ {
		if c.HF[i] == nil && newcfg.HF[i] == nil {
			continue
		}
		if isForkIncompatible(c.HF[i], newcfg.HF[i], head) {
			return newCompatError(fmt.Sprintf("Aquachain HF%v block", i), c.HF[i], newcfg.HF[i])
		}
	}
	if isForkIncompatible(c.HomesteadBlock, newcfg.HomesteadBlock, head) {
		return newCompatError("Homestead fork block", c.HomesteadBlock, newcfg.HomesteadBlock)
	}
	if isForkIncompatible(c.DAOForkBlock, newcfg.DAOForkBlock, head) {
		return newCompatError("DAO fork block", c.DAOForkBlock, newcfg.DAOForkBlock)
	}
	if c.IsDAOFork(head) && c.DAOForkSupport != newcfg.DAOForkSupport {
		return newCompatError("DAO fork support flag", c.DAOForkBlock, newcfg.DAOForkBlock)
	}
	if isForkIncompatible(c.EIP150Block, newcfg.EIP150Block, head) {
		return newCompatError("EIP150 fork block", c.EIP150Block, newcfg.EIP150Block)
	}
	if isForkIncompatible(c.EIP155Block, newcfg.EIP155Block, head) {
		return newCompatError("EIP155 fork block", c.EIP155Block, newcfg.EIP155Block)
	}
	if isForkIncompatible(c.EIP158Block, newcfg.EIP158Block, head) {
		return newCompatError("EIP158 fork block", c.EIP158Block, newcfg.EIP158Block)
	}
	if c.IsEIP158(head) && !configNumEqual(c.ChainId, newcfg.ChainId) {
		return newCompatError("EIP158 chain ID", c.EIP158Block, newcfg.EIP158Block)
	}
	if isForkIncompatible(c.ByzantiumBlock, newcfg.ByzantiumBlock, head) {
		return newCompatError("Byzantium fork block", c.ByzantiumBlock, newcfg.ByzantiumBlock)
	}
	if isForkIncompatible(c.ConstantinopleBlock, newcfg.ConstantinopleBlock, head) {
		return newCompatError("Constantinople fork block", c.ConstantinopleBlock, newcfg.ConstantinopleBlock)
	}
	return nil
}

// isForkIncompatible returns true if a fork scheduled at s1 cannot be rescheduled to
// block s2 because head is already past the fork.
func isForkIncompatible(s1, s2, head *big.Int) bool {
	return (isForked(s1, head) || isForked(s2, head)) && !configNumEqual(s1, s2)
}

// isForked returns whether a fork scheduled at block s is active at the given head block.
func isForked(s, head *big.Int) bool {
	if s == nil || head == nil {
		return false
	}
	return s.Cmp(head) <= 0
}

func configNumEqual(x, y *big.Int) bool {
	if x == nil {
		return y == nil
	}
	if y == nil {
		return x == nil
	}
	return x.Cmp(y) == 0
}

// ConfigCompatError is raised if the locally-stored blockchain is initialised with a
// ChainConfig that would alter the past.
type ConfigCompatError struct {
	What string
	// block numbers of the stored and new configurations
	StoredConfig, NewConfig *big.Int
	// the block number to which the local chain must be rewound to correct the error
	RewindTo uint64
}

func newCompatError(what string, storedblock, newblock *big.Int) *ConfigCompatError {
	var rew *big.Int
	switch {
	case storedblock == nil:
		rew = newblock
	case newblock == nil || storedblock.Cmp(newblock) < 0:
		rew = storedblock
	default:
		rew = newblock
	}
	err := &ConfigCompatError{what, storedblock, newblock, 0}
	if rew != nil && rew.Sign() > 0 {
		err.RewindTo = rew.Uint64() - 1
	}
	return err
}

func (err *ConfigCompatError) Error() string {
	return fmt.Sprintf("mismatching %s in database (have %d, want %d, rewindto %d)", err.What, err.StoredConfig, err.NewConfig, err.RewindTo)
}

// Rules wraps ChainConfig and is merely syntatic sugar or can be used for functions
// that do not have or require information about the block.
//
// Rules is a one time interface meaning that it shouldn't be used in between transition
// phases.
type Rules struct {
	ChainId                                   *big.Int
	IsHomestead, IsEIP150, IsEIP155, IsEIP158 bool
	IsByzantium                               bool
}

func (c *ChainConfig) Rules(num *big.Int) Rules {
	chainId := c.ChainId
	if chainId == nil {
		chainId = new(big.Int)
	}
	return Rules{ChainId: new(big.Int).Set(chainId), IsHomestead: c.IsHomestead(num), IsEIP150: c.IsEIP150(num), IsEIP155: c.IsEIP155(num), IsEIP158: c.IsEIP158(num), IsByzantium: c.IsByzantium(num)}
}
