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

// Package aquaapi implements the general Aquachain API functions.
package aquaapi

import (
	"context"
	"math/big"

	"gitlab.com/aquachain/aquachain"
	"gitlab.com/aquachain/aquachain/aqua/accounts"
	"gitlab.com/aquachain/aquachain/aqua/event"
	"gitlab.com/aquachain/aquachain/aquadb"
	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/core"
	"gitlab.com/aquachain/aquachain/core/state"
	"gitlab.com/aquachain/aquachain/core/types"
	"gitlab.com/aquachain/aquachain/core/vm"
	"gitlab.com/aquachain/aquachain/params"
	"gitlab.com/aquachain/aquachain/rpc"
)

// Backend interface provides the common API services (that are provided by
// both full and light clients) with access to necessary functions.
type Backend interface {
	// General Aquachain API
	SyncProgress() aquachain.SyncProgress
	ProtocolVersion() int
	SuggestPrice(ctx context.Context) (*big.Int, error)
	ChainDb() aquadb.Database
	EventMux() *event.TypeMux
	AccountManager() *accounts.Manager

	// BlockChain API
	SetHead(number uint64)
	HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error)
	BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error)
	StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error)
	GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error)
	GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error)
	GetTd(blockHash common.Hash) *big.Int
	GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error)
	SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription
	SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription
	SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription

	// TxPool API
	SendTx(ctx context.Context, signedTx *types.Transaction) error
	GetPoolTransactions() (types.Transactions, error)
	GetPoolTransaction(txHash common.Hash) *types.Transaction
	GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error)
	Stats() (pending int, queued int)
	TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions)
	SubscribeTxPreEvent(chan<- core.TxPreEvent) event.Subscription

	ChainConfig() *params.ChainConfig
	CurrentBlock() *types.Block
}

func GetAPIs(apiBackend Backend) []rpc.API {
	nonceLock := new(AddrLocker)
	return []rpc.API{
		{
			Namespace: "aqua",
			Version:   "1.0",
			Service:   NewPublicAquachainAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "aqua",
			Version:   "1.0",
			Service:   NewPublicBlockChainAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "aqua",
			Version:   "1.0",
			Service:   NewPublicTransactionPoolAPI(apiBackend, nonceLock, false),
			Public:    true,
		}, {
			Namespace: "txpool",
			Version:   "1.0",
			Service:   NewPublicTxPoolAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(apiBackend),
		}, {
			Namespace: "aqua",
			Version:   "1.0",
			Service:   NewPublicAccountAPI(apiBackend.AccountManager()), // "aqua.accounts"
			Public:    true,
		}, {
			Namespace: "personal",
			Version:   "1.0",
			Service:   NewPrivateAccountAPI(apiBackend, nonceLock),
			Public:    false,
			// }, { // eth alias
			// 	Namespace: "eth",
			// 	Version:   "1.0",
			// 	Service:   NewPublicAquachainAPI(apiBackend),
			// 	Public:    true,
			// }, {
			// 	Namespace: "eth",
			// 	Version:   "1.0",
			// 	Service:   NewPublicBlockChainAPI(apiBackend),
			// 	Public:    true,
			// }, {
			// 	Namespace: "eth",
			// 	Version:   "1.0",
			// 	Service:   NewPublicTransactionPoolAPI(apiBackend, nonceLock, false),
			// 	Public:    true,
			// }, {
			// 	Namespace: "eth",
			// 	Version:   "1.0",
			// 	Service:   NewPublicAccountAPI(apiBackend.AccountManager()),
			// 	Public:    true,
		}, { // btcrpc
			Namespace: "btc",
			Version:   "1.0",
			Service:   NewPublicBitcoinAPI(NewPublicBlockChainAPI(apiBackend)), // TODO: fix
		},
	}
}
