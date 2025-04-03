package subcommands

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/aqua"
	"gitlab.com/aquachain/aquachain/aqua/accounts"
	"gitlab.com/aquachain/aquachain/aqua/accounts/keystore"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/sense"
	"gitlab.com/aquachain/aquachain/common/toml"
	"gitlab.com/aquachain/aquachain/node"
	"gitlab.com/aquachain/aquachain/opt/aquaclient"
	"gitlab.com/aquachain/aquachain/p2p"
	"gitlab.com/aquachain/aquachain/subcommands/aquaflags"
	"gitlab.com/aquachain/aquachain/subcommands/buildinfo"
	"gitlab.com/aquachain/aquachain/subcommands/mainctxs"
)

// SetBuildInfo also calls 'buildinfo.SetBuildInfo'
//
// TODO: deduplicate by using buildinfo.BuildInfo in this package
func SetBuildInfo(commit, date, tag string, clientIdentifier0 string) {

	buildinfo.SetBuildInfo(buildinfo.BuildInfo{
		GitCommit:        commit,
		BuildDate:        date,
		GitTag:           tag,
		BuildTags:        "",
		ClientIdentifier: clientIdentifier0,
	})
}

func Subcommands() []*cli.Command {
	return []*cli.Command{
		echoCommand,
		initCommand,
		importCommand,
		exportCommand,
		copydbCommand,
		removedbCommand,
		dumpCommand,
		// See monitorcmd.go:
		//monitorCommand,
		// See accountcmd.go:
		accountCommand,

		// See walletcmd_lite.go
		paperCommand,
		// See consolecmd.go:
		consoleCommand,
		daemonCommand, // previously default
		attachCommand,
		javascriptCommand,
		// See misccmd.go:
		makecacheCommand,
		makedagCommand,
		versionCommand,
		bugCommand,
		licenseCommand,
		// See config.go
		dumpConfigCommand,
	}
}

func SubcommandByName(s string) *cli.Command {
	for _, c := range Subcommands() {
		if c.Name == s {
			return c
		}
	}
	return nil
}

var dumpConfigCommand = &cli.Command{
	Action:      MigrateFlags(dumpConfig),
	Name:        "dumpconfig",
	Usage:       "Show configuration values",
	ArgsUsage:   "",
	Flags:       append(nodeFlags, rpcFlags...),
	Category:    "MISCELLANEOUS COMMANDS",
	Description: `The dumpconfig command shows configuration values.`,
}

// dumpConfig is the dumpconfig command.
func dumpConfig(ctx context.Context, cmd *cli.Command) error {
	buildinfo := buildinfo.GetBuildInfo()
	gitCommit := buildinfo.GitCommit
	clientIdentifier := buildinfo.ClientIdentifier

	var opts []Cfgopt
	if cmd.String("config") == "none" {
		opts = append(opts, NoPreviousConfig)
	}
	_, cfg := MakeConfigNode(ctx, cmd, gitCommit, clientIdentifier, mainctxs.MainCancelCause(), opts...)
	comment := ""

	if cfg.Aqua.Genesis != nil {
		cfg.Aqua.Genesis = nil
		comment += "# Note: this config doesn't contain the genesis block.\n\n"
	}

	out, err := toml.Marshal(&cfg)
	if err != nil {
		return err
	}
	io.WriteString(os.Stdout, comment)
	os.Stdout.Write(out)
	return nil
}

var StartNodeCommand = startNode

func MakeFullNode(ctx context.Context, cmd *cli.Command) *node.Node {
	buildinfo := buildinfo.GetBuildInfo()
	gitCommit := buildinfo.GitCommit
	clientIdentifier := buildinfo.ClientIdentifier

	stack, cfg := MakeConfigNode(ctx, cmd, gitCommit, clientIdentifier, mainctxs.MainCancelCause())
	RegisterAquaService(mainctxs.Main(), stack, cfg.Aqua, cfg.Node.NodeName())

	// Add the Aquachain Stats daemon if requested.
	if cfg.Aquastats.URL != "" {
		RegisterAquaStatsService(stack, cfg.Aquastats.URL)
	}
	return stack
}

