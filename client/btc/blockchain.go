package btc

import "github.com/btcsuite/btcd/wire"

func (ec *BtcElectrumClient) Tip() (tip int64, synced bool) {
	// TODO: get tip from electrumx
	return 0, false
}

func (ec *BtcElectrumClient) GetBlockHeaders(startHeight, count int64) ([]*wire.BlockHeader, error) {

	return nil, nil
}

func (ec *BtcElectrumClient) GetBlockHeader(height int64) *wire.BlockHeader {

	return nil
}

func (ec *BtcElectrumClient) RegisterTipChangeNotify() (<-chan int64, error) {

	return nil, nil
}

func (ec *BtcElectrumClient) UnregisterTipChangeNotify() {

}
