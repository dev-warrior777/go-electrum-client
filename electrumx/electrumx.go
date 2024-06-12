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
	conn                 *ServerConn
	scripthashNotifyChan <-chan *ScripthashStatusResult
	headersNotifyChan    <-chan *HeadersNotifyResult
	connected            bool
}

type ServerAddr struct {
	Net, Addr string
}

func (a ServerAddr) Network() string {
	return a.Net
}
func (a ServerAddr) String() string {
	return a.Addr
}

// Ensure simpleAddr implements the net.Addr interface.
var _ net.Addr = ServerAddr{}

type ElectrumXConfig struct {
	// The blockchain, Bitcoin, Dash, etc
	Chain wallet.CoinType

	// NetType parameters. Set mainnet, testnet using this.
	Params *chaincfg.Params

	// The user-agent that shall be visible to the network
	UserAgent string

	// Location of the data directory
	DataDir string

	// If you wish to connect to a single trusted electrumX peer set this.
	// For now it *must be set* while we move to multi-node electrum interface
	TrustedPeer net.Addr

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

	GetTipChangeNotify() (<-chan int64, error)

	GetScripthashNotify() (<-chan *ScripthashStatusResult, error)
	SubscribeScripthashNotify(ctx context.Context, scripthash string) (*ScripthashStatusResult, error)
	UnsubscribeScripthashNotify(ctx context.Context, scripthash string)

	BlockHeader(height int64) (*wire.BlockHeader, error)
	BlockHeaders(startHeight int64, blockCount int64) ([]*wire.BlockHeader, error)
	// BlockHeader(ctx context.Context, height int64) (string, error)
	// BlockHeaders(ctx context.Context, startHeight int64, blockCount int) (*GetBlockHeadersResult, error)
	GetHistory(ctx context.Context, scripthash string) (HistoryResult, error)
	GetListUnspent(ctx context.Context, scripthash string) (ListUnspentResult, error)
	GetTransaction(ctx context.Context, txid string) (*GetTransactionResult, error)
	GetRawTransaction(ctx context.Context, txid string) (string, error)
	//
	EstimateFeeRate(ctx context.Context, confTarget int64) (int64, error)
	Broadcast(ctx context.Context, rawTx string) (string, error)
}
