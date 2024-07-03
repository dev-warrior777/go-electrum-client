package electrumx

import (
	"context"
	"net"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/dev-warrior777/go-electrum-client/wallet"
	"golang.org/x/net/proxy"
)

type Server struct {
	conn      *ServerConn
	connected bool
}

type NodeServerAddr struct {
	Net, Addr string
}

func (nsa NodeServerAddr) Network() string {
	return nsa.Net
}

func (nsa NodeServerAddr) String() string {
	return nsa.Addr
}

func (nsa *NodeServerAddr) IsEqual(other *NodeServerAddr) bool {
	return nsa.Addr == other.Addr && nsa.Net == other.Net
}

// Ensure simpleAddr implements the net.Addr interface.
var _ net.Addr = NodeServerAddr{}

type ElectrumXConfig struct {
	// The blockchain, Bitcoin, Dash, etc
	Chain wallet.CoinType

	// Size of a block header for this chain, normally 80 bytes.
	BlockHeaderSize int

	// NetType parameters..
	Params *chaincfg.Params

	// The user-agent visible to the network
	UserAgent string

	// Location of the data directory
	DataDir string

	// If you wish to connect to a single trusted electrumX peer set this.
	// For now it *must be set* while we move to multi-node electrum interface
	TrustedPeer *NodeServerAddr

	// A Tor proxy can be set here causing the wallet will use Tor. TODO:
	Proxy proxy.Dialer

	// If not testing do not overwrite existing wallet files
	Testing bool
}

type Nettype string

var Regtest Nettype = "regtest"
var Testnet Nettype = "testnet"
var Mainnet Nettype = "mainnet"

var DebugMode bool

type ElectrumX interface {
	Start(ctx context.Context) error

	GetTip() int64
	GetBlockHeader(height int64) (*wire.BlockHeader, error)
	GetBlockHeaders(startHeight int64, blockCount int64) ([]*wire.BlockHeader, error)
	GetTipChangeNotify() (<-chan int64, error)

	SubscribeScripthashNotify(ctx context.Context, scripthash string) (*ScripthashStatusResult, error)
	UnsubscribeScripthashNotify(ctx context.Context, scripthash string)
	GetScripthashNotify() (<-chan *ScripthashStatusResult, error)

	GetHistory(ctx context.Context, scripthash string) (HistoryResult, error)
	GetListUnspent(ctx context.Context, scripthash string) (ListUnspentResult, error)
	GetTransaction(ctx context.Context, txid string) (*GetTransactionResult, error)
	GetRawTransaction(ctx context.Context, txid string) (string, error)
	//
	EstimateFeeRate(ctx context.Context, confTarget int64) (int64, error)
	Broadcast(ctx context.Context, rawTx string) (string, error)
}
