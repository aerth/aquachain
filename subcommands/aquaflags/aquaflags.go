// all the flags
package aquaflags

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/aqua"
	"gitlab.com/aquachain/aquachain/common/metrics"
	"gitlab.com/aquachain/aquachain/common/sense"
	"gitlab.com/aquachain/aquachain/core"
	"gitlab.com/aquachain/aquachain/core/state"
	"gitlab.com/aquachain/aquachain/node"
	"gitlab.com/aquachain/aquachain/p2p"
	"gitlab.com/aquachain/aquachain/params"
	"gitlab.com/aquachain/aquachain/subcommands/buildinfo"
)

// type DirectoryFlag = utils.DirectoryFlag // TODO remove these type aliases
// type DirectoryString = utils.DirectoryString

// These are all the command line flags we support.
// If you add to this list, please remember to include the
// flag in the appropriate command definition.
//
// The flags are defined here so their names and help texts
// are the same for all commands.

var (
	// General settings

	// General settings
	// DataDirFlag = &cli.StringFlag{
	// 	Name:  "datadir",
	// 	Usage: "Data directory for the databases, IPC socket, and keystore (also see -keystore flag)",
	// 	Value: NewDirectoryString(node.DefaultDatadir()),
	// }
	DataDirFlag = &cli.StringFlag{
		Name:  "datadir",
		Usage: "Data directory for the databases, IPC socket, and keystore (also see -keystore flag)",
		Value: node.DefaultDatadir(),
		Action: func(ctx context.Context, cmd *cli.Command, v string) error {
			if v == "" {
				return fmt.Errorf("invalid directory: %q", v)
			}
			stat, err := os.Stat(v)
			if err != nil {
				return err
			}
			if !stat.IsDir() {
				return fmt.Errorf("invalid directory: %q", v)
			}
			return nil
		},
		Category:  "AQUACHAIN",
		Validator: checkStringFlag,
	}
	KeyStoreDirFlag = &cli.StringFlag{
		Name:      "keystore",
		Usage:     "Directory for the keystore (default = inside the datadir)",
		Action:    checkDirectoryFlag,
		Category:  "AQUACHAIN",
		Validator: checkStringFlag,
	}
	UseUSBFlag = &cli.BoolFlag{
		Name:     "usb",
		Usage:    "Enables USB hardware wallet support (disabled in pure-go builds)",
		Category: "UNSTABLE",
		Hidden:   !buildinfo.CGO,
	}
	DoitNowFlag = &cli.BoolFlag{
		Name:  "now",
		Usage: "Start the node immediately, do not start countdown",
		Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
			if v {
				cmd.Set("now", "true")
				p2p.NoCountdown = true
				node.NoCountdown = true
			}
			return nil
		},
		Category: "AQUACHAIN",
	}
	ChainFlag = &cli.StringFlag{
		Name:  "chain",
		Usage: "Chain select (aqua, testnet, testnet2, testnet3)",
		Value: "aqua",
		Action: func(ctx context.Context, cmd *cli.Command, v string) error {
			cfg := params.GetChainConfig(v)
			if cfg == nil {
				return fmt.Errorf("invalid chain name: %q, try one of %q", v, params.ValidChainNames())
			}
			// cmd.Set("chain", v)
			return nil
		},
		Category:  "AQUACHAIN",
		Validator: checkStringFlag,
	}
	AlertModeFlag = &cli.BoolFlag{
		Name:     "alerts",
		Usage:    "Enable alert notifications (requires env $ALERT_TOKEN, $ALERT_PLATFORM, and $ALERT_CHANNEL)",
		Category: "AQUACHAIN",
	}
	DeveloperFlag = &cli.BoolFlag{
		Name:  "dev",
		Usage: "Ephemeral proof-of-authority network with a pre-funded developer account, mining enabled",
		Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
			return fmt.Errorf("flag %q is deprecated, use '-chain dev'", "dev")
		},
		Hidden:   true,
		Category: "DEPRECATED",
	}
	DeveloperPeriodFlag = &cli.IntFlag{
		Name:  "dev.period",
		Usage: "Block period to use in '-chain dev' mode (0 = mine only if transaction pending)",

		Category: "TESTING",
	}
	IdentityFlag = &cli.StringFlag{
		Name:      "identity",
		Usage:     "Custom node name (used in p2p networking, eg: \"CoolPool\", becomes \"Aquachain-CoolPool/v1.2.3-release/linux-amd64/go.1.24.1\")",
		Validator: checkStringFlag,
		Category:  "AQUACHAIN",
	}
	WorkingDirectoryFlag = &cli.StringFlag{
		Name: "WorkingDirectory",
		Action: func(ctx context.Context, cmd *cli.Command, v string) error {
			if v != "" {
				return fmt.Errorf("flag %q is deprecated, use '-jspath <path>'", "WorkingDirectory")
			}
			return nil
		},
		Hidden:   true,
		Category: "RPC CLIENT",
	}
	JavascriptDirectoryFlag = &cli.StringFlag{
		Name:      "jspath",
		TakesFile: true,
		Usage:     "Working directory for importing JS files into console (default = current directory)",
		Value:     ".",
		Action: func(ctx context.Context, cmd *cli.Command, v string) error {
			if v == "none" {
				return nil
			}
			if v == "" {
				return fmt.Errorf("invalid directory: %q", v)
			}
			if v == "." {
				return nil
			}
			stat, err := os.Stat(v)
			if err != nil {
				return err
			}
			if !stat.IsDir() {
				return fmt.Errorf("invalid directory: %q", v)
			}
			return nil
		},
		Validator: checkStringFlag,
		Category:  "RPC CLIENT",
	}
	SyncModeFlag = &cli.StringFlag{
		Name:  "syncmode",
		Usage: `Blockchain sync mode ("fast", "full")`,
		Value: "full",
		Action: func(ctx context.Context, cmd *cli.Command, v string) error {
			if v != "fast" && v != "full" && v != "offline" {
				return fmt.Errorf("invalid sync mode: %q", v)
			}
			return checkStringFlag(v)
		},
		Validator: checkStringFlag,
		Category:  "SYNC",
	}

	GCModeFlag = &cli.StringFlag{
		Name:  "gcmode",
		Usage: `Garbage collection mode to use, either "full" (full gc) or "archive" (disable). Default is "archive" for full accurate state (for example, 'admin.supply')`,
		Value: "archive",
		Validator: func(gcmode string) error {
			if gcmode != "full" && gcmode != "archive" {
				return fmt.Errorf("invalid gcmode: %q (try archive or full)", gcmode)
			}
			return nil
		},
		Category: "SYNC",
	}
)

