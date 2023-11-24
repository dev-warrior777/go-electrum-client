package wltbtc

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/wallet/txrules"
	"github.com/dev-warrior777/go-electrum-client/client"
	"github.com/dev-warrior777/go-electrum-client/wallet"
	"github.com/tyler-smith/go-bip39"
)

//////////////////////////////////////////////////////////////////////////////
//	ElectrumWallet

// BtcElectrumWallet implements ElectrumWallet

// TODO: adjust interface while developing because .. simpler
var _ = wallet.ElectrumWallet(&BtcElectrumWallet{})

const WalletVersion = "0.1.0"

var ErrEmptyPassword = errors.New("empty password")

type BtcElectrumWallet struct {
	params *chaincfg.Params

	feeProvider *wallet.FeeProvider

	repoPath string

	storageManager *StorageManager
	txstore        *TxStore
	keyManager     *KeyManager

	mutex *sync.RWMutex

	creationDate time.Time

	running bool
}

// NewBtcElectrumWallet mskes new wallet with a new seed. The Mnemonic should
// be saved offline by the user.
func NewBtcElectrumWallet(config *wallet.WalletConfig, pw string) (*BtcElectrumWallet, error) {
	if pw == "" {
		return nil, ErrEmptyPassword
	}

	ent, err := bip39.NewEntropy(128)
	if err != nil {
		return nil, err
	}
	mnemonic, err := bip39.NewMnemonic(ent)
	if err != nil {
		return nil, err
	}
	// TODO: dbg remove
	fmt.Println("Save: ", mnemonic)

	seed := bip39.NewSeed(mnemonic, "")

	return makeBtcElectrumWallet(config, pw, seed)
}

// RecreateElectrumWallet makes new wallet with a mnenomic seed from an existing wallet.
// pw does not need to be the same as the old wallet
func RecreateElectrumWallet(config *wallet.WalletConfig, pw, mnemonic string) (*BtcElectrumWallet, error) {
	if pw == "" {
		return nil, ErrEmptyPassword
	}
	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		return nil, err
	}

	return makeBtcElectrumWallet(config, pw, seed)
}

func LoadBtcElectrumWallet(config *wallet.WalletConfig, pw string) (*BtcElectrumWallet, error) {
	if pw == "" {
		return nil, ErrEmptyPassword
	}

	return loadBtcElectrumWallet(config, pw)
}

func makeBtcElectrumWallet(config *wallet.WalletConfig, pw string, seed []byte) (*BtcElectrumWallet, error) {

	// dbg
	fmt.Println("seed: ", hex.EncodeToString(seed))

	mPrivKey, err := hdkeychain.NewMaster(seed, config.Params)
	if err != nil {
		return nil, err
	}
	mPubKey, err := mPrivKey.Neuter()
	if err != nil {
		return nil, err
	}
	w := &BtcElectrumWallet{
		repoPath:     config.DataDir,
		params:       config.Params,
		creationDate: time.Now(),
		feeProvider: wallet.NewFeeProvider(
			config.MaxFee,
			config.HighFee,
			config.MediumFee,
			config.LowFee,
			"",
			nil,
		),
		mutex: new(sync.RWMutex),
	}

	sm := NewStorageManager(config.DB.Enc(), config.Params)
	sm.store.Version = "0.1"
	sm.store.Xprv = mPrivKey.String()
	sm.store.Xpub = mPubKey.String()
	if config.StoreEncSeed {
		sm.store.Seed = make([]byte, len(seed))
		copy(sm.store.Seed, seed)
	}
	err = sm.Put(pw)
	if err != nil {
		return nil, err
	}
	w.storageManager = sm

	w.keyManager, err = NewKeyManager(config.DB.Keys(), w.params, mPrivKey)
	mPrivKey.Zero()
	mPubKey.Zero()
	if err != nil {
		return nil, err
	}

	w.txstore, err = NewTxStore(w.params, config.DB, w.keyManager)
	if err != nil {
		return nil, err
	}

	err = config.DB.Cfg().PutCreationDate(w.creationDate)
	if err != nil {
		return nil, err
	}

	// Debug: remove
	if config.Params != &chaincfg.MainNetParams {
		fmt.Println("Created: ", w.creationDate)
		fmt.Println(hex.EncodeToString(sm.store.Seed))
		fmt.Println("Created Addresses:")
		fmt.Println(" --- Receiving ---")
		for i, adr := range w.txstore.adrs {
			fmt.Printf("%d %v\n", i, adr)
			if i == client.GAP_LIMIT-1 {
				fmt.Println(" --- Change ---")
			}
		}
	}
	return w, nil
}

