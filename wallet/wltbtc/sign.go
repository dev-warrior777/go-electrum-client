package wltbtc

import (
	"errors"
)

// Sign an unsigned transaction with the wallet
func (w *BtcElectrumWallet) SignTx(pw string, txBytes []byte) (int, []byte, error) {
	if ok := w.storageManager.IsValidPw(pw); !ok {
		return -1, nil, errors.New("invalid password")
	}
	// utx := wire.NewMsgTx(wire.TxVersion)

	// utx.Deserialize()

	return -1, nil, nil
}
