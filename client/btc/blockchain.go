package btc

import (
	"context"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/wire"
)

// Tip returns ElectrumXInterface current tip
func (ec *BtcElectrumClient) Tip() (tip int64) {
	return ec.GetX().GetTip()
}

func (ec *BtcElectrumClient) tipChange(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case tip, ok := <-ec.rcvTipChangeNotify:
			if ok {
				fmt.Printf("tip change - new headers tip is %d\n", tip)
				ec.updateWalletTip(tip)
				// send tip change to an api user if channel exists
				ec.sendTipChangeNotifyMtx.Lock()
				defer ec.sendTipChangeNotifyMtx.Unlock()
				if ec.sendTipChangeNotify != nil {
					ec.sendTipChangeNotify <- tip
				}
			}
		}
	}
}

// RegisterTipChangeNotify sends a new tip change channel back to an api user
func (ec *BtcElectrumClient) RegisterTipChangeNotify() (<-chan int64, error) {
	ec.sendTipChangeNotifyMtx.Lock()
	defer ec.sendTipChangeNotifyMtx.Unlock()
	if ec.sendTipChangeNotify != nil {
		return nil, errors.New("notify already registered - unregister to close the channel")
	}
	ec.sendTipChangeNotify = make(chan int64, 1)
	return ec.sendTipChangeNotify, nil
}

// UnregisterTipChangeNotify closes the current tip change channel
func (ec *BtcElectrumClient) UnregisterTipChangeNotify() {
	ec.sendTipChangeNotifyMtx.Lock()
	defer ec.sendTipChangeNotifyMtx.Unlock()
	if ec.sendTipChangeNotify != nil {
		close(ec.sendTipChangeNotify)
		ec.sendTipChangeNotify = nil
	}
}

// GetBlockHeader returns a block header from ElectrumXInterface current stored headers
func (ec *BtcElectrumClient) GetBlockHeader(height int64) (*wire.BlockHeader, error) {
	return ec.GetX().GetBlockHeader(height)
}

// GetBlockHeaders returns a range of block headers from ElectrumXInterface current stored headers
func (ec *BtcElectrumClient) GetBlockHeaders(startHeight, count int64) ([]*wire.BlockHeader, error) {
	return ec.GetX().GetBlockHeaders(startHeight, count)
}