func loadBtcElectrumWallet(config *wallet.WalletConfig, pw string) (*BtcElectrumWallet, error) {

	sm := NewStorageManager(config.DB.Enc(), config.Params)

	err := sm.Get(pw)
	if err != nil {
		return nil, err
	}

	mPrivKey, err := hdkeychain.NewKeyFromString(sm.store.Xprv)
	if err != nil {
		return nil, err
	}

	w := &BtcElectrumWallet{
		repoPath:       config.DataDir,
		storageManager: sm,
		params:         config.Params,
		feeProvider: wallet.NewFeeProvider(
			config.MaxFee,
			config.HighFee,
			config.MediumFee,
			config.LowFee,
			// move to client
			"",
			nil,
		),
		mutex: new(sync.RWMutex),
	}

	w.keyManager, err = NewKeyManager(config.DB.Keys(), w.params, mPrivKey)
	mPrivKey.Zero()
	if err != nil {
		return nil, err
	}

	w.txstore, err = NewTxStore(w.params, config.DB, w.keyManager)
	if err != nil {
		return nil, err
	}

	w.creationDate, err = config.DB.Cfg().GetCreationDate()
	if err != nil {
		return nil, err
	}

	// Debug: remove
	if config.Params != &chaincfg.MainNetParams {
		fmt.Println("Stored Creation Date: ", w.creationDate)
		fmt.Println(hex.EncodeToString(sm.store.Seed))
		fmt.Println("Loaded Addresses:")
		fmt.Println(" --- Receiving ---")
		for i, adr := range w.txstore.adrs {
			fmt.Printf("%d %v\n", i, adr)
			if i == client.GAP_LIMIT-1 {
				fmt.Println(" --- Change ---")
			}
		}
	}

	return w, nil
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//
// API
//
//////////////

// /////////////////////
// start interface impl.

func (w *BtcElectrumWallet) Start() {
	w.running = true
}

func (w *BtcElectrumWallet) CreationDate() time.Time {
	return w.creationDate
}

func (w *BtcElectrumWallet) Params() *chaincfg.Params {
	return w.params
}

func (w *BtcElectrumWallet) CurrencyCode() string {
	if w.params.Name == chaincfg.MainNetParams.Name {
		return "btc"
	} else {
		return "tbtc"
	}
}

func (w *BtcElectrumWallet) IsDust(amount int64) bool {
	// This is a per mempool policy thing .. < 1000 sats for now
	return btcutil.Amount(amount) < txrules.DefaultRelayFeePerKb
}

func (w *BtcElectrumWallet) GetUnusedAddress(purpose wallet.KeyPurpose) (btcutil.Address, error) {
	key, err := w.keyManager.GetUnusedKey(purpose)
	if err != nil {
		return nil, nil
	}
	addrPubKeyHash, _ := key.Address(w.params)
	if err != nil {
		return nil, nil
	}
	return btcutil.Address(addrPubKeyHash), nil
}

// Marks the address as used (involved in at least one transaction)
func (w *BtcElectrumWallet) MarkAddressUsed(address btcutil.Address) error {
	return w.txstore.Keys().MarkKeyAsUsed(address.ScriptAddress())
}

func (w *BtcElectrumWallet) CreateNewAddress(purpose wallet.KeyPurpose) btcutil.Address {
	panic("deprecated method")
}

func (w *BtcElectrumWallet) DecodeAddress(addr string) (btcutil.Address, error) {
	return btcutil.DecodeAddress(addr, w.params)
}

func (w *BtcElectrumWallet) ScriptToAddress(script []byte) (btcutil.Address, error) {
	_, addresses, _, err := txscript.ExtractPkScriptAddrs(script, w.params)
	if err != nil {
		return &btcutil.AddressPubKeyHash{}, err
	}
	if len(addresses) == 0 {
		return &btcutil.AddressPubKeyHash{}, errors.New("unknown script")
	}
	return addresses[0], nil
}

func (w *BtcElectrumWallet) AddressToScript(address btcutil.Address) ([]byte, error) {
	return txscript.PayToAddrScript(address)
}

func (w *BtcElectrumWallet) AddWatchedScript(script []byte) error {
	err := w.txstore.WatchedScripts().Put(script)
	if err != nil {
		return err
	}
	return nil
}

func (w *BtcElectrumWallet) HasAddress(address btcutil.Address) bool {
	_, err := w.keyManager.GetKeyForScript(address.ScriptAddress())
	return err == nil
}

func (w *BtcElectrumWallet) ListAddresses() []btcutil.Address {
	keys := w.keyManager.GetKeys()
	addresses := []btcutil.Address{}
	for _, k := range keys {
		address, err := k.Address(w.params)
		if err != nil {
			continue
		}
		addresses = append(addresses, address)
	}
	return addresses
}

func (w *BtcElectrumWallet) Balance() (int64, int64) {

	//TODO: check if this works with the new logic

	checkIfStxoIsConfirmed := func(utxo wallet.Utxo, stxos []wallet.Stxo) bool {
		for _, stxo := range stxos {
			if stxo.Utxo.WatchOnly {
				continue
			}
			if stxo.SpendTxid.IsEqual(&utxo.Op.Hash) {
				return stxo.SpendHeight > 0
			} else if stxo.Utxo.IsEqual(&utxo) {
				return stxo.Utxo.AtHeight > 0
			}
		}
		return false
	}

	confirmed := int64(0)
	unconfirmed := int64(0)
	utxos, _ := w.txstore.Utxos().GetAll()
	stxos, _ := w.txstore.Stxos().GetAll()
	for _, utxo := range utxos {
		if utxo.WatchOnly {
			continue
		}
		if utxo.AtHeight > 0 {
			confirmed += utxo.Value
		} else {
			if checkIfStxoIsConfirmed(utxo, stxos) {
				confirmed += utxo.Value
			} else {
				unconfirmed += utxo.Value
			}
		}
	}
	return confirmed, unconfirmed
}

func (w *BtcElectrumWallet) Transactions() ([]wallet.Txn, error) {
	return w.txstore.Txns().GetAll(false)
}

func (w *BtcElectrumWallet) HasTransaction(txid chainhash.Hash) bool {
	_, err := w.txstore.Txns().Get(txid)
	// error only for 'no rows in rowset'
	return err == nil
}

func (w *BtcElectrumWallet) GetTransaction(txid chainhash.Hash) (wallet.Txn, error) {
	txn, err := w.txstore.Txns().Get(txid)
	if err == nil {
		tx := wire.NewMsgTx(1)
		rbuf := bytes.NewReader(txn.Bytes)
		err := tx.BtcDecode(rbuf, wire.ProtocolVersion, wire.WitnessEncoding)
		if err != nil {
			return txn, err
		}
		outs := []wallet.TransactionOutput{}
		for i, out := range tx.TxOut {
			var address btcutil.Address
			_, addrs, _, err := txscript.ExtractPkScriptAddrs(out.PkScript, w.params)
			if err != nil {
				fmt.Printf("error extracting address from txn pkscript: %v\n", err)
				return txn, err
			}
			if len(addrs) == 0 {
				address = nil
			} else {
				address = addrs[0]
			}
			tout := wallet.TransactionOutput{
				Address: address,
				Value:   out.Value,
				Index:   uint32(i),
			}
			outs = append(outs, tout)
		}
		txn.Outputs = outs
	}
	return txn, err
}

// Return the confirmed txids and heights for an address
func (w *BtcElectrumWallet) GetAddressHistory(address btcutil.Address) ([]wallet.AddressHistory, error) {
	var history []wallet.AddressHistory

	//TODO:

	return history, nil
}

// Send bitcoins to an external wallet
func (w *BtcElectrumWallet) Spend(amount int64, toAddress btcutil.Address, feeLevel wallet.FeeLevel) ([]byte, error) {
	// not yet implemented
	return nil, wallet.ErrWalletFnNotImplemented
}

// Calculates the estimated size of the transaction and returns the total fee for the given feePerByte
func (w *BtcElectrumWallet) EstimateFee(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, feePerByte uint64) int64 {
	// not yet implemented
	return 0
}

// Build a transaction that sweeps all coins from an address. If it is a p2sh multisig, the redeemScript must be included
func (w *BtcElectrumWallet) SweepAddress(utxos []wallet.Utxo, address *btcutil.Address, key *hdkeychain.ExtendedKey, redeemScript *[]byte, feeLevel wallet.FeeLevel) ([]byte, error) {
	// not yet implemented
	return nil, wallet.ErrWalletFnNotImplemented
}

// Create a signature for a multisig transaction
func (w *BtcElectrumWallet) CreateMultisigSignature(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, key *hdkeychain.ExtendedKey, redeemScript []byte, feePerByte uint64) ([]wallet.Signature, error) {
	// not yet implemented
	return nil, wallet.ErrWalletFnNotImplemented
}

// Combine signatures and optionally broadcast
func (w *BtcElectrumWallet) Multisign(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, sigs1 []wallet.Signature, sigs2 []wallet.Signature, redeemScript []byte, feePerByte uint64, broadcast bool) ([]byte, error) {
	// not yet implemented
	return nil, wallet.ErrWalletFnNotImplemented
}

// Generate a multisig script from public keys. If a timeout is included the returned script should be a timelocked escrow which releases using the timeoutKey.
func (w *BtcElectrumWallet) GenerateMultisigScript(keys []hdkeychain.ExtendedKey, threshold int, timeout time.Duration, timeoutKey *hdkeychain.ExtendedKey) (address btcutil.Address, redeemScript []byte, err error) {
	// not yet implemented
	return nil, nil, wallet.ErrWalletFnNotImplemented

}

// Add a transaction to the database
func (w *BtcElectrumWallet) AddTransaction(tx *wire.MsgTx, height int64, timestamp time.Time) error {
	_, err := w.txstore.AddTransaction(tx, height, timestamp)
	return err
}

func (w *BtcElectrumWallet) Close() {
	if w.running {
		// Any other tear down here .. long running threads, etc.
		w.running = false
	}
}

// end interface impl
/////////////////////

func (w *BtcElectrumWallet) DumpHeaders(writer io.Writer) {
	// w.blockchain.db.Print(writer)
	panic("DumpHeaders: Non-SPV wallet")
}
