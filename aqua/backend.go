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

// Package aqua implements the Aquachain protocol.
package aqua

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"gitlab.com/aquachain/aquachain/aqua/accounts"
	"gitlab.com/aquachain/aquachain/aqua/downloader"
	"gitlab.com/aquachain/aquachain/aqua/event"
	"gitlab.com/aquachain/aquachain/aqua/filters"
	"gitlab.com/aquachain/aquachain/aqua/gasprice"
	"gitlab.com/aquachain/aquachain/aquadb"
	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/config"
	"gitlab.com/aquachain/aquachain/common/hexutil"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/consensus"
	"gitlab.com/aquachain/aquachain/consensus/aquahash"
	"gitlab.com/aquachain/aquachain/consensus/clique"
	"gitlab.com/aquachain/aquachain/core"
	"gitlab.com/aquachain/aquachain/core/bloombits"
	"gitlab.com/aquachain/aquachain/core/types"
	"gitlab.com/aquachain/aquachain/core/vm"
	"gitlab.com/aquachain/aquachain/internal/aquaapi"
	"gitlab.com/aquachain/aquachain/node"

	"gitlab.com/aquachain/aquachain/opt/miner"
	"gitlab.com/aquachain/aquachain/p2p"
	"gitlab.com/aquachain/aquachain/params"
	"gitlab.com/aquachain/aquachain/rlp"
	"gitlab.com/aquachain/aquachain/rpc"
)

// Aquachain implements the Aquachain full node service.
type Aquachain struct {
	ctx         context.Context // TODO use
	config      *config.Aquaconfig
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan  chan bool    // Channel for shutting down the aquachain
	stopDbUpgrade func() error // stop chain db sequential key upgrade

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager

	// DB interfaces
	chainDb aquadb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	ApiBackend *AquaApiBackend

	miner    *miner.Miner
	gasPrice *big.Int
	aquabase common.Address

	netRPCService *aquaapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and aquabase)
}

// New creates a new Aquachain object (including the
// initialisation of the common Aquachain object)
func New(ctx context.Context, nodectx *node.ServiceContext, config *config.Aquaconfig, nodename string) (*Aquachain, error) {
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	if strings.Count(nodename, "/") < 3 && !strings.HasPrefix(nodename, "test") {
		return nil, fmt.Errorf("invalid node name %s, this indicates a bad compilation mode. please report bug", nodename)
	}
	log.Info("Node name", "name", nodename)
	config.SetNodeName(nodename)
	chainDb, err := CreateDB(nodectx, config, "chaindata")
	if err != nil {
		return nil, fmt.Errorf("creating db: %w", err)
	}
	stopDbUpgrade := upgradeDeduplicateData(chainDb)
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "HF-Ready", chainConfig.HF, "config", chainConfig)

	if config.ChainId != chainConfig.ChainId.Uint64() {
		return nil, fmt.Errorf("ChainID mismatch: configured %d, chain %d", config.ChainId, chainConfig.ChainId)
	}

	aqua := &Aquachain{
		ctx:            ctx,
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       nodectx.EventMux,
		accountManager: nodectx.AccountManager,
		engine:         CreateConsensusEngine(nodectx, config.Aquahash, chainConfig, chainDb, nodename),
		shutdownChan:   make(chan bool),
		stopDbUpgrade:  stopDbUpgrade,
		gasPrice:       new(big.Int).SetUint64(config.GasPrice),
		aquabase:       config.Aquabase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainConfig, chainDb, params.BloomBitsBlocks),
	}

	// if chainConfig == params.EthnetChainConfig {
	// 	ProtocolName = "eth"
	// 	ProtocolVersions = []uint{63, 62}
	// 	ProtocolLengths = []uint64{17, 8}
	// }

	log.Info("Initialising Aquachain protocol", "versions", ProtocolVersions, "network", config.ChainId)

	//if !config.SkipBcVersionCheck {
	bcVersion := core.GetBlockChainVersion(chainDb)
	if bcVersion != core.BlockChainVersion && bcVersion != 0 {
		return nil, fmt.Errorf("blockchain DB version mismatch (%d / %d). Run aquachain upgradedb", bcVersion, core.BlockChainVersion)
	}
	core.WriteBlockChainVersion(chainDb, core.BlockChainVersion)
	//}
	var (
		vmConfig    = vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
		cacheConfig = &core.CacheConfig{Disabled: config.NoPruning, TrieNodeLimit: config.TrieCache, TrieTimeLimit: config.TrieTimeout}
	)
	aqua.blockchain, err = core.NewBlockChain(ctx, chainDb, cacheConfig, aqua.chainConfig, aqua.engine, vmConfig)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		aqua.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	aqua.bloomIndexer.Start(aqua.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = nodectx.ResolvePath(config.TxPool.Journal)
	}
	aqua.txPool = core.NewTxPool(config.TxPool, aqua.chainConfig, aqua.blockchain)

	if aqua.protocolManager, err = NewProtocolManager(aqua.chainConfig, config.SyncMode, config.ChainId, aqua.eventMux, aqua.txPool, aqua.engine, aqua.blockchain, chainDb); err != nil {
		return nil, err
	}
	aqua.miner = miner.New(aqua, aqua.chainConfig, aqua.EventMux(), aqua.engine)
	aqua.miner.SetExtra(makeExtraData(config.ExtraData))

	aqua.ApiBackend = &AquaApiBackend{aqua, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = new(big.Int).SetUint64(config.GasPrice)
	}
	aqua.ApiBackend.gpo = gasprice.NewOracle(aqua.ApiBackend, gpoParams)

	return aqua, nil
}

