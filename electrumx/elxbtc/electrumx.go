package elxbtc

import (
	"context"
	"errors"

	"github.com/btcsuite/btcd/wire"
	"github.com/dev-warrior777/go-electrum-client/electrumx"
)

type ElectrumXInterface struct {
	config  *electrumx.ElectrumXConfig
	network *electrumx.Network
}

func NewElectrumXInterface(config *electrumx.ElectrumXConfig) (*ElectrumXInterface, error) {
	x := ElectrumXInterface{
		config:  config,
		network: nil,
	}
	return &x, nil
}

func (x *ElectrumXInterface) Start(ctx context.Context) error {
	n := electrumx.NewNetwork(x.config)
	err := n.Start(ctx)
	if err != nil {
		return err
	}
	x.network = n
	return nil
}

// func (x *ElectrumXInterface) run(clientCtx context.Context) {

// 	// Monitor connection loop

// 	for {
// 	newServer:
// 		for {
// 			select {
// 			case <-clientCtx.Done():
// 				return
// 			case <-x.server.Conn.Done():
// 				x.serverMtx.Lock()
// 				x.server.Connected = false
// 				x.serverMtx.Unlock()
// 				break newServer
// 			case hdrs := <-x.server.HeadersNotifyChan:
// 				if hdrs != nil && x.networkRunning() {
// 					x.headersNotify <- hdrs
// 				}
// 			case status := <-x.server.ScripthashNotifyChan:
// 				if status != nil && x.networkRunning() {
// 					x.scripthashNotify <- status
// 				}
// 			}
// 		}

// 		fmt.Println("disconnected: will try a new connection in 5 sec")

// 		for {
// 			time.Sleep(5 * time.Second)
// 			fmt.Println("trying to make a new connection")

// 			// connect to electrumX
// 			sc, err := electrumx.ConnectServer(clientCtx, x.serverAddr, x.connectOpts)
// 			if err == nil {
// 				x.serverMtx.Lock()
// 				x.server.Conn = sc
// 				x.server.HeadersNotifyChan = sc.GetHeadersNotify()
// 				x.server.ScripthashNotifyChan = sc.GetScripthashNotify()
// 				x.server.Connected = true
// 				x.serverMtx.Unlock()
// 				break
// 			}
// 		}
// 	}
// }

var ErrNoNetwork error = errors.New("network not running")

func (x *ElectrumXInterface) GetTipChangeNotify() (<-chan int64, error) {
	if x.network == nil {
		return nil, ErrNoNetwork
	}
	return x.network.GetTipChangeNotify(), nil
}

func (x *ElectrumXInterface) GetScripthashNotify() (<-chan *electrumx.ScripthashStatusResult, error) {
	if x.network == nil {
		return nil, ErrNoNetwork
	}
	return x.network.GetScripthashNotify(), nil
}

func (x *ElectrumXInterface) SubscribeScripthashNotify(ctx context.Context, scripthash string) (*electrumx.ScripthashStatusResult, error) {
	if x.network == nil {
		return nil, ErrNoNetwork
	}
	return x.network.SubscribeScripthashNotify(ctx, scripthash)
}

func (x *ElectrumXInterface) UnsubscribeScripthashNotify(ctx context.Context, scripthash string) {
	if x.network == nil {
		return
	}
	x.network.UnsubscribeScripthashNotify(ctx, scripthash)
}

func (x *ElectrumXInterface) BlockHeader(height int64) (*wire.BlockHeader, error) {
	if x.network == nil {
		return nil, ErrNoNetwork
	}
	return x.network.BlockHeader(height)
}

func (x *ElectrumXInterface) BlockHeaders(startHeight int64, blockCount int64) ([]*wire.BlockHeader, error) {
	if x.network == nil {
		return nil, ErrNoNetwork
	}
	return x.network.BlockHeaders(startHeight, blockCount)
}

func (x *ElectrumXInterface) GetHistory(ctx context.Context, scripthash string) (electrumx.HistoryResult, error) {
	if x.network == nil {
		return nil, ErrNoNetwork
	}
	return x.network.GetHistory(ctx, scripthash)
}

func (x *ElectrumXInterface) GetListUnspent(ctx context.Context, scripthash string) (electrumx.ListUnspentResult, error) {
	if x.network == nil {
		return nil, ErrNoNetwork
	}
	return x.network.GetListUnspent(ctx, scripthash)
}

func (x *ElectrumXInterface) GetTransaction(ctx context.Context, txid string) (*electrumx.GetTransactionResult, error) {
	if x.network == nil {
		return nil, ErrNoNetwork
	}
	return x.network.GetTransaction(ctx, txid)
}

func (x *ElectrumXInterface) GetRawTransaction(ctx context.Context, txid string) (string, error) {
	if x.network == nil {
		return "", ErrNoNetwork
	}
	return x.network.GetRawTransaction(ctx, txid)
}

func (x *ElectrumXInterface) Broadcast(ctx context.Context, rawTx string) (string, error) {
	if x.network == nil {
		return "", ErrNoNetwork
	}
	return x.network.Broadcast(ctx, rawTx)
}

func (x *ElectrumXInterface) EstimateFeeRate(ctx context.Context, confTarget int64) (int64, error) {
	if x.network == nil {
		return 0, ErrNoNetwork
	}
	return x.network.EstimateFeeRate(ctx, confTarget)
}
