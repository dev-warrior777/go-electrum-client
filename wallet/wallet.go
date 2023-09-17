// Package wallet provides a simple interface to btcwallet and electrumx.
package wallet

type ElectrumWallet interface{}

// Addresses returns all addresses generated in the current wallet.

// GenAddresses generates a number of addresses for the wallet.

// SendBitcoin sends some amount of bitcoin specifying minimum confirmations.

// Create creates a wallet with the specified path, private key password and seed.
// Seed can be created using: hdkeychain.GenerateSeed(hdkeychain.RecommendedSeedLen)

// Load loads a wallet with the specified path, private key password and seed.

// func openWallet(db walletdb.DB, privPass string, seed []byte) (*Wallet, error)
// func (w *Wallet) watchAddress(addr string) error
// func (w *Wallet) handleTransactions(c <-chan string)
// func (w *Wallet) insertTx(tx string) error
