package btc

import (
	"bytes"
	"errors"

	"github.com/btcsuite/btcd/wire"
)

// /////////////////////////////////////////////////////////////////////////////
// Helpers
// ///////

func newWireTx(b []byte, checkIo bool) (*wire.MsgTx, error) {
	tx := wire.NewMsgTx(wire.TxVersion)
	r := bytes.NewBuffer(b)
	err := tx.Deserialize(r)
	if checkIo {
		if len(tx.TxIn) == 0 {
			return nil, errors.New("tx: no inputs")
		}
		if len(tx.TxOut) == 0 {
			return nil, errors.New("tx: no outputs")
		}
	}
	return tx, err
}

func serializeWireTx(tx *wire.MsgTx) ([]byte, error) {
	b := make([]byte, 0, tx.SerializeSize())
	w := bytes.NewBuffer(b)
	err := tx.Serialize(w)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}
