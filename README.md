# Aquachain

Latest Source: https://gitlab.com/aquachain/aquachain

Mirrored: https://github.com/aquachain/aquachain

View or Improve the Documentation online: https://aquachain.github.io/docs/

[![Website](https://img.shields.io/badge/Website-aqua.github.io-blue.svg)](https://aquachain.github.io)
[![Explorer](https://img.shields.io/badge/Block_Explorer-aquachain.github.io/explorer-blue.svg)](https://aquachain.github.io/explorer/)

[![Build and Test](https://github.com/aquachain/aquachain/actions/workflows/autobuildtest.yml/badge.svg?branch=master)](https://github.com/aquachain/aquachain/actions/workflows/autobuildtest.yml)
[![Release](https://github.com/aquachain/aquachain/actions/workflows/release.yml/badge.svg)](https://github.com/aquachain/aquachain/releases/)

[![Chat on Matrix](https://matrix.to/img/matrix-badge.svg)](https://matrix.to/#/!ZIGUfKJVCVhrCjBRfz:matrix.org/lobby?via=matrix.org)
[![Chat on Discord](https://img.shields.io/badge/chat-on%20discord-blue.svg)](https://discordapp.com/invite/J7jBhZf)
[![Chat on Telegram](https://img.shields.io/badge/chat-on%20telegram-blue.svg)](https://t.me/AquaCrypto)
[![Telegram News](https://img.shields.io/badge/telegram-news-blue.svg)](https://t.me/Aquachain)
[![Twitter](https://img.shields.io/twitter/follow/aquacrypto?style=social)](https://twitter.com/aquacrypto)


[![SafeTrade](https://img.shields.io/badge/SafeTrade-AQUA/BTC-green.svg)](https://safetrade.com/exchange/AQUA-BTC?type=pro?aqua)
[![DexScreener](https://img.shields.io/badge/DexScreener-AQUA_BEP20-green.svg)](https://dexscreener.com/bsc/0x38fab266089aaf3bc2f11b791213840ea3d587c7)

** Found a bug **in this software**? Documentation lacking?
See https://gitlab.com/aquachain/aquachain/wikis/bugs **

See bottom of this document for more useful links. 
Your contributions are welcome.

## General Purpose Distributed Computing

Aquachain: peer-to-peer programmable money, distributed code contract platform.

    Target Block Time: 240 second blocks (4 minute)
    Block Reward: 1 AQUA
    Max Supply: 42 million
    Algorithm: argon2id (CPU or GPU mined)
    ChainID/NetworkID: 61717561

### Known Explorers:

- https://aquachain.github.io/explorer/
- https://chain.n-e-t.name/explorer/

### Known Pools:

- https://aquachain.github.io/pools.json

### Known Wallets:

- https://frame.sh (Desktop)
- https://metamask.io/ (Browser Extension)
- https://download.mycrypto.com (Desktop, creates mnemonic phrases)
- https://walleth.org (Android, burner accounts etc)

## GET AQUACHAIN

The `aquachain` command (full node, RPC server, and wallet) is a portable 
program that doesn't really need an 'installer', you can run it from anywhere. 

When you first start `aquachain` you will connect to the peer-to-peer network 
and start downloading the chain from whatever peers it can find. 

To change the way aquachain runs, for example testnet, or with json-rpc 
endpoints, use command line flags or TOML config file. (see Usage section) 

List all command line flags using the `-h` flag, or `aquachain help [subcommand]`

You should keep backups of your keystore files, if any, and regularly check 
unlocking them. It is generally better to keep private keys away from your node.

Wallets connect to RPC nodes and offer an easy-to-use interface (while keeping your keys off the server).
Hosting your own RPC server is easy and improves privacy and has zero downtime issues.

By default, using the -rpc flag listens only on 127.0.0.1:8543 and 
offers everything needed to connect wallets such as Frame, Metamask, Hardhat.

## COMPILING

Requires only [Go](https://golang.org/dl). Use the latest.

** Patches can be submitted at Github or Gitlab or Mailing List **

```
git clone https://gitlab.com/aquachain/aquachain
cd aquachain
make
```
Programs are built to the ./bin/ directory.

### Linux Servers

To build a single deb for your server, `make clean cross deb release=1` (do not forget the release=1 argument)

Installing will ask some basic questions and install a few new configuration files:

- `/etc/default/aquachain` (edit this to change environmental variables)
- `/etc/aquachain/aquachain.conf` (this is generated by debconf)
- `/etc/systemd/system/aquachain.service`(systemd service file)
- `/usr/local/bin/start_aquachain.bash` (helper script to we use to read the config files and start the daemon with `systemctl`)

Aquachain RPC will start automatically on boot, and can be controlled with `systemctl` and reconfigured with `dpkg-reconfigure aquachain`. This deb installation feature is new and experimental as of v1.7.17.

### Windows

On windows, double-click make.bat to compile aquachain.exe onto your Desktop.


## Releases

While compiled releases may exist, it is better to compile it yourself from latest source using above methods.

They can be cross-compiled with `make clean release release=1`

- [Releases](https://github.com/aquachain/aquachain/releases/latest)

## SYNCHRONIZING

"Imported new chain segment" means you received new blocks from the network.

When a single block is imported, the address of the successful miner is printed.

When you start seeing a single block every 4 minutes or so,
you know that you are fully synchronized with the network.

## USAGE

Enter AQUA javascript console: `aquachain.exe`

Start Daemon (geth's default): `aquachain.exe daemon`

Run localhost rpc (port 8543): `aquachain.exe -rpc` 

See more commands: [Wiki](https://gitlab.com/aquachain/aquachain/wikis/Basics)

Type `help` at the `AQUA>` prompt for common AQUA console commands.

Run `aquachain.exe help` for command line flags and options.

## Config File (TOML)

You can use a TOML file to configure your node. To generate a TOML file based on the current environment, run `aquachain $FLAGS dumpconfig`.

The config file is useful for adding TrustedPeers, StaticPeers, and BootstrapNodes in the config file).

Part of the config file for using manual peers:

```toml
[Node]
[Node.P2P]
BootstrapNodes = ["enode://..."]
StaticPeers = ["enode://..."]
TrustedPeers = ["enode://..."]
NoDiscovery = false
```

To share make a list of your peers, for sharing with others (for example, in case of discovery-network failure), first attach a console (sudo -u aqua aquachain attach) and in it run:

```javascript
x = admin.peers;
for (i = 0; i < x; i++){
  console.log("enode://"+x[i].id + "@"+ x[i].network.remoteAddress)
}
```

## Shell Completion

To enable shell completion, run `aquachain completion bash` and redirect the output to a file in your shell's completion directory.

systemwide:
`aquachain completion bash > /etc/bash_completion.d/aquachain`

user-only:
```
mkdir -p ~/.bash_completion.d
aquachain completion bash > ~/.bash_completion.d/aquachain
```

## Environmental Variables

See also below in RPC section for more environment variables. If using deb package, you can add these to `/etc/default/aquachain` for easy configuration.

(Run `make doc-print-env` to generate a list such as this one with all variables used.)

```
HELP2=1 (show alternate help text)
NO_COUNTDOWN=1 (disable countdown on startup)
NO_KEYS=1 (disable keystore system)
NO_SIGN=1 (disable signing when keys are enabled and even unlocked)
SCHEDULE_TIMEOUT=60s (quit after 60 seconds)
```
#### ENV: Alerts Platform

Get notified of alerts instantly. Set these environment variables to enable alerts. You can run multiple nodes with the same alert config. This is new as of v1.7.17. It sends alerts when the node is started, when it is stopped, and when reorganizations occur. You must also use the -alerts flag, even if all these are set.

```
ALERT_CHANNEL=-121234567
ALERT_PLATFORM="telegram"
ALERTS_INFO=0 to disable INFO alerts
ALERTS_WARN=0 to disable WARN alerts for some reason
ALERTS_TOKEN=your-bot-token
```

#### ENV: Logging

```
COLOR=1 (colorize logs even if not a tty)
JSONLOG=1 (log in JSON format, recommended for servers)
LOGLEVEL=3 for INFO
LOGLEVEL=4 for WARN
TESTLOGLVL is used when running 'go test' for some packages.
```

## RESOURCES

On some platforms, such as OpenBSD, the login class capabilities must be increased before synchronizing the chain. 
Forgetting to do this will likely crash ("Out of Memory") while the machine has plenty of unused RAM. (see `man login.conf`)

## RPC SERVER

See "RPC" section in ./Documentation folder and online at:
https://aquachain.github.io/docs/

Start HTTP JSON/RPC server for local (127.0.0.1) connections only:
aquachain -rpc

Start HTTP JSON/RPC server for remote connections, listening on 192.168.1.5:8543,
able to be accessed only by 192.168.1.6:

    aquachain -rpc -rpchost 192.168.1.5 -allowip 192.168.1.6/32

With no other RPC flags, the `-rpc` flag alone is safe for local usage (from the same machine).

### Security Note about RPC

Please be aware that hosting a public RPC server (0.0.0.0) will allow strangers access to your system.

Do not use the `-rpcaddr` flag unless you absolutely know what you are doing.

For hosting public RPC servers, please consider using -nokeys (_new!_) and implementing
rate limiting on http (and, if using, websockets) , either via reverse proxy such as
caddyserver or nginx, or firewall.

### RPC Access Restrictions

By default, running aquachain with no flags (either the `daemon` subcommand or default `console` subcommand) only allows outside communication from the same user on the same machine, via the ipc socket. Further, all signing methods are disabled. 

To allow TOTALLY UNSAFE methods such as `_sendTransaction` and `_sign`, you must use environment variables to allow access to these methods.

Here are the environment variables that can be set to allow access to these methods:



```
NO_KEYS must NOT be set to allow interacting with keys
NO_SIGN must NOT be set to allow signing
UNSAFE_ALLOW_SIGN_INPROC=1 for console
UNSAFE_ALLOW_SIGN_IPC=1 for attach
UNSAFE_RPC_SIGNING=1 for all RPC methods
UNSAFE_RPC_SIGNING_HTTP=1 for HTTP RPC methods
UNSAFE_RPC_SIGNING_WS=1 for WS RPC methods
```

### Further Restricting RPC access


Can't connect to your remote RPC server? `-allowip` flag is closed **by default**

Recent builds of aquachain include support for the `-allowip` flag.

Default allowip is 127.0.0.1/24, which doesn't allow any LAN or WAN addresses access to your RPC methods. But it does allow localhost access.

To add IPs, use `aquachain -rpc -rpchost 192.168.1.4 -allowip 192.168.1.5/32,192.168.2.30/32`

The CIDR networks are comma separated, no spaces. (the `/32` after an IP means 'one IP')

## RPC Clients

The JSON/RPC server is able to be used with "Web3" libraries for languages such
as **Python** or **Javascript**. 

These include [Ethers](https://docs.ethers.io/v5/) 
or [Hardhat](https://hardhat.org/)
or [web3js](https://web3js.readthedocs.io/)
or [web3py](https://web3py.readthedocs.io/)

For compatibility with existing tools, all calls to `eth_` methods are translated to `aqua_`, behind-the-scenes.

This repository is also a Go library! See each package's documentation (godoc) for more information on usage.

## Major differences from upstream

Aquachain is similar to Ethereum, but differs in a few important ways.

### Max Supply

Max supply of 42 million, estimated to occur Thu Aug  5 04:23:29 AM UTC 2337
(@11600079809)

### No Pre-mine

The number of coins in existence is important.

On the aquachain network, no coins were created without being mined.

One coin for each block, with occasional uncle reward.

Each year the chain grows by approximately 120000 blocks.

Compare in contrast to the Ethereum network, which distributed around 80 million coins to their foundation and pre-sale on day one.
In such a system with pre-mined coin, there will never be a time where miners have outmined the pre-mined team allocation.

By dedicating a rig or two to mining this chain you are promoting a healthy strong network!

Anyone, anywhere, can start mining today to receive a stream of coins to their wallet.

### Block header

Each block has a 'version' which is only known by consensus rules.

When using the aquachain go packages, before calling block.Hash(), the version MUST be 'set' using block.SetVersion(n). **Failure to do so will panic.**

This should always be done exactly ONCE immediately after deserializing a block.

### Hashing Algorithm Switch

Mining software should use the HF map consensus rules to determine the header version based on block height, which determines which hash function to use.

Header version 1 uses ethash.

Header version 2 uses argon2id(1,1,1)

### Full node components

Many non-critical geth components were removed to create a node that is better on resources.

We have a "100% go" build with no C dependencies.

When you compile aquachain, you should have built a static binary.

### Block target time

The 240 seconds block time has successfully deterred typical "DEX" and "ERC20 token" projects from existing on the chain.

This frees up the blocks for you and your user's transactions to be mined at an inexpensive gas price.

Typical DEX/ERC20 projects depend on a relatively short block time to (seemingly) provide realtime trading.

The dangers of a slow block time include but are not limited to frontrunning.

Please do not create a DEX/ERC20 project here.

### EVM Version

Solidity developers take note: the Aquachain EVM is not the same as other chains.

Certain opcodes simply do not exist here such as CREATE2 and CHAINID and others.

When compiling solidity code, use 'Byzantium' EVM target and enable optimization.

If you think there should be a change to our EVM, submit a pull request or join chat.

### ChainID

Aquachain chainId is 61717561

### Hard Fork Constitution

By enforcing these HF limitations, we prevent manipulation of the aquachain economy. Rules should be easy to add, and should be difficult to remove one. (subject to change)

  1. A HF will never be made to increase supply

  2. A HF will never be made to censor or block a transaction

  3. A HF will never be made to refund coin for any reason.

  4. A HF will never be made to benefit one or only a small subset of miners.

  5. A HF will never be made to benefit one or only a small subset of users.

  6. A HF will never be made for no good reason.

  7. A HF will never be made when a simple upgrade could replace it.

  8. A HF should be made as soon as possible if an "ASIC threat is realized"

  9. A HF should be made as soon as possible if an exploit is found that can be abused for profit or leads to denial of service or loss of funds.

  10. A HF should be made as soon as possible if a "serious bug" is found in the core aquachain source code that can lead to loss of money for the average aquachain user.

  11. If no HF are required, a scheduled hard fork can be made to tweak the algorithm every 4 to 6 months, to prevent ASIC production.



## Resources

Website - https://aquachain.github.io

Explorer - https://aquachain.github.io/explorer/

About - https://telegra.ph/Aquachain-AQUA---Decentralized-Processing-07-20

Mining - https://telegra.ph/Mining-AQUA-05-27

Wiki - https://gitlab.com/aquachain/aquachain/wikis

ANN - https://bitcointalk.org/index.php?topic=3138231.0

Gitlab - http://gitlab.com/aquachain/aquachain

Github - http://github.com/aquachain

Telegram News: https://t.me/Aquachain

Godoc - https://godoc.org/gitlab.com/aquachain/aquachain#pkg-subdirectories

Report bugs - https://github.com/aquachain/aquachain/issues

Matrix: https://matrix.to/#/!ZIGUfKJVCVhrCjBRfz:matrix.org/lobby?via=matrix.org

Telegram Chat: https://t.me/AquaCrypto

Discord: https://discordapp.com/invite/J7jBhZf

Bugs: https://gitlab.com/aquachain/aquachain/wikis/bugs

## Contributing

Aquachain is free open source software and your contributions are welcome.

### Some tips and tricks for hacking on Aquachain core:

- Always `gofmt -w -l -s` before commiting. If you forget, adding a simple
  'gofmt -w -l -s' commit message works.
- Before making a merge request, try `make test` to run all tests. If any
  tests pass, the PR can not be merged into the master branch.
- Rebase: Don't `git pull` to update your branch. instead, from your branch, type `git rebase -i master` and resolve any conflicts (do this often and there wont be any!)
- Prefix commit message with package name, such as "core: fix blockchain"
