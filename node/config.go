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

package node

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"gitlab.com/aquachain/aquachain/aqua/accounts"
	"gitlab.com/aquachain/aquachain/aqua/accounts/keystore"
	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/crypto"
	"gitlab.com/aquachain/aquachain/p2p"
	"gitlab.com/aquachain/aquachain/p2p/discover"
	"gitlab.com/aquachain/aquachain/subcommands/buildinfo"
)

const (
	datadirPrivateKey      = "nodekey"            // Path within the datadir to the node's private key
	datadirDefaultKeyStore = "keystore"           // Path within the datadir to the keystore
	datadirStaticNodes     = "static-nodes.json"  // Path within the datadir to the static node list
	datadirTrustedNodes    = "trusted-nodes.json" // Path within the datadir to the trusted node list
	datadirNodeDatabase    = "nodes"              // Path within the datadir to store the node infos
)

// Config represents a small collection of configuration values to fine tune the
// P2P network layer of a protocol stack. These values can be further extended by
// all registered services.
//
// Fields with 'toml:"-" are ignored by the TOML parser/writer.
type Config struct {
	// Name sets the instance name of the node. It must not contain the / character and is
	// used in the devp2p node identifier. Instance name of aquachain is "aquachain".
	//
	// Set by the compiler, TODO remove and use global
	Name string `toml:"-"`

	// Version should be set to the version number of the program. It is used
	// in the devp2p node identifier.
	Version string `toml:"-"`

	// UserIdent, if set, is used as an additional component in the devp2p node identifier.
	UserIdent string `toml:",omitempty"`

	// DataDir is the file system folder the node should use for any data storage
	// requirements. The configured data directory will not be directly shared with
	// registered services, instead those can use utility methods to create/access
	// databases or flat files. This enables ephemeral nodes which can fully reside
	// in memory.
	DataDir string

	// Configuration of peer-to-peer networking.
	P2P *p2p.Config

	// KeyStoreDir is the file system folder that contains private keys. The directory can
	// be specified as a relative path, in which case it is resolved relative to the
	// current directory.
	//
	// If KeyStoreDir is empty, the default location is the "keystore" subdirectory of
	// DataDir. If DataDir is unspecified and KeyStoreDir is empty, an ephemeral directory
	// is created by New and destroyed when the node is stopped.
	KeyStoreDir string `toml:",omitempty"`

	// UseLightweightKDF lowers the memory and CPU requirements of the key store
	// scrypt KDF at the expense of security.
	UseLightweightKDF bool `toml:",omitempty"`

	// UseUSB enables hardware wallet monitoring and connectivity.
	UseUSB bool `toml:",omitempty"`

	// IPCPath is the requested location to place the IPC endpoint. If the path is
	// a simple file name, it is placed inside the data directory (or on the root
	// pipe path on Windows), whereas if it's a resolvable path name (absolute or
	// relative), then that specific path is enforced. An empty path disables IPC.
	IPCPath string `toml:",omitempty"`

	// HTTPHost is the host interface on which to start the HTTP RPC server. If this
	// field is empty, no HTTP API endpoint will be started.
	HTTPHost string `toml:",omitempty"`

	// HTTPPort is the TCP port number on which to start the HTTP RPC server. The
	// default zero value is/ valid and will pick a port number randomly (useful
	// for ephemeral nodes).
	HTTPPort int `toml:",omitempty"`

	// HTTPCors is the Cross-Origin Resource Sharing header to send to requesting
	// clients. Please be aware that CORS is a browser enforced security, it's fully
	// useless for custom HTTP clients.
	HTTPCors []string `toml:",omitempty"`

	// HTTPVirtualHosts is the list of virtual hostnames which are allowed on incoming requests.
	// This is by default {'localhost'}. Using this prevents attacks like
	// DNS rebinding, which bypasses SOP by simply masquerading as being within the same
	// origin. These attacks do not utilize CORS, since they are not cross-domain.
	// By explicitly checking the Host-header, the server will not allow requests
	// made against the server with a malicious host domain.
	// Requests using ip address directly are not affected
	HTTPVirtualHosts []string `toml:",omitempty"`
	RPCAllowIP       []string `toml:",omitempty"`

	// HTTPModules is a list of API modules to expose via the HTTP RPC interface.
	// If the module list is empty, all RPC API endpoints designated public will be
	// exposed.
	HTTPModules []string `toml:",omitempty"`

	// WSHost is the host interface on which to start the websocket RPC server. If
	// this field is empty, no websocket API endpoint will be started.
	WSHost string `toml:",omitempty"`

	// WSPort is the TCP port number on which to start the websocket RPC server. The
	// default zero value is/ valid and will pick a port number randomly (useful for
	// ephemeral nodes).
	WSPort int `toml:",omitempty"`

	// WSOrigins is the list of domain to accept websocket requests from. Please be
	// aware that the server can only act upon the HTTP request the client sends and
	// cannot verify the validity of the request header.
	WSOrigins []string `toml:",omitempty"`

	// WSModules is a list of API modules to expose via the websocket RPC interface.
	// If the module list is empty, all RPC API endpoints designated public will be
	// exposed.
	WSModules []string `toml:",omitempty"`

	// WSExposeAll exposes all API modules via the WebSocket RPC interface rather
	// than just the public ones.
	//
	// *WARNING* Only set this if the node is running in a trusted network, exposing
	// private APIs to untrusted users is a major security risk.
	WSExposeAll bool `toml:",omitempty"`

	// Logger is a custom logger to use with the p2p.Server.
	Logger log.LoggerI `toml:",omitempty"`

	// NoKeys disables all signing and keystore functions
	NoKeys bool

	// RPCBehindProxy if true, tried X-FORWARDED-FOR and X-REAL-IP headers to
	// fetch client's remote IP
	RPCBehindProxy bool

	CloseMain   func(error)     `toml:"-"`
	Context     context.Context `toml:"-"`
	NoInProc    bool            `toml:",omitempty"` // disable in-process node (for testing)
	NoCountdown bool            `toml:",omitempty"` // disable countdown on startup
}

