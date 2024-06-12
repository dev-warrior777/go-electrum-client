package elxbtc

import (
	"context"
	"errors"

	"github.com/dev-warrior777/go-electrum-client/electrumx"
)

type ElectrumXInterface struct {
	config           *electrumx.ElectrumXConfig
	scripthashNotify chan *electrumx.ScripthashStatusResult
	headersNotify    chan *electrumx.HeadersNotifyResult
	network          *electrumx.Network
}

func NewElectrumXInterface(config *electrumx.ElectrumXConfig) (*ElectrumXInterface, error) {
	x := ElectrumXInterface{
		config:           config,
		scripthashNotify: make(chan *electrumx.ScripthashStatusResult, 16), // 128 bytes/slot
		headersNotify:    make(chan *electrumx.HeadersNotifyResult, 16),    // 168 bytes/slot
		network:          nil,
	}
	return &x, nil
}

func (x *ElectrumXInterface) Start(clientCtx context.Context) error {
	n := electrumx.NewNetwork(x.config)
	err := n.Start(clientCtx)
	if err != nil {
		return err
	}
	x.network = n
	return nil
}

func (x *ElectrumXInterface) Stop() {
	// nothing for now
}

// func (x *ElectrumXInterface) start(clientCtx context.Context) error {

// 	return nil

// network := x.config.Params.Name
// genesis := x.config.Paramx.GenesisHash.String()
// fmt.Println("starting single node on", network, "genesis", genesis)

// // connect to electrumX
// sc, err := electrumx.ConnectServer(clientCtx, x.serverAddr, x.connectOpts)
// if err != nil {
// 	return err
// }

// x.server.Conn = sc
// x.server.HeadersNotifyChan = sc.GetHeadersNotify()
// x.server.ScripthashNotifyChan = sc.GetScripthashNotify()
// x.server.Connected = true

// fmt.Printf("** Connected to %s using %s **\n", network, sc.Proto())

// feats, err := sc.Features(clientCtx)
// if err != nil {
// 	return err
// }

// if feats.Genesis != genesis {
// 	return errors.New("wrong genesis hash for Bitcoin")
// }

// // now server is up check if we have required functions like GetTransaction
// // which is not supported on at least one server .. maybe more.
// switch network {
// case "testnet", "testnet3":
// 	txid := "581d837b8bcca854406dc5259d1fb1e0d314fcd450fb2d4654e78c48120e0135"
// 	_, err := sc.GetTransaction(clientCtx, txid)
// 	if err != nil {
// 		return err
// 	}
// case "mainnet":
// 	txid := "f53a8b83f85dd1ce2a6ef4593e67169b90aaeb402b3cf806b37afc634ef71fbc"
// 	_, err := sc.GetTransaction(clientCtx, txid)
// 	if err != nil {
// 		return err
// 	}
// 	// ignore regtest
// }

// go x.run(clientCtx)

// return nil
// }

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

// TODO: remove when removing single-node
// func (x *SingleNode) RegisterNetworkRestart() <-chan *electrumx.NetworkRestart {
// 	return make(chan *electrumx.NetworkRestart, 1)
// }

// TODO: remove when removing single-node
func (x *ElectrumXInterface) GetHeadersNotify() (<-chan *electrumx.HeadersNotifyResult, error) {
	if x.network != nil {
		return nil, ErrNoNetwork
	}
	return x.headersNotify, nil
}

func (x *ElectrumXInterface) SubscribeHeaders(ctx context.Context) (*electrumx.HeadersNotifyResult, error) {
	if x.network != nil {
		return nil, ErrNoNetwork
	}
	return x.network.SubscribeHeaders(ctx)
}

func (x *ElectrumXInterface) GetScripthashNotify() (<-chan *electrumx.ScripthashStatusResult, error) {
	if x.network != nil {
		return nil, ErrNoNetwork
	}
	return x.scripthashNotify, nil
}

func (x *ElectrumXInterface) SubscribeScripthashNotify(ctx context.Context, scripthash string) (*electrumx.ScripthashStatusResult, error) {
	if x.network != nil {
		return nil, ErrNoNetwork
	}
	return x.network.SubscribeScripthashNotify(ctx, scripthash)
}

func (x *ElectrumXInterface) UnsubscribeScripthashNotify(ctx context.Context, scripthash string) {
	if x.network != nil {
		return
	}
	x.network.UnsubscribeScripthashNotify(ctx, scripthash)
}

func (x *ElectrumXInterface) BlockHeader(ctx context.Context, height int64) (string, error) {
	if x.network != nil {
		return "", ErrNoNetwork
	}
	return x.network.BlockHeader(ctx, height)
}

func (x *ElectrumXInterface) BlockHeaders(ctx context.Context, startHeight int64, blockCount int) (*electrumx.GetBlockHeadersResult, error) {
	if x.network != nil {
		return nil, ErrNoNetwork
	}
	return x.network.BlockHeaders(ctx, startHeight, blockCount)
}

func (x *ElectrumXInterface) GetHistory(ctx context.Context, scripthash string) (electrumx.HistoryResult, error) {
	if x.network != nil {
		return nil, ErrNoNetwork
	}
	return x.network.GetHistory(ctx, scripthash)
}

func (x *ElectrumXInterface) GetListUnspent(ctx context.Context, scripthash string) (electrumx.ListUnspentResult, error) {
	if x.network != nil {
		return nil, ErrNoNetwork
	}
	return x.network.GetListUnspent(ctx, scripthash)
}

func (x *ElectrumXInterface) GetTransaction(ctx context.Context, txid string) (*electrumx.GetTransactionResult, error) {
	if x.network != nil {
		return nil, ErrNoNetwork
	}
	return x.network.GetTransaction(ctx, txid)
}

func (x *ElectrumXInterface) GetRawTransaction(ctx context.Context, txid string) (string, error) {
	if x.network != nil {
		return "", ErrNoNetwork
	}
	return x.network.GetRawTransaction(ctx, txid)
}

func (x *ElectrumXInterface) Broadcast(ctx context.Context, rawTx string) (string, error) {
	if x.network != nil {
		return "", ErrNoNetwork
	}
	return x.network.Broadcast(ctx, rawTx)
}

func (x *ElectrumXInterface) EstimateFeeRate(ctx context.Context, confTarget int64) (int64, error) {
	if x.network != nil {
		return 0, ErrNoNetwork
	}
	return x.network.EstimateFeeRate(ctx, confTarget)
}
