package elxfiro

import (
	"context"
	"errors"
	"io"

	"github.com/btcsuite/btcd/wire"
	"github.com/dev-warrior777/go-electrum-client/electrumx"
)

// These configure ElectrumX network for: BTC
const (
	FIRO_COIN                     = "firo"
	FIRO_HEADER_SIZE              = 80 // check this for MTP legacy. Now FiroPoW (ProgPow clone) .. should be 80
	FIRO_STARTPOINT_REGTEST       = 0
	FIRO_STARTPOINT_TESTNET       = 2560000 // TODO:
	FIRO_STARTPOINT_MAINNET       = 823000  // TODO:
	FIRO_MAX_ONLINE_PEERS_REGTEST = 0
	FIRO_MAX_ONLINE_PEERS_TESTNET = 0 // only one testnet server currently with outdated firod version last I looked
	FIRO_MAX_ONLINE_PEERS_MAINNET = 3 // only 4 servers last I looked
	FIRO_MAX_ONION                = 0
)

type headerDeserialzer struct{}

func (d headerDeserialzer) Deserialize(r io.Reader) (*electrumx.BlockHeader, error) {
	wireHdr := &wire.BlockHeader{}
	err := wireHdr.Deserialize(r)
	if err != nil {
		return nil, err
	}
	blockHeader := &electrumx.BlockHeader{}
	blockHeader.Version = wireHdr.Version
	chainHash := wireHdr.BlockHash()
	blockHeader.Hash = electrumx.WireHash(chainHash)
	blockHeader.Prev = electrumx.WireHash(wireHdr.PrevBlock)
	blockHeader.Merkle = electrumx.WireHash(wireHdr.MerkleRoot)
	return blockHeader, nil
}

type ElectrumXInterface struct {
	config  *electrumx.ElectrumXConfig
	network *electrumx.Network
}

func NewElectrumXInterface(config *electrumx.ElectrumXConfig) (*ElectrumXInterface, error) {
	config.Coin = FIRO_COIN
	config.BlockHeaderSize = FIRO_HEADER_SIZE
	config.StartPoints = make(map[string]int64)
	config.StartPoints[electrumx.REGTEST] = int64(FIRO_STARTPOINT_REGTEST)
	config.StartPoints[electrumx.TESTNET] = int64(FIRO_STARTPOINT_TESTNET)
	config.StartPoints[electrumx.MAINNET] = int64(FIRO_STARTPOINT_MAINNET)
	config.MaxOnion = FIRO_MAX_ONION
	switch config.NetType {
	case electrumx.Regtest:
		config.MaxOnlinePeers = FIRO_MAX_ONLINE_PEERS_REGTEST
	case electrumx.Testnet:
		config.MaxOnlinePeers = FIRO_MAX_ONLINE_PEERS_TESTNET
	case electrumx.Mainnet:
		config.MaxOnlinePeers = FIRO_MAX_ONLINE_PEERS_MAINNET
	default:
		config.MaxOnlinePeers = 2
	}
	deserializer := headerDeserialzer{}
	config.HeaderDeserializer = &deserializer
	x := ElectrumXInterface{
		config:  config,
		network: nil,
	}
	return &x, nil
}

func (x *ElectrumXInterface) Start(ctx context.Context) error {
	network := electrumx.NewNetwork(x.config)
	err := network.Start(ctx)
	if err != nil {
		return err
	}
	x.network = network
	return nil
}

var ErrNoNetwork error = errors.New("firo: network not running")

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

func (x *ElectrumXInterface) GetSyncStatus() bool {
	if x.network == nil {
		return false
	}
	return x.network.Synced()
}

func (x *ElectrumXInterface) GetBlockHeader(height int64) (*electrumx.ClientBlockHeader, error) {
	if x.network == nil {
		return nil, ErrNoNetwork
	}
	return x.network.BlockHeader(height)
}

func (x *ElectrumXInterface) GetBlockHeaders(startHeight int64, blockCount int64) ([]*electrumx.ClientBlockHeader, error) {
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