// IPCEndpoint resolves an IPC endpoint based on a configured value, taking into
// account the set data folders as well as the designated platform we're currently
// running on.
func (c *Config) IPCEndpoint() string {
	// Short circuit if IPC has not been enabled
	if c.IPCPath == "" {
		return ""
	}
	// On windows we can only use plain top-level pipes
	if runtime.GOOS == "windows" {
		if strings.HasPrefix(c.IPCPath, `\\.\pipe\`) {
			return c.IPCPath
		}
		return `\\.\pipe\` + c.IPCPath
	}
	// Resolve names into the data directory full paths otherwise
	if filepath.Base(c.IPCPath) == c.IPCPath {
		if c.DataDir == "" {
			log.Warn("No datadir set, IPCPath will be ephemeral")
			dir, err := os.MkdirTemp("", "aquachain-")
			if err != nil {
				log.GracefulShutdown(fmt.Errorf("mkdirtemp: %v", err))

			}
			return filepath.Join(dir, c.IPCPath)
		}
		return filepath.Join(c.DataDir, c.IPCPath)
	}
	return c.IPCPath
}

// NodeDB returns the path to the discovery node database.
func (c *Config) NodeDB() string {
	if c.DataDir == "" {
		return "" // ephemeral
	}
	return c.resolvePath(datadirNodeDatabase)
}

// DefaultIPCEndpoint returns the IPC path used by default (for client to assume endpoint)
func DefaultIPCEndpoint(clientIdentifier string) string {
	if clientIdentifier == "" {
		clientIdentifier = strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
		if clientIdentifier == "" {
			panic("empty executable name")
		}
	}
	config := &Config{DataDir: defaultDataDir(), IPCPath: clientIdentifier + ".ipc"}
	return config.IPCEndpoint()
}

// HTTPEndpoint resolves an HTTP endpoint based on the configured host interface
// and port parameters.
func (c Config) HTTPEndpoint() string {
	return getEndpoint(c.HTTPHost, c.HTTPPort)
}

// DefaultHTTPEndpoint returns the HTTP endpoint used by default.
func DefaultHTTPEndpoint() string {
	return (Config{HTTPHost: DefaultHTTPHost, HTTPPort: DefaultHTTPPort}).HTTPEndpoint()
}

// WSEndpoint resolves an websocket endpoint based on the configured host interface
// and port parameters.
func (c *Config) WSEndpoint() string {
	return getEndpoint(c.WSHost, c.WSPort)
}

func getEndpoint(host string, port int) string {
	if host == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", host, port)
}

// DefaultWSEndpoint returns the websocket endpoint used by default.
func DefaultWSEndpoint() string {
	config := &Config{WSHost: DefaultWSHost, WSPort: DefaultWSPort}
	return config.WSEndpoint()
}

func (c *Config) NodeName() string {
	return GetNodeName(c)
}

// GetNodeName returns the devp2p node identifier. (uses cfg.Name, cfg.Version, and if exists cfg.UserIdent)
// eg: Aquachain/v1.7.17-dev-f09095/linux-amd64/go1.23.5
func GetNodeName(cfg *Config) string {
	name := cfg.Name // main name of program
	if name == "" {
		panic("empty name given to GetNodeName")
	}
	if name == "aquachain" {
		name = "Aquachain"
	}

	// userident
	if cfg.UserIdent != "" {
		name += "-" + cfg.UserIdent // e.g. Aquachain-supercoolpool
	}

	// version info
	if cfg.Version == "" {
		// guess from buildinfo
		cfg.Version = buildinfo.GetBuildInfo().GitTag
	}
	if cfg.Version != "" {
		name += "/v" + strings.TrimPrefix(cfg.Version, "v") // eg: Aquachain-supercoolpool/v1.7.17-sometag
	}

	// build info
	name += "/" + runtime.GOOS + "-" + runtime.GOARCH // eg: Aquachain-supercoolpool/v1.7.17-sometag/linux-amd64
	name += "/" + common.ShortGoVersion()             // eg: Aquachain-supercoolpool/v1.7.17-sometag/linux-amd64/go1.23.5
	return name
}

// These resources are resolved differently for "aquachain" instances.
var isOldAquachainResource = map[string]bool{
	"chaindata":          true,
	"nodes":              true,
	"nodekey":            true,
	"static-nodes.json":  true,
	"trusted-nodes.json": true,
}

func (c *Config) name() string {
	if c.Name == "" {
		panic("empty node.Config name")
	}
	if strings.Contains(c.Name, "/") {
		panic("c.Name must not contain '/'")
	}
	return c.Name
}

// resolvePath resolves path in the instance directory.
func (c *Config) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if c.DataDir == "" {
		return ""
	}
	// Backwards-compatibility: ensure that data directory files created
	// by aquachain 1.4 are used if they exist.
	if c.name() == "aquachain" && isOldAquachainResource[path] {
		oldpath := ""
		if c.Name == "aquachain" {
			oldpath = filepath.Join(c.DataDir, path)
		}
		if oldpath != "" && common.FileExist(oldpath) {
			// TODO: print warning
			return oldpath
		}
	}
	return filepath.Join(c.instanceDir(), path)
}

func (c *Config) instanceDir() string {
	if c.DataDir == "" {
		return ""
	}
	if c.Name == "" {
	}

	return filepath.Join(c.DataDir, c.Name)
}

// NodeKey retrieves the currently configured private key of the node, checking
// first any manually set key, falling back to the one found in the configured
// data folder. If no key can be found, a new one is generated.
func (c *Config) NodeKey() *btcec.PrivateKey {
	// Use any specifically configured key.
	if c.P2P != nil && c.P2P.PrivateKey != nil {
		return c.P2P.PrivateKey
	}
	// Generate ephemeral key if no datadir is being used.
	if c.DataDir == "" {
		key, err := crypto.GenerateKey()
		if err != nil {
			log.Crit(fmt.Sprintf("Failed to generate ephemeral node key: %v", err))
		}
		return key
	}

	keyfile := c.resolvePath(datadirPrivateKey)
	if key, err := crypto.LoadECDSA(keyfile); err == nil {
		return key
	}
	// No persistent key found, generate and store a new one.
	key, err := crypto.GenerateKey()
	if err != nil {
		log.Crit(fmt.Sprintf("Failed to generate node key: %v", err))
	}
	instanceDir := filepath.Join(c.DataDir, c.name())
	if err := os.MkdirAll(instanceDir, 0700); err != nil {
		log.Error(fmt.Sprintf("Failed to persist node key: %v", err))
		return key
	}
	keyfile = filepath.Join(instanceDir, datadirPrivateKey)
	if err := crypto.SaveECDSA(keyfile, key); err != nil {
		log.Error(fmt.Sprintf("Failed to persist node key: %v", err))
	}
	return key
}

// StaticNodes returns a list of node enode URLs configured as static nodes.
func (c *Config) StaticNodes() []*discover.Node {
	return c.parsePersistentNodes(c.resolvePath(datadirStaticNodes))
}

// TrustedNodes returns a list of node enode URLs configured as trusted nodes.
func (c *Config) TrustedNodes() []*discover.Node {
	return c.parsePersistentNodes(c.resolvePath(datadirTrustedNodes))
}

// parsePersistentNodes parses a list of discovery node URLs loaded from a .json
// file from within the data directory.
func (c *Config) parsePersistentNodes(path string) []*discover.Node {
	// Short circuit if no node config is present
	if c.DataDir == "" {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	// Load the nodes from the config file.
	var nodelist []string
	if err := common.LoadJSON(path, &nodelist); err != nil {
		log.Error(fmt.Sprintf("Can't load node file %s: %v", path, err))
		return nil
	}
	// Interpret the list as a discovery node array
	var nodes []*discover.Node
	for _, url := range nodelist {
		if url == "" {
			continue
		}
		node, err := discover.ParseNode(url)
		if err != nil {
			log.Error(fmt.Sprintf("Node URL %s: %v\n", url, err))
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes
}

// AccountConfig determines the settings for scrypt and keydirectory
func (c *Config) AccountConfig() (int, int, string, error) {
	scryptN := keystore.StandardScryptN
	scryptP := keystore.StandardScryptP
	if c.UseLightweightKDF {
		scryptN = keystore.LightScryptN
		scryptP = keystore.LightScryptP
	}
	if c.NoKeys {
		log.Info("node/config: keystore is disabled")
		return scryptN, scryptP, "", nil
	}

	var (
		keydir string
		err    error
	)
	switch {
	case filepath.IsAbs(c.KeyStoreDir):
		keydir = c.KeyStoreDir
	case c.DataDir != "":
		if c.KeyStoreDir == "" {
			keydir = filepath.Join(c.DataDir, datadirDefaultKeyStore)
			log.Info("node/config: keystore directory not specified, defaulting to datadir/keystore", "pleasebackup", keydir)
		} else {
			keydir, err = filepath.Abs(c.KeyStoreDir)
		}
	case c.KeyStoreDir != "":
		keydir, err = filepath.Abs(c.KeyStoreDir)
	}
	log.Warn("node/config: keystore is enabled", "keydir", keydir)
	return scryptN, scryptP, keydir, err
}

func makeAccountManager(conf *Config) (*accounts.Manager, string, error) {
	scryptN, scryptP, keydir, err := conf.AccountConfig()
	if keydir == "" {
		log.Info("node/config: no keystore directory")
		return nil, "", nil
	}
	var ephemeral string
	if keydir == "" {
		// There is no datadir.
		keydir, err = ioutil.TempDir("", "aquachain-keystore")
		ephemeral = keydir
	}

	if err != nil {
		return nil, "", err
	}
	if err := os.MkdirAll(keydir, 0700); err != nil {
		return nil, "", err
	}
	stat, err := os.Stat(keydir)
	if err != nil {
		return nil, "", err
	}

	if stat.Mode()&0077 != 0 {
		return nil, "", fmt.Errorf("keystore directory has insecure permissions: %s", keydir)
	}

	// Assemble the account manager and supported backends
	backends := []accounts.Backend{
		keystore.NewKeyStore(keydir, scryptN, scryptP),
	}
	return accounts.NewManager(backends...), ephemeral, nil
}