func checkStringFlag(v string) error {
	if strings.HasPrefix(v, "-") {
		return fmt.Errorf("uh oh, flag value looks like a flag: %q", v)
	}
	return nil
}

func checkDirectoryFlag(ctx context.Context, cmd *cli.Command, v string) error {
	if v == "" {
		return fmt.Errorf("invalid directory: %q", v)
	}
	if err := checkStringFlag(v); err != nil {
		return err
	}
	stat, err := os.Stat(v)
	if err != nil {
		return err
	}
	if !stat.IsDir() {
		return fmt.Errorf("invalid directory: %q", v)
	}
	return nil
}

var (
	// Aquahash settings
	AquahashCacheDirFlag = &cli.StringFlag{
		Name:      "aquahash.cachedir",
		Usage:     "Directory to store the aquahash v1 DAG verification caches for blocks 1-22800 (default = inside the datadir)",
		Category:  "PERFORMANCE",
		Action:    checkDirectoryFlag,
		Validator: checkStringFlag,
	}
	AquahashCachesInMemoryFlag = &cli.IntFlag{
		Name:     "aquahash.cachesinmem",
		Usage:    "Number of recent aquahash caches to keep in memory (16MB each)",
		Value:    0,
		Category: "PERFORMANCE",
	}
	AquahashCachesOnDiskFlag = &cli.IntFlag{
		Name:     "aquahash.cachesondisk",
		Usage:    "Number of recent aquahash caches to keep on disk (16MB each)",
		Value:    0,
		Category: "PERFORMANCE",
	}
	AquahashDatasetDirFlag = &cli.StringFlag{
		Name:      "aquahash.dagdir",
		Usage:     "Directory to store the aquahash mining DAGs (default = inside home folder)",
		Value:     aqua.DefaultConfig.Aquahash.DatasetDir,
		Action:    checkDirectoryFlag,
		Category:  "PERFORMANCE",
		Validator: checkStringFlag,
	}
	AquahashDatasetsInMemoryFlag = &cli.IntFlag{
		Name:     "aquahash.dagsinmem",
		Usage:    "Number of recent aquahash mining DAGs to keep in memory (1+GB each)",
		Value:    int64(aqua.DefaultConfig.Aquahash.DatasetsInMem),
		Category: "PERFORMANCE",
	}
	AquahashDatasetsOnDiskFlag = &cli.IntFlag{
		Name:     "aquahash.dagsondisk",
		Usage:    "Number of recent aquahash mining DAGs to keep on disk (1+GB each)",
		Value:    int64(aqua.DefaultConfig.Aquahash.DatasetsOnDisk),
		Category: "PERFORMANCE",
	}
	// Transaction pool settings
	TxPoolNoLocalsFlag = &cli.BoolFlag{
		Name:     "txpool.nolocals",
		Usage:    "Disables price exemptions for locally submitted transactions",
		Category: "TRANSACTION POOL",
	}
	TxPoolJournalFlag = &cli.StringFlag{
		Name:      "txpool.journal",
		Usage:     "Disk journal for local transaction to survive node restarts",
		Value:     core.DefaultTxPoolConfig.Journal,
		Category:  "TRANSACTION POOL",
		Validator: checkStringFlag,
	}
	TxPoolRejournalFlag = &cli.DurationFlag{
		Name:     "txpool.rejournal",
		Usage:    "Time interval to regenerate the local transaction journal",
		Value:    core.DefaultTxPoolConfig.Rejournal,
		Category: "TRANSACTION POOL",
	}
	TxPoolPriceLimitFlag = &cli.UintFlag{
		Name:     "txpool.pricelimit",
		Usage:    "Minimum gas price limit to enforce for acceptance into the pool",
		Value:    aqua.DefaultConfig.TxPool.PriceLimit,
		Category: "TRANSACTION POOL",
	}
	TxPoolPriceBumpFlag = &cli.UintFlag{
		Name:     "txpool.pricebump",
		Usage:    "Price bump percentage to replace an already existing transaction",
		Value:    aqua.DefaultConfig.TxPool.PriceBump,
		Category: "TRANSACTION POOL",
	}
	TxPoolAccountSlotsFlag = &cli.UintFlag{
		Name:     "txpool.accountslots",
		Usage:    "Minimum number of executable transaction slots guaranteed per account",
		Value:    aqua.DefaultConfig.TxPool.AccountSlots,
		Category: "TRANSACTION POOL",
	}
	TxPoolGlobalSlotsFlag = &cli.UintFlag{
		Name:     "txpool.globalslots",
		Usage:    "Maximum number of executable transaction slots for all accounts",
		Value:    aqua.DefaultConfig.TxPool.GlobalSlots,
		Category: "TRANSACTION POOL",
	}
	TxPoolAccountQueueFlag = &cli.UintFlag{
		Name:     "txpool.accountqueue",
		Usage:    "Maximum number of non-executable transaction slots permitted per account",
		Value:    aqua.DefaultConfig.TxPool.AccountQueue,
		Category: "TRANSACTION POOL",
	}
	TxPoolGlobalQueueFlag = &cli.UintFlag{
		Name:     "txpool.globalqueue",
		Usage:    "Maximum number of non-executable transaction slots for all accounts",
		Value:    aqua.DefaultConfig.TxPool.GlobalQueue,
		Category: "TRANSACTION POOL",
	}
	TxPoolLifetimeFlag = &cli.DurationFlag{
		Name:     "txpool.lifetime",
		Usage:    "Maximum amount of time non-executable transaction are queued",
		Value:    aqua.DefaultConfig.TxPool.Lifetime,
		Category: "TRANSACTION POOL",
	}
	// Performance tuning settings
	CacheFlag = &cli.IntFlag{
		Name:     "cache",
		Usage:    "Megabytes of memory allocated to internal caching (consider 2048)",
		Value:    1024,
		Category: "PERFORMANCE",
	}
	CacheDatabaseFlag = &cli.IntFlag{
		Name:     "cache.database",
		Usage:    "Percentage of cache memory allowance to use for database io",
		Value:    75,
		Category: "PERFORMANCE",
	}
	CacheGCFlag = &cli.IntFlag{
		Name:     "cache.gc",
		Usage:    "Percentage of cache memory allowance to use for trie pruning",
		Value:    25,
		Category: "PERFORMANCE",
	}
	TrieCacheGenFlag = &cli.IntFlag{
		Name:     "trie-cache-gens",
		Usage:    "Number of trie node generations to keep in memory",
		Value:    int64(state.MaxTrieCacheGen),
		Category: "PERFORMANCE",
	}
	// Miner settings
	MiningEnabledFlag = &cli.BoolFlag{
		Name:     "mine",
		Usage:    "Enable mining (not optimized, not recommended for mainnet)",
		Category: "MINING",
	}
	MinerThreadsFlag = &cli.IntFlag{
		Name:     "minerthreads",
		Usage:    "Number of CPU threads to use for mining",
		Value:    int64(runtime.NumCPU()),
		Category: "MINING",
	}
	TargetGasLimitFlag = &cli.UintFlag{
		Name:        "targetgaslimit",
		Usage:       "Target gas limit sets the artificial target gas floor for the blocks to mine",
		Value:       params.GenesisGasLimit,
		Destination: &params.TargetGasLimit,
		Category:    "TRANSACTION POOL",
	}
	AquabaseFlag = &cli.StringFlag{
		Name:      "aquabase",
		Usage:     "Public address for block mining rewards (default = first account created)",
		Value:     "0",
		Category:  "AQUACHAIN",
		Validator: checkStringFlag,
	}
	GasPriceFlag = &cli.UintFlag{
		Name:     "gasprice",
		Usage:    "Minimal gas price to accept for mining a transactions",
		Value:    aqua.DefaultConfig.GasPrice,
		Category: "TRANSACTION POOL",
	}
	ExtraDataFlag = &cli.StringFlag{
		Name:      "extradata",
		Usage:     "Block extra data set by the miner (default = client version)",
		Category:  "MINING",
		Validator: checkStringFlag,
	}
	// Account settings
	UnlockedAccountFlag = &cli.StringFlag{
		Name:      "unlock",
		Usage:     "Comma separated list of accounts to unlock (CAREFUL!)",
		Value:     "",
		Category:  "AQUACHAIN",
		Validator: checkStringFlag,
	}
	PasswordFileFlag = &cli.StringFlag{
		Name:     "password",
		Usage:    "Password file to use for non-interactive password input",
		Aliases:  []string{"unlock.password"},
		Value:    "",
		Category: "AQUACHAIN",
	}

	VMEnableDebugFlag = &cli.BoolFlag{
		Name:     "vmdebug",
		Usage:    "Record information useful for VM and contract debugging",
		Category: "DEBUGGING",
	}
	// Logging and debug settings
	AquaStatsURLFlag = &cli.StringFlag{
		Name:      "aquastats",
		Usage:     "Reporting URL of a aquastats service (nodename:secret@host:port)",
		Category:  "METRICS",
		Validator: checkStringFlag,
	}
	MetricsEnabledFlag = &cli.BoolFlag{
		Name:     metrics.MetricsEnabledFlag,
		Usage:    "Enable metrics collection and reporting",
		Category: "METRICS",
	}
	FakePoWFlag = &cli.BoolFlag{
		Name:     "fakepow",
		Usage:    "Disables proof-of-work verification for testing purposes",
		Category: "TESTING",
		Hidden:   true,
	}
	NoCompactionFlag = &cli.BoolFlag{
		Name:     "nocompaction",
		Usage:    "Disables db compaction after import",
		Category: "PERFORMANCE",
	}
	// RPC settings
	RPCEnabledFlag = &cli.BoolFlag{
		Name:     "rpc",
		Usage:    "Enable the HTTP-RPC server",
		Category: "RPC SERVER",
	}
	RPCListenAddrFlag = &cli.StringFlag{
		Name:      "rpcaddr",
		Usage:     "HTTP-RPC server listening interface",
		Value:     node.DefaultHTTPHost,
		Category:  "RPC SERVER",
		Validator: checkStringFlag,
	}
	RPCPortFlag = &cli.IntFlag{
		Name:     "rpcport",
		Usage:    "HTTP-RPC server listening port (default is 8543 for aqua, 8743 for testnet3)",
		Value:    0,
		Category: "RPC SERVER",
	}
	RPCCORSDomainFlag = &cli.StringFlag{
		Name:      "rpccorsdomain",
		Usage:     "Comma separated list of domains from which to accept cross origin requests (browser enforced)",
		Value:     "",
		Category:  "RPC SERVER",
		Validator: checkStringFlag,
	}
	RPCVirtualHostsFlag = &cli.StringFlag{
		Name:      "rpcvhosts",
		Usage:     "Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard.",
		Value:     "localhost",
		Category:  "RPC SERVER",
		Validator: checkStringFlag,
	}

	RPCApiFlag = &cli.StringFlag{
		Name:      "rpcapi",
		Usage:     "API's offered over the HTTP-RPC interface",
		Value:     "",
		Category:  "RPC SERVER",
		Validator: checkStringFlag,
	}
	RPCUnlockFlag = &cli.BoolFlag{
		Name:     "UNSAFE_RPC_UNLOCK",
		Usage:    "for allowing keystore and signing via RPC endpoints. do not use. use a separate signer instance.",
		Category: "RPC SERVER",
	}
	IPCDisabledFlag = &cli.BoolFlag{
		Name:     "ipcdisable",
		Usage:    "Disable the IPC-RPC server",
		Category: "RPC SERVER",
	}
	IPCPathFlag = &cli.StringFlag{
		Name:      "ipcpath",
		Usage:     "Filename for IPC socket/pipe within the datadir (explicit paths escape it)",
		Action:    checkDirectoryFlag,
		Category:  "RPC SERVER",
		Validator: checkStringFlag,
	}
	WSEnabledFlag = &cli.BoolFlag{
		Name:     "ws",
		Usage:    "Enable the WS-RPC server",
		Category: "RPC SERVER",
	}
	WSListenAddrFlag = &cli.StringFlag{
		Name:      "wsaddr",
		Usage:     "WS-RPC server listening interface",
		Value:     node.DefaultWSHost,
		Category:  "RPC SERVER",
		Validator: checkStringFlag,
	}
	WSPortFlag = &cli.IntFlag{
		Name:     "wsport",
		Usage:    "WS-RPC server listening port (default is 8544 for aqua, 8744 for testnet3)",
		Value:    0,
		Category: "RPC SERVER",
	}
	WSApiFlag = &cli.StringFlag{
		Name:      "wsapi",
		Usage:     "API's offered over the WS-RPC interface",
		Value:     "",
		Category:  "RPC SERVER",
		Validator: checkStringFlag,
	}
	WSAllowedOriginsFlag = &cli.StringFlag{
		Name:      "wsorigins",
		Usage:     "Origins from which to accept websockets requests (see also rpcvhosts)",
		Value:     "",
		Category:  "RPC SERVER",
		Validator: checkStringFlag,
	}
	RPCAllowIPFlag = &cli.StringFlag{
		Name:      "allowip",
		Usage:     "Comma separated allowed RPC clients (CIDR notation OK) (http/ws)",
		Value:     "127.0.0.1/24",
		Category:  "RPC SERVER",
		Validator: checkStringFlag,
	}
	RPCBehindProxyFlag = &cli.BoolFlag{
		Name:     "behindproxy",
		Usage:    "If RPC is behind a reverse proxy. (RPC_BEHIND_PROXY env) Changes the way IP is determined when comparing to allowed IP addresses",
		Category: "RPC SERVER",
	}
	ExecFlag = &cli.StringFlag{
		Name:      "exec",
		Usage:     "Execute JavaScript statement",
		Category:  "CONSOLE FLAGS",
		Validator: checkStringFlag,
	}
	PreloadJSFlag = &cli.StringFlag{
		Name:      "preload",
		Usage:     "Comma separated list of JavaScript files to preload into the console",
		Category:  "CONSOLE FLAGS",
		Validator: checkStringFlag,
	}

	// Network Settings
	MaxPeersFlag = &cli.IntFlag{
		Name:     "maxpeers",
		Usage:    "Maximum number of network peers (network disabled if set to 0)",
		Value:    25,
		Category: "NETWORKING",
	}
	MaxPendingPeersFlag = &cli.IntFlag{
		Name:     "maxpendpeers",
		Usage:    "Maximum number of pending connection attempts (defaults used if set to 0)",
		Value:    0,
		Category: "NETWORKING",
	}
	ListenPortFlag = &cli.IntFlag{
		Name:     "port",
		Usage:    "Network listening port",
		Value:    21303,
		Category: "NETWORKING",
	}
	ListenAddrFlag = &cli.StringFlag{
		Name:      "addr",
		Usage:     "Network listening addr (all interfaces, port 21303 TCP and UDP)",
		Value:     "",
		Category:  "NETWORKING",
		Validator: checkStringFlag,
	}
	BootnodesFlag = &cli.StringFlag{
		Name:      "bootnodes",
		Aliases:   []string{"p2p.bootnodes"},
		Usage:     "Comma separated enode URLs for P2P discovery bootstrap (set v4+v5 instead for light servers)",
		Value:     "",
		Category:  "NETWORKING",
		Validator: checkStringFlag,
	}
	// BootnodesV4Flag = &cli.StringFlag{
	// 	Name:  "bootnodesv4",
	// 	Usage: "Comma separated enode URLs for P2P v4 discovery bootstrap (light server, full nodes)",
	// 	Value: "",
	// }
	NodeKeyFileFlag = &cli.StringFlag{
		Name:      "nodekey",
		Usage:     "P2P node key file",
		Category:  "NETWORKING",
		Validator: checkStringFlag,
	}
	NodeKeyHexFlag = &cli.StringFlag{
		Name:     "nodekeyhex",
		Usage:    "P2P node key as hex (for testing)",
		Category: "NETWORKING",
	}
	NATFlag = &cli.StringFlag{
		Name:     "nat",
		Usage:    "NAT port mapping mechanism (any|none|upnp|pmp|extip:<IP>)",
		Value:    "any",
		Category: "NETWORKING",
	}
	NoDiscoverFlag = &cli.BoolFlag{
		Name:     "nodiscover",
		Usage:    "Disables the peer discovery mechanism (manual peer addition)",
		Category: "NETWORKING",
	}
	OfflineFlag = &cli.BoolFlag{
		Name:     "offline",
		Usage:    "Disables networking for offline maintenance",
		Category: "NETWORKING",
	}
	NoKeysFlag = &cli.BoolFlag{
		Name:  "nokeys",
		Usage: "Disables keystore entirely (env: NO_KEYS)",
		Value: sense.IsNoKeys() || sense.EnvBool(sense.Getenv("NO_KEYS")) || sense.EnvBool(sense.Getenv("NOKEYS")), // both just in case
		Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
			if v {
				os.Setenv("NO_KEYS", "1")
				if !sense.IsNoKeys() {
					return fmt.Errorf("failed to set NO_KEYS=1")
				}
			}
			return nil
		},
		Category: "AQUACHAIN",
	}
	NoSignFlag = &cli.BoolFlag{
		Name:  "nosign",
		Usage: "Disables all signing via RPC endpoints (env:NO_SIGN) (useful when wallet is unlocked for signing blocks on a public testnet3 server)",
		Value: sense.EnvBool(sense.Getenv("NO_SIGN")) || sense.EnvBool(sense.Getenv("NOSIGN")), // both just in case
		Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
			if v {
				os.Setenv("NO_SIGN", "1")
				if !sense.IsNoSign() {
					return fmt.Errorf("failed to set NO_SIGN=1")
				}

				return nil
			}
			return nil
		},
		Category: "AQUACHAIN",
	}
	NetrestrictFlag = &cli.StringFlag{
		Name:     "netrestrict",
		Usage:    "Restricts network communication to the given IP networks (CIDR masks)",
		Category: "NETWORKING",
	}
	// Gas price oracle settings
	GpoBlocksFlag = &cli.IntFlag{
		Name:     "gpoblocks",
		Usage:    "Number of recent blocks to check for gas prices",
		Value:    int64(aqua.DefaultConfig.GPO.Blocks),
		Category: "TRANSACTION POOL",
	}
	GpoPercentileFlag = &cli.IntFlag{
		Name:     "gpopercentile",
		Usage:    "Suggested gas price is the given percentile of a set of recent transaction gas prices",
		Value:    int64(aqua.DefaultConfig.GPO.Percentile),
		Category: "TRANSACTION POOL",
	}
	HF8MainnetFlag = &cli.IntFlag{
		Name:     "hf8",
		Usage:    "Hard fork #8 activation block (if HF8 is to be activated, -hf8 <blocknumber>)",
		Value:    -1,
		Category: "AQUACHAIN",
	}
)

