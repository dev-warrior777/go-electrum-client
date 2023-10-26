package electrumx

import (
	"net"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dev-warrior777/go-electrum-client/wallet"
	"golang.org/x/net/proxy"
)

type NodeConfig struct {
	// The blockchain, Bitcoin, Dash, etc
	Chain wallet.CoinType

	// Network parameters. Set mainnet, testnet using this.
	Params *chaincfg.Params

	// The user-agent that shall be visible to the network
	UserAgent string

	// Location of the data directory
	DataDir string

	// If you wish to connect to a single trusted electrumX peer set this. TODO:
	TrustedPeer net.Addr

	// A Tor proxy can be set here causing the wallet will use Tor. TODO:
	Proxy proxy.Dialer

	// If not testing do not overwrite existing wallet files
	Testing bool
}

type Network string

var Regtest Network = "regtest"
var Testnet Network = "testnet"
var Mainnet Network = "mainnet"

var DebugMode bool

type ElectrumXNode interface {
	// Start()
	// Stop()
}

type SingleNode struct {
	NodeConfig *NodeConfig
	Server     *ServerConn
}

type MultiNode struct {
	ServerMap map[string]*ServerConn
}
