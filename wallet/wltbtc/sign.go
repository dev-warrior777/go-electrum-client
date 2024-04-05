package wltbtc

import (
	"errors"

	"github.com/btcsuite/btcd/wire"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

// Sign an unsigned transaction with the wallet
func (w *BtcElectrumWallet) SignTx(pw string, tx *wire.MsgTx, info *wallet.SigningInfo) (int, []byte, error) {
	if ok := w.storageManager.IsValidPw(pw); !ok {
		return -1, nil, errors.New("invalid password")
	}

	return -1, nil, nil
}