var (
	// The app that holds all commands and flags.
	// app = NewApp(gitCommit, "the aquachain command line interface")
	// flags that configure the node
	NodeFlags = [...]cli.Flag{
		// DoitNowFlag,
		IdentityFlag,
		UnlockedAccountFlag,
		PasswordFileFlag,
		BootnodesFlag,
		DataDirFlag,
		KeyStoreDirFlag,
		NoKeysFlag,
		UseUSBFlag,
		AquahashCacheDirFlag,
		AquahashCachesInMemoryFlag,
		AquahashCachesOnDiskFlag,
		AquahashDatasetDirFlag,
		AquahashDatasetsInMemoryFlag,
		AquahashDatasetsOnDiskFlag,
		TxPoolNoLocalsFlag,
		TxPoolJournalFlag,
		TxPoolRejournalFlag,
		TxPoolPriceLimitFlag,
		TxPoolPriceBumpFlag,
		TxPoolAccountSlotsFlag,
		TxPoolGlobalSlotsFlag,
		TxPoolAccountQueueFlag,
		TxPoolGlobalQueueFlag,
		TxPoolLifetimeFlag,
		SyncModeFlag,
		GCModeFlag,
		CacheFlag,
		CacheDatabaseFlag,
		CacheGCFlag,
		TrieCacheGenFlag,
		ListenPortFlag,
		ListenAddrFlag,
		MaxPeersFlag,
		MaxPendingPeersFlag,
		AquabaseFlag,
		GasPriceFlag,
		MinerThreadsFlag,
		MiningEnabledFlag,
		TargetGasLimitFlag,
		NATFlag,
		NoDiscoverFlag,
		OfflineFlag,
		NetrestrictFlag,
		NodeKeyFileFlag,
		NodeKeyHexFlag,
		DeveloperFlag,
		DeveloperPeriodFlag,
		VMEnableDebugFlag,

		AquaStatsURLFlag,
		MetricsEnabledFlag,
		FakePoWFlag,
		NoCompactionFlag,
		GpoBlocksFlag,
		GpoPercentileFlag,
		ExtraDataFlag,
		// ConfigFileFlag,
		HF8MainnetFlag,
		// ChainFlag,
	}

	RPCFlags = [...]cli.Flag{
		RPCEnabledFlag,
		RPCUnlockFlag,
		RPCCORSDomainFlag,
		RPCVirtualHostsFlag,
		RPCListenAddrFlag,
		RPCAllowIPFlag,
		RPCBehindProxyFlag,
		RPCPortFlag,
		RPCApiFlag,
		WSEnabledFlag,
		WSListenAddrFlag,
		WSPortFlag,
		WSApiFlag,
		WSAllowedOriginsFlag,
		IPCDisabledFlag,
		IPCPathFlag,
		AlertModeFlag,
	}
)

var NoEnvFlag = &cli.BoolFlag{Name: "noenv", Usage: "Skip loading existing .env file", Category: "AQUACHAIN"}

var (
	SocksClientFlag = &cli.StringFlag{
		Name:     "socks",
		Value:    "",
		Usage:    "SOCKS5 proxy for outgoing RPC connections (eg: -socks socks5h://localhost:1080)",
		Category: "RPC CLIENT",
	}
	ConsoleFlags = []cli.Flag{JavascriptDirectoryFlag, ExecFlag, PreloadJSFlag, SocksClientFlag}
	DaemonFlags  = append(NodeFlags[:], RPCFlags[:]...)
)

var ConfigFileFlag = &cli.StringFlag{
	Name:     "config",
	Usage:    "TOML configuration file. NEW: In case of multiple instances, use -config=none to disable auto-reading available config files",
	Category: "AQUACHAIN",
}

var (
// NodeFlags    = nodeFlags
// RPCFlags     = rpcFlags
// ConsoleFlags = consoleFlags
// DaemonFlags  = daemonFlags
)
