package btc

import (
	"context"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/wire"
)

func (ec *BtcElectrumClient) Tip() (tip int64) {
	return ec.GetX().GetTip()
}

func (ec *BtcElectrumClient) tipChange(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case t, ok := <-ec.tipChangeNotify:
			if ok {
				fmt.Printf("tip change %d\n", t)
				// TODO: wire in for external caller
			}
		}
	}
}

func (ec *BtcElectrumClient) GetBlockHeader(height int64) (*wire.BlockHeader, error) {
	return ec.GetX().GetBlockHeader(height)
}

func (ec *BtcElectrumClient) GetBlockHeaders(startHeight, count int64) ([]*wire.BlockHeader, error) {
	return ec.GetX().GetBlockHeaders(startHeight, count)
}

func (ec *BtcElectrumClient) RegisterTipChangeNotify() (<-chan int64, error) {
	// TODO: for external caller
	return nil, errors.New("not yet implemented")
}

func (ec *BtcElectrumClient) UnregisterTipChangeNotify() {
	// TODO: for external caller
}
