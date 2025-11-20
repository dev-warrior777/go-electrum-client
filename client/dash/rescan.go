// This code is available on the terms of the project LICENSE.md file,
// also available online at https://blueoakcouncil.org/license/1.0.0.

package dash

import (
	"context"
	"encoding/hex"

	"github.com/bisoncraft/go-electrum-client/client"
	"github.com/bisoncraft/go-electrum-client/wallet"
)

// RescanWallet asks ElectrumX for info for our wallet keys back to latest
// checkpoint height.
// We need to do this for a recreated wallet.
func (ec *DashElectrumClient) RescanWallet(ctx context.Context) error {
	w := ec.GetWallet()
	if w == nil {
		return ErrNoWallet
	}
	node := ec.GetX()
	if node == nil {
		return ErrNoElectrumX
	}

	// highest key index we will try for now
	highestKeyIndex := 100
	historyHitIndex := 0

	for keyIndex := 0; keyIndex <= highestKeyIndex; keyIndex++ {
		// flip-flop internal/external to improve locality
		for change := 0; change < 2; change++ {
			keyPath := &wallet.KeyPath{
				Change: wallet.KeyChange(change),
				Index:  keyIndex,
			}
			address, err := w.GetAddress(keyPath)
			if err != nil {
				ec.Log.Errorf("bad address for: %d:%d - %v", keyIndex, change, err)
				continue
			}
			scripthash, err := addressToElectrumScripthash(address)
			if err != nil {
				ec.Log.Errorf("cannot make script hash for address: %s - %v", address.String(), err)
				continue
			}
			ec.Log.Tracef("%s %s  Index:change %d:%d", address.String(), scripthash, keyIndex, change)

			history, err := node.GetHistory(ctx, scripthash)
			if err != nil {
				ec.Log.Errorf("error: %v - for scripthash %s", scripthash, err)
				continue
			}
			if len(history) == 0 {
				ec.Log.Tracef("No history for script hash from node: %s", scripthash)
				continue
			}
			// got history - update the highest hit index
			historyHitIndex = keyIndex
			// for _, h := range history {
			// 	fmt.Println(" Height:", h.Height)
			// 	fmt.Println(" TxHash: ", h.TxHash)
			// 	fmt.Println(" Fee: ", h.Fee)
			// }
			pkScriptBytes, err := w.AddressToScript(address)
			if err != nil {
				ec.Log.Errorf("cannot make pkScript for address: %s - %v", address.String(), err)
				continue
			}
			subscription := &wallet.Subscription{
				PkScript:           hex.EncodeToString(pkScriptBytes),
				ElectrumScripthash: scripthash,
				Address:            address.String(),
			}
			err = w.AddSubscription(subscription)
			if err != nil {
				ec.Log.Errorf("cannot add subscritpion for address: %s - %v", address.String(), err)
				// ec.dumpSubscription("failed to add", subscription)
				continue
			}
			ec.Log.Tracef("Added subscription for address: %s to wallet subscriptions", address.String())
		}

		// if no more history hits for another GAP_LIMIT tries consider the job done.
		if keyIndex > historyHitIndex+client.GAP_LIMIT {
			ec.Log.Debugf("keyIndex: %d greater than highest history found index %d by GAP_LIMIT %d",
				keyIndex, historyHitIndex, client.GAP_LIMIT)
			break
		}
	}

	return nil
}