// startNode boots up the system node and all registered protocols, after which
// it unlocks any requested accounts, and starts the RPC/IPC interfaces and the
// miner.
func startNode(ctx context.Context, cmd *cli.Command, stack *node.Node) chan struct{} {
	unlocks := strings.Split(strings.TrimSpace(cmd.String(aquaflags.UnlockedAccountFlag.Name)), ",")
	if len(unlocks) == 1 && unlocks[0] == "" {
		unlocks = []string{} // TODO?
	}
	if len(unlocks) > 0 && stack.Config().NoKeys {
		Fatalf("Unlocking accounts is not supported with NO_KEYS mode")
	}
	if !sense.IsNoKeys() {
		for _, v := range unlocks {
			log.Info("Unlocking account", "account", v)
		}
		if len(unlocks) > 0 && unlocks[0] != "" {
			log.Warn("Unlocking account", "unlocks", unlocks)
			passwords := MakePasswordList(cmd)
			if len(passwords) == 0 && cmd.IsSet(aquaflags.PasswordFileFlag.Name) && cmd.String(aquaflags.PasswordFileFlag.Name) == "" {
				// empty password "" means no password
				passwords = append(passwords, "")
			}
			// Unlock any account specifically requested
			ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
			for i, account := range unlocks {
				if trimmed := strings.TrimSpace(account); trimmed != "" {
					unlockAccount(cmd, ks, trimmed, i, passwords)
				}
			}
		}
	}
	// NoCountdown -now flag
	cfg := stack.Config()
	cfg.NoCountdown = cfg.NoCountdown || sense.IsNoCountdown()
	if cfg.NoCountdown {
		node.NoCountdown = true
		p2p.NoCountdown = true
	}
	if cmd.Bool(aquaflags.NoKeysFlag.Name) && !stack.Config().NoKeys {
		log.Crit("NO_KEYS mode is not enabled, but --no-keys flag was set")
		return nil
	}

	// Start up the node itself
	nodeStarted := StartNode(ctx, stack)

	// Register wallet event handlers to open and auto-derive wallets
	if !stack.Config().NoKeys && !stack.Config().NoInProc {
		log.Info("Listening for wallet events in keystore")
		events := make(chan accounts.WalletEvent, 16)
		am := stack.AccountManager()
		for i := 0; i < 10; i++ {
			am = stack.AccountManager()
			if am != nil {
				log.Debug("Account manager found")
				break
			}
			log.Warn("Account manager not running, waiting", "i", i)
			select {
			case <-ctx.Done():
				return nil
			case <-mainctxs.Main().Done():
				return nil
			case <-time.After(time.Second):
			}
		}
		if am == nil {
			Fatalf("Account manager not running, NO_KEYS=%v  NoKeys=%v", stack.Config().NoKeys, cmd.Bool(aquaflags.NoKeysFlag.Name))
		}
		am.Subscribe(events)
		go func() {
			select {
			case <-nodeStarted:
			case <-ctx.Done():
			default:
				if !stack.Config().P2P.Offline {
					log.Info("Starting Account Manager once node is running")
				}
			}

			// allow keystore to start up before p2p if p2p wont be starting
			if !stack.Config().P2P.Offline {
				select {
				case <-ctx.Done():
					return
				case <-nodeStarted:
				case <-time.After(time.Second * 10):
					log.Warn("Node not started after 10 seconds, bailing on wallet event listener")
					return
				}
			}
			// Create an chain state reader for self-derivation
			rpcClient, err := stack.Attach(ctx, "accountManager")
			if err != nil {
				log.GracefulShutdown(log.Errorf("account manager failed to attach to node: %v", err))
				return
			}
			stateReader := aquaclient.NewClient(rpcClient)
			defer rpcClient.Close()
			defer am.Close()
			// Open any wallets already attached
			for _, wallet := range stack.AccountManager().Wallets() {
				if err := wallet.Open(""); err != nil {
					log.Warn("Failed to open wallet", "url", wallet.URL(), "err", err)
				}
			}
			// Listen for wallet event till termination
			for {
				select {
				case <-ctx.Done():
					return
				case <-mainctxs.Main().Done():
					return
				case event := <-events:

					log.Info("Wallet event", "kind", event.Kind.String(), "wallet", filepath.Base(event.Wallet.URL().String()))
					switch event.Kind {
					case accounts.WalletArrived:
						if err := event.Wallet.Open(""); err != nil {
							log.Warn("New wallet appeared, failed to open", "url", filepath.Base(event.Wallet.URL().String()), "err", err)
						}
					case accounts.WalletOpened:
						status, _ := event.Wallet.Status()
						log.Info("New wallet appeared", "url", filepath.Base(event.Wallet.URL().String()), "status", status)

						if event.Wallet.URL().Scheme == "ledger" {
							event.Wallet.SelfDerive(accounts.DefaultLedgerBaseDerivationPath, stateReader)
						} else {
							event.Wallet.SelfDerive(accounts.DefaultBaseDerivationPath, stateReader)
						}

					case accounts.WalletDropped:
						log.Info("Old wallet dropped", "url", filepath.Base(event.Wallet.URL().String()))
						event.Wallet.Close()
					}
				}
			}

		}()
	}
	// Start auxiliary services if enabled
	if cmd.Bool(aquaflags.MiningEnabledFlag.Name) || cmd.Bool(aquaflags.DeveloperFlag.Name) {
		var aquachain *aqua.Aquachain
		if err := stack.Service(&aquachain); err != nil {
			Fatalf("Aquachain service not running: %v", err)
		}
		// Use a reduced number of threads if requested
		if threads := cmd.Int(aquaflags.MinerThreadsFlag.Name); threads > 0 {
			type threaded interface {
				SetThreads(threads int)
			}
			if th, ok := aquachain.Engine().(threaded); ok {
				th.SetThreads(int(threads))
			}
		}
		// Set the gas price to the limits from the CLI and start mining
		if cmd.IsSet(aquaflags.GasPriceFlag.Name) {
			if x := aquaflags.GlobalBig(cmd, aquaflags.GasPriceFlag.Name); x != nil {
				aquachain.TxPool().SetGasPrice(x)
			}
		}
		log.Info("gas price", "min", aquachain.TxPool().GasPrice())
		if err := aquachain.StartMining(true); err != nil {
			Fatalf("Failed to start mining: %v", err)
		}
	}
	return nodeStarted
}
