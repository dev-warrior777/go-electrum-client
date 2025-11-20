// This code is available on the terms of the project LICENSE.md file,
// also available online at https://blueoakcouncil.org/license/1.0.0.

package elxfiro

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"decred.org/dcrdex/dex"
	"github.com/bisoncraft/go-electrum-client/electrumx"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

const (
	FiroHeaderSize        = 80
	FiroProgpowExtra      = 40
	FiroProgpowHeaderSize = FiroHeaderSize + FiroProgpowExtra
)

// These configure ElectrumX network for: FIRO
const (
	FiroCoin                  = "firo"
	FiroHeaderSizeRegtest     = 80
	FiroHeaderSizeFiropow     = 120
	FiroStartpointRegtest     = 0
	FiroStartpointTestnet     = 170_000
	FiroStartpointMainnet     = 987_000
	FiroGenesisRegtest        = "a42b98f04cc2916e8adfb5d9db8a2227c4629bc205748ed2f33180b636ee885b"
	FiroGenesisTestnet        = "aa22adcc12becaf436027ffe62a8fb21b234c58c23865291e5dc52cf53f64fca"
	FiroGenesisMainnet        = "4381deb85b1b2c9843c222944b616d997516dcbd6a964e1eaf0def0830695233"
	FiroMaxOnlinePeersRegtest = 0
	FiroMaxOnlinePeersTestnet = 0 // only one testnet server 95.179.164.13:51002 - Firo  Core 0.14.15.0
	FiroMaxOnlinePeersMainnet = 3 // only 4 servers                              - Firo  Core 0.14.15.0
	FiroMaxOnion              = 0
	FiroStrategyFlagsRegtest  = electrumx.NoDeleteKnownPeers // only one server
	FiroStrategyFlagsTestnet  = electrumx.NoDeleteKnownPeers // only one server
	FiroStrategyFlagsMainnet  = electrumx.NoDeleteKnownPeers // 4 servers
)

type headerDeserializer struct{}

func (d headerDeserializer) Deserialize(r io.Reader) (*electrumx.BlockHeader, error) {
	blockHeader := &electrumx.BlockHeader{}
	sz := int64(FiroProgpowHeaderSize)
	fullHeader := make([]byte, sz)
	_, err := io.ReadFull(r, fullHeader)
	if err != nil {
		return nil, err
	}

	// hash full header
	hash := chainhash.DoubleHashH(fullHeader)
	blockHeader.Hash = electrumx.WireHash(hash)

	// deserialize the block header without the extra progpow bytes
	blockHeaderRdr := bytes.NewReader(fullHeader[:FiroHeaderSize])
	wireHdr := &wire.BlockHeader{}
	err = wireHdr.Deserialize(blockHeaderRdr)
	if err != nil {
		return nil, err
	}
	blockHeader.Version = wireHdr.Version
	blockHeader.Prev = electrumx.WireHash(wireHdr.PrevBlock)
	blockHeader.Merkle = electrumx.WireHash(wireHdr.MerkleRoot)
	return blockHeader, nil
}

type regtestHeaderDeserializer struct{}

func (d regtestHeaderDeserializer) Deserialize(r io.Reader) (*electrumx.BlockHeader, error) {
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
	config.Coin = FiroCoin
	config.MaxOnion = FiroMaxOnion

	switch config.NetType {
	case electrumx.Regtest:
		config.Flags = FiroStrategyFlagsRegtest
		config.HeaderDeserializer = regtestHeaderDeserializer{}
		config.BlockHeaderSize = FiroHeaderSizeRegtest
		config.Genesis = FiroGenesisRegtest
		config.StartPoint = FiroStartpointRegtest
		config.MaxOnlinePeers = FiroMaxOnlinePeersRegtest
	case electrumx.Testnet:
		config.Flags = FiroStrategyFlagsTestnet
		config.HeaderDeserializer = headerDeserializer{}
		config.BlockHeaderSize = FiroHeaderSizeFiropow
		config.Genesis = FiroGenesisTestnet
		config.StartPoint = FiroStartpointTestnet
		config.MaxOnlinePeers = FiroMaxOnlinePeersTestnet
	case electrumx.Mainnet:
		config.Flags = FiroStrategyFlagsTestnet
		config.HeaderDeserializer = headerDeserializer{}
		config.BlockHeaderSize = FiroHeaderSizeFiropow
		config.Genesis = FiroGenesisMainnet
		config.StartPoint = FiroStartpointMainnet
		config.MaxOnlinePeers = FiroMaxOnlinePeersMainnet
	default:
		return nil, fmt.Errorf("config error")
	}

	x := ElectrumXInterface{
		config:  config,
		network: nil,
	}
	return &x, nil
}

func (x *ElectrumXInterface) Start(ctx context.Context, logger dex.Logger) error {
	network := electrumx.NewNetwork(x.config, logger)
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
