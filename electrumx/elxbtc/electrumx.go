package elxbtc

import (
	"context"
	"errors"

	"github.com/btcsuite/btcd/wire"
	"github.com/dev-warrior777/go-electrum-client/electrumx"
)

const (
	BTC_HEADER_SIZE        = 80
	BTC_STARTPOINT_REGTEST = 0
	BTC_STARTPOINT_TESTNET = 2560000
	BTC_STARTPOINT_MAINNET = 823000
)

type ElectrumXInterface struct {
	config  *electrumx.ElectrumXConfig
	network *electrumx.Network
}

func NewElectrumXInterface(config *electrumx.ElectrumXConfig) (*ElectrumXInterface, error) {
	config.BlockHeaderSize = BTC_HEADER_SIZE
	config.StartPoints = make(map[string]int64)
	config.StartPoints[electrumx.REGTEST] = int64(BTC_STARTPOINT_REGTEST)
	config.StartPoints[electrumx.TESTNET] = int64(BTC_STARTPOINT_TESTNET)
	config.StartPoints[electrumx.MAINNET] = int64(BTC_STARTPOINT_MAINNET)
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

var ErrNoNetwork error = errors.New("btc: network not running")

func (x *ElectrumXInterface) GetTip() int64 {
	if x.network == nil {
		return 0
	}
	tip, err := x.network.Tip()
	if err != nil {
		return 0
	}
	return tip
}

func (x *ElectrumXInterface) GetBlockHeader(height int64) (*wire.BlockHeader, error) {
	if x.network == nil {
		return nil, ErrNoNetwork
	}
	return x.network.BlockHeader(height)
}

func (x *ElectrumXInterface) GetBlockHeaders(startHeight int64, blockCount int64) ([]*wire.BlockHeader, error) {
	if x.network == nil {
		return nil, ErrNoNetwork
	}
	return x.network.BlockHeaders(startHeight, blockCount)
}

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
