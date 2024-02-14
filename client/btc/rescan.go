package btc

import (
	"fmt"

	"github.com/dev-warrior777/go-electrum-client/wallet"
)

// RescanWallet asks ElectrumX for info for our wallet keys back to latest
// checkpoint height.
// We need to do this for a recreated wallet.
func (ec *BtcElectrumClient) RescanWallet() error {
	w := ec.GetWallet()
	if w == nil {
		return ErrNoWallet
	}

	// hdrs := ec.clientHeaders

	// We start from a recent height. For testnet/mainnet that is the lastest
	// checkpoint, for regtest that is 0. Since all wallets have a birthday
	// after this height we do not need to search any further back than this.
	// startPointHeight := hdrs.startPoint
	// endHeight := hdrs.tip

	// highest key index we will try for now
	highestKeyIndex := 100

	for purpose := 0; purpose < 2; purpose++ {
		for keyIndex := 0; keyIndex <= highestKeyIndex; keyIndex++ {
			keyPath := &wallet.KeyPath{
				Purpose: wallet.KeyPurpose(purpose),
				Index:   keyIndex,
			}
			address, err := w.GetAddress(keyPath)
			if err != nil {
				fmt.Printf("bad address for: %d:%d\n", keyIndex, purpose)
				continue
			}
			fmt.Printf("Address: %s Index:purpose %d:%d\n", address.String(), keyIndex, purpose)
		}
	}

	return nil
}
