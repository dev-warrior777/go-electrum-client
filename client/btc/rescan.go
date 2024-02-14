package btc

import "github.com/dev-warrior777/go-electrum-client/wallet"

// RescanWallet asks ElectrumX for info for our wallet keys back to latest
// checkpoint height.
// We need to do this for a recreated wallet.
func (ec *BtcElectrumClient) RescanWallet() error {
	w := ec.GetWallet()
	if w == nil {
		return ErrNoWallet
	}

	hdrs := ec.clientHeaders

	// We start from a recent height. For testnet/mainnet that is the lastest
	// checkpoint, for regtest that is 0. Since all wallets have a birthday
	// after this height we do not need to search any further back than this.
	startPointHeight := hdrs.startPoint
	scanRange := hdrs.tip - startPointHeight + 1

	var k int64
	for purpose := 0; purpose < 2; purpose++ {
		for k = 0; k < scanRange; k++ {
			keyPath := &wallet.KeyPath{
				Purpose: wallet.KeyPurpose(purpose),
				Index:   int(k),
			}
			w.GetAddress(keyPath)

		}
	}

	return nil
}