func makeExtraData(extra []byte) []byte {
	// create default extradata

	thing := []interface{}{
		uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
		"aqua",
		runtime.GOOS,
		common.ShortGoVersion(),
	}
	defaultExtra, _ := rlp.EncodeToBytes(thing)
	if len(extra) == 0 {
		extra = defaultExtra
	} else if len(extra) > 1 {
		if extra[0] == '0' && extra[1] == 'x' {
			b, err := hex.DecodeString(string(extra[2:]))
			if err != nil {
				log.Warn("Ignoring custom extradata", "error", err)
				extra = defaultExtra
			} else if len(b) > 32 {
				log.Warn("Ignoring too-big extradata", "len", len(b))
				extra = defaultExtra
			} else {
				extra = b
			}
		}
	}

	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Extra invalid:", "extra", thing[1], "name", thing[2], "version", thing[3])
		extra = extra[:params.MaximumExtraDataSize-1]
		log.Warn("Miner extra data exceed limit, truncating!", "extra", hexutil.Bytes(extra).String(), "limit", params.MaximumExtraDataSize)
	}
	return extra
}

func DecodeExtraData(extra []byte) (version [3]uint8, osname string, extradata []byte, err error) {
	var (
		v                   []interface{}
		major, minor, patch uint8
	)
	err = rlp.DecodeBytes(extra, &v)
	if err != nil {
		return version, osname, extra, err
	}
	if len(v) < 3 {
		log.Info("Extra data invalid", "len", len(v))
		return version, osname, extra, nil
	}
	// extract version
	vr, ok := v[0].([]uint8)
	if !ok || len(vr) != 3 {
		// fmt.Printf("%T type, len %v\n", v[0], len(vr))
		err = fmt.Errorf("could not decode version")
		return version, osname, extra, err
	}
	major, minor, patch = vr[0], vr[1], vr[2]
	version = [3]uint8{major, minor, patch}
	if osnameBytes, ok := v[2].([]byte); ok {
		osname = string(osnameBytes)
	}

	if len(v) > 3 {
		if extradataBytes, ok := v[3].([]byte); ok {
			extradata = extradataBytes
		}
	}
	return version, osname, extradata, nil
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *config.Aquaconfig, name string) (aquadb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*aquadb.LDBDatabase); ok {
		db.Meter("aqua/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an Aquachain service
func CreateConsensusEngine(ctx *node.ServiceContext, config *aquahash.Config, chainConfig *params.ChainConfig, db aquadb.Database, nodename string) consensus.Engine {
	startVersion := chainConfig.GetGenesisVersion()
	switch {
	case config.PowMode == aquahash.ModeFake:
		log.Warn("Aquahash used in fake mode")
		return aquahash.NewFaker()
	case config.PowMode == aquahash.ModeTest:
		log.Warn("Aquahash used in test mode")
		return aquahash.NewTester()
	case config.PowMode == aquahash.ModeShared:
		log.Warn("Aquahash used in shared mode")
		return aquahash.NewSharedTesting()
	case chainConfig.Clique != nil:
		log.Info("Starting Clique", "period", chainConfig.Clique.Period, "epoch", chainConfig.Clique.Epoch)
		return clique.New(chainConfig.Clique, db)
	default:
		log.Info("Starting aquahash", "version", startVersion)
		if startVersion > 1 {
			engine := aquahash.New(&aquahash.Config{StartVersion: startVersion})
			engine.SetThreads(-1)
			return engine
		}
		engine := aquahash.New(&aquahash.Config{
			CacheDir:       ctx.ResolvePath(config.CacheDir),
			CachesInMem:    config.CachesInMem,
			CachesOnDisk:   config.CachesOnDisk,
			DatasetDir:     config.DatasetDir,
			DatasetsInMem:  config.DatasetsInMem,
			DatasetsOnDisk: config.DatasetsOnDisk,
			StartVersion:   startVersion,
		})
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs returns the collection of RPC services the aquachain package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Aquachain) APIs() []rpc.API {

	apis := aquaapi.GetAPIs(s.ApiBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)
	if s.protocolManager != nil {
		apis = append(apis, rpc.API{
			Namespace: "aqua",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		})
	}

	// Append all the local APIs and return
	x := append(apis, []rpc.API{
		{
			Namespace: "aqua",
			Version:   "1.0",
			Service:   NewPublicAquachainAPI(s),
			Public:    true,
		}, {
			Namespace: "aqua",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "aqua",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "testing",
			Version:   "1.0",
			Service:   NewPublicTestingAPI(s.chainConfig, s, s.config.GetNodeName()),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
	log.Info("APIs", "count", len(x))
	return x
}

func (s *Aquachain) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *Aquachain) Aquabase() (eb common.Address, err error) {
	s.lock.RLock()
	aquabase := s.aquabase
	s.lock.RUnlock()

	if aquabase != (common.Address{}) {
		return aquabase, nil
	}
	am := s.AccountManager()
	if am == nil {
		return common.Address{}, fmt.Errorf("aquabase must be explicitly specified (no-keybase mode)")
	}
	if wallets := am.Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			aquabase := accounts[0].Address

			s.lock.Lock()
			s.aquabase = aquabase
			s.lock.Unlock()

			log.Info("Aquabase automatically configured", "address", aquabase)
			return aquabase, nil
		}
	}
	return common.Address{}, fmt.Errorf("aquabase must be explicitly specified")
}

// set in js console via admin interface or wrapper from cli flags
func (self *Aquachain) SetAquabase(aquabase common.Address) {
	self.lock.Lock()
	self.aquabase = aquabase
	self.lock.Unlock()

	self.miner.SetAquabase(aquabase)
}

func (s *Aquachain) StartMining(local bool) error {
	eb, err := s.Aquabase()
	if err != nil {
		log.Error("Cannot start mining without aquabase", "err", err)
		return fmt.Errorf("aquabase missing: %v", err)
	}
	if cliquecfg, ok := s.engine.(*clique.Clique); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("Aquabase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		type xt interface {
			CliqueSigner() clique.SignerFn
		}
		cliquecfg.Authorize(eb, wallet.(xt).CliqueSigner())
	}
	if local {
		// If local (CPU) mining is started, we can disable the transaction rejection
		// mechanism introduced to speed sync times. CPU mining on mainnet is ludicrous
		// so noone will ever hit this path, whereas marking sync done on CPU mining
		// will ensure that private networks work in single miner mode too.

		if s.protocolManager != nil { // offline mode
			atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)
		}
	}
	go s.miner.Start(eb)
	return nil
}

func (s *Aquachain) StopMining()         { s.miner.Stop() }
func (s *Aquachain) IsMining() bool      { return s.miner.Mining() }
func (s *Aquachain) Miner() *miner.Miner { return s.miner }

func (s *Aquachain) AccountManager() *accounts.Manager { return s.accountManager }
func (s *Aquachain) BlockChain() *core.BlockChain      { return s.blockchain }
func (s *Aquachain) TxPool() *core.TxPool              { return s.txPool }
func (s *Aquachain) EventMux() *event.TypeMux          { return s.eventMux }
func (s *Aquachain) Engine() consensus.Engine          { return s.engine }
func (s *Aquachain) ChainDb() aquadb.Database          { return s.chainDb }
func (s *Aquachain) IsListening() bool                 { return true } // Always listening
func (s *Aquachain) AquaVersion() int {
	if s.protocolManager != nil {
		return int(s.protocolManager.SubProtocols[0].Version)
	}
	log.Warn("Aquachain protocol manager not available, no aqua protocol version")
	return 0
}
func (s *Aquachain) NetVersion() uint64 { return s.config.ChainId }
func (s *Aquachain) Downloader() *downloader.Downloader {
	if s.protocolManager == nil {
		return nil
	}
	return s.protocolManager.downloader
}

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *Aquachain) Protocols() []p2p.Protocol {
	if s.protocolManager == nil {
		return nil
	}
	return s.protocolManager.SubProtocols
}

// Start implements node.Service, starting all internal goroutines needed by the
// Aquachain protocol implementation.
func (s *Aquachain) Start(srvr *p2p.Server) error {
	log.Info("Starting Aquachain protocol", "network", s.config.ChainId, "version", s.AquaVersion())
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers()

	// Start the RPC service
	s.netRPCService = aquaapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	// Start the networking layer
	if s.protocolManager != nil {
		s.protocolManager.Start(maxPeers)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Aquachain protocol.
func (s *Aquachain) Stop() error {
	log.Info("Shutdown: Aquachain backend service stopping")
	defer func() {
		if err := recover(); err != nil {
			log.Error("Aquachain backend service stopped with panic", "err", err)
			return
		}
		log.Info("Shutdown: Aquachain backend service stopped")
	}()
	if s.stopDbUpgrade != nil {
		s.stopDbUpgrade()
	}
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	if s.protocolManager != nil {
		s.protocolManager.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
