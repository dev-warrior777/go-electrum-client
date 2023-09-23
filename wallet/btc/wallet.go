package btc

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	hd "github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/wallet/txrules"
	"github.com/dev-warrior777/go-electrum-client/wallet"
	"github.com/dev-warrior777/go-electrum-client/wallet/db"
	"github.com/tyler-smith/go-bip39"
)

// BtcElectrumClient
type BtcElectrumClient struct {
	config       *wallet.Config
	wallet       wallet.ElectrumWallet
	chainManager *wallet.ChainManager
}

func NewBtcElectrumClient(cfg *wallet.Config) *BtcElectrumClient {
	manager := wallet.NewChainManager(cfg)

	return &BtcElectrumClient{
		config:       cfg,
		chainManager: manager,
	}
}

func (ec *BtcElectrumClient) CreateWallet(privPass string) error {
	cfg := ec.config

	// Select wallet datastore
	sqliteDatastore, err := db.Create(cfg.DataDir)
	if err != nil {
		return err
	}
	cfg.DB = sqliteDatastore

	ec.wallet, err = NewBtcElectrumWallet(cfg, privPass)
	return nil
}

//	ElectrumWallet

// var _ = (*wallet.ElectrumWallet)(nil)

// BtcElectrumWallet implements ElectrumWallet

type BtcElectrumWallet struct {
	params *chaincfg.Params

	masterPrivateKey *hdkeychain.ExtendedKey
	masterPublicKey  *hdkeychain.ExtendedKey

	mnemonic string

	feeProvider *wallet.FeeProvider

	repoPath string

	// TODO: maybe a scaled down blockchain with headers of interest to wallet?
	// blockchain  *Blockchain
	txstore    *TxStore
	keyManager *KeyManager

	mutex *sync.RWMutex

	creationDate time.Time

	running bool
}

// TODO: adjust interface simpler
// var _ = wallet.Wallet(&BtcElectrumWallet{})

const WalletVersion = "0.1.0"

func NewBtcElectrumWallet(config *wallet.Config, privPass string) (*BtcElectrumWallet, error) {
	if config.Mnemonic == "" {
		ent, err := bip39.NewEntropy(128)
		if err != nil {
			return nil, err
		}
		mnemonic, err := bip39.NewMnemonic(ent)
		if err != nil {
			return nil, err
		}
		config.Mnemonic = mnemonic
		config.CreationDate = time.Now()
	}
	seed := bip39.NewSeed(config.Mnemonic, "")

	mPrivKey, err := hdkeychain.NewMaster(seed, config.Params)
	if err != nil {
		return nil, err
	}
	mPubKey, err := mPrivKey.Neuter()
	if err != nil {
		return nil, err
	}
	w := &BtcElectrumWallet{
		repoPath:         config.DataDir,
		masterPrivateKey: mPrivKey,
		masterPublicKey:  mPubKey,
		mnemonic:         config.Mnemonic,
		params:           config.Params,
		creationDate:     config.CreationDate,
		feeProvider: wallet.NewFeeProvider(
			config.MaxFee,
			config.HighFee,
			config.MediumFee,
			config.LowFee,
			config.FeeAPI.String(),
			config.Proxy,
		),
		mutex: new(sync.RWMutex),
	}

	w.keyManager, err = NewKeyManager(config.DB.Keys(), w.params, w.masterPrivateKey)
	if err != nil {
		return nil, err
	}

	w.txstore, err = NewTxStore(w.params, config.DB, w.keyManager)
	if err != nil {
		return nil, err
	}
	fmt.Println("Addresses")
	for i, adr := range w.txstore.adrs {
		fmt.Printf("%d %v\n", i, adr)
	}

	return w, nil
}

func (w *BtcElectrumWallet) Start() {
	w.running = true

	/* start the Chain Manager here maybe */
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//
// API
//
//////////////

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

func (w *BtcElectrumWallet) MasterPrivateKey() *hdkeychain.ExtendedKey {
	return w.masterPrivateKey
}

func (w *BtcElectrumWallet) MasterPublicKey() *hdkeychain.ExtendedKey {
	return w.masterPublicKey
}

func (w *BtcElectrumWallet) ChildKey(keyBytes []byte, chaincode []byte, isPrivateKey bool) (*hdkeychain.ExtendedKey, error) {
	parentFP := []byte{0x00, 0x00, 0x00, 0x00}
	var id []byte
	if isPrivateKey {
		id = w.params.HDPrivateKeyID[:]
	} else {
		id = w.params.HDPublicKeyID[:]
	}
	hdKey := hdkeychain.NewExtendedKey(
		id,
		keyBytes,
		chaincode,
		parentFP,
		0,
		0,
		isPrivateKey)
	return hdKey.Derive(0)
}

func (w *BtcElectrumWallet) Mnemonic() string {
	return w.mnemonic
}

func (w *BtcElectrumWallet) CurrentAddress(purpose wallet.KeyPurpose) btcutil.Address {
	key, _ := w.keyManager.GetCurrentKey(purpose)
	addr, _ := key.Address(w.params)
	return btcutil.Address(addr)
}

func (w *BtcElectrumWallet) NewAddress(purpose wallet.KeyPurpose) btcutil.Address {
	i, _ := w.txstore.Keys().GetUnused(purpose)
	key, _ := w.keyManager.generateChildKey(purpose, uint32(i[1]))
	addr, _ := key.Address(w.params)
	w.txstore.Keys().MarkKeyAsUsed(addr.ScriptAddress())
	w.txstore.PopulateAdrs()
	return btcutil.Address(addr)
}

func (w *BtcElectrumWallet) DecodeAddress(addr string) (btcutil.Address, error) {
	return btcutil.DecodeAddress(addr, w.params)
}

func (w *BtcElectrumWallet) ScriptToAddress(script []byte) (btcutil.Address, error) {
	return scriptToAddress(script, w.params)
}

func scriptToAddress(script []byte, params *chaincfg.Params) (btcutil.Address, error) {
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(script, params)
	if err != nil {
		return &btcutil.AddressPubKeyHash{}, err
	}
	if len(addrs) == 0 {
		return &btcutil.AddressPubKeyHash{}, errors.New("unknown script")
	}
	return addrs[0], nil
}

func (w *BtcElectrumWallet) AddressToScript(addr btcutil.Address) ([]byte, error) {
	return txscript.PayToAddrScript(addr)
}

func (w *BtcElectrumWallet) HasKey(addr btcutil.Address) bool {
	_, err := w.keyManager.GetKeyForScript(addr.ScriptAddress())
	return err == nil
}

func (w *BtcElectrumWallet) GetKey(addr btcutil.Address) (*btcec.PrivateKey, error) {
	key, err := w.keyManager.GetKeyForScript(addr.ScriptAddress())
	if err != nil {
		return nil, err
	}
	return key.ECPrivKey()
}

func (w *BtcElectrumWallet) ListAddresses() []btcutil.Address {
	keys := w.keyManager.GetKeys()
	addrs := []btcutil.Address{}
	for _, k := range keys {
		addr, err := k.Address(w.params)
		if err != nil {
			continue
		}
		addrs = append(addrs, addr)
	}
	return addrs
}

func (w *BtcElectrumWallet) ListKeys() []btcec.PrivateKey {
	keys := w.keyManager.GetKeys()
	list := []btcec.PrivateKey{}
	for _, k := range keys {
		priv, err := k.ECPrivKey()
		if err != nil {
			continue
		}
		list = append(list, *priv)
	}
	return list
}

func (w *BtcElectrumWallet) Balance() (confirmed, unconfirmed int64) {
	utxos, _ := w.txstore.Utxos().GetAll()
	stxos, _ := w.txstore.Stxos().GetAll()
	for _, utxo := range utxos {
		if !utxo.WatchOnly {
			if utxo.AtHeight > 0 {
				confirmed += utxo.Value
			} else {
				if w.checkIfStxoIsConfirmed(utxo, stxos) {
					confirmed += utxo.Value
				} else {
					unconfirmed += utxo.Value
				}
			}
		}
	}
	return confirmed, unconfirmed
}

func (w *BtcElectrumWallet) Transactions() ([]wallet.Txn, error) {
	height := w.ChainTip()
	txns, err := w.txstore.Txns().GetAll(false)
	if err != nil {
		return txns, err
	}
	for i, tx := range txns {
		var confirmations int64
		var status wallet.StatusCode
		confs := height - tx.Height + 1
		if tx.Height <= 0 {
			confs = tx.Height
		}
		switch {
		case confs < 0:
			status = wallet.StatusDead
		case confs == 0 && time.Since(tx.Timestamp) <= time.Hour*6:
			status = wallet.StatusUnconfirmed
		case confs == 0 && time.Since(tx.Timestamp) > time.Hour*6:
			status = wallet.StatusDead
		case confs > 0 && confs < 6:
			status = wallet.StatusPending
			confirmations = confs
		case confs > 5:
			status = wallet.StatusConfirmed
			confirmations = confs
		}
		tx.Confirmations = confirmations
		tx.Status = status
		txns[i] = tx
	}
	return txns, nil
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
			var addr btcutil.Address
			_, addrs, _, err := txscript.ExtractPkScriptAddrs(out.PkScript, w.params)
			if err != nil {
				fmt.Printf("error extracting address from txn pkscript: %v\n", err)
				return txn, err
			}
			if len(addrs) == 0 {
				addr = nil
			} else {
				addr = addrs[0]
			}
			tout := wallet.TransactionOutput{
				Address: addr,
				Value:   out.Value,
				Index:   uint32(i),
			}
			outs = append(outs, tout)
		}
		txn.Outputs = outs
	}
	return txn, err
}

func (w *BtcElectrumWallet) GetConfirmations(txid chainhash.Hash) (int64, int64, error) {
	txn, err := w.txstore.Txns().Get(txid)
	if err != nil {
		return 0, 0, err
	}
	if txn.Height == 0 {
		return 0, 0, nil
	}
	chainTip := w.ChainTip()
	return chainTip - txn.Height + 1, txn.Height, nil
}

func (w *BtcElectrumWallet) checkIfStxoIsConfirmed(utxo wallet.Utxo, stxos []wallet.Stxo) bool {
	for _, stxo := range stxos {
		if !stxo.Utxo.WatchOnly {
			if stxo.SpendTxid.IsEqual(&utxo.Op.Hash) {
				if stxo.SpendHeight > 0 {
					return true
				} else {
					return w.checkIfStxoIsConfirmed(stxo.Utxo, stxos)
				}
			} else if stxo.Utxo.IsEqual(&utxo) {
				if stxo.Utxo.AtHeight > 0 {
					return true
				} else {
					return false
				}
			}
		}
	}
	return false
}

func (w *BtcElectrumWallet) Params() *chaincfg.Params {
	return w.params
}

func (w *BtcElectrumWallet) ChainTip() int64 {
	// not yet implemented - Get from ElectrumX
	return 0
}

func (w *BtcElectrumWallet) ExchangeRates() wallet.ExchangeRates {
	// not yet implemented
	return nil
}

// Get the current fee per byte
func (w *BtcElectrumWallet) GetFeePerByte(feeLevel wallet.FeeLevel) uint64 {
	// not yet implemented
	return 0
}

// Send bitcoins to an external wallet
func (w *BtcElectrumWallet) Spend(amount int64, addr btcutil.Address, feeLevel wallet.FeeLevel) (*chainhash.Hash, error) {
	// not yet implemented
	return nil, wallet.ErrWalletFnNotImplemented
}

// Bump the fee for the given transaction
func (w *BtcElectrumWallet) BumpFee(txid chainhash.Hash) (*chainhash.Hash, error) {
	return nil, wallet.ErrWalletFnNotImplemented
}

// Calculates the estimated size of the transaction and returns the total fee for the given feePerByte
func (w *BtcElectrumWallet) EstimateFee(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, feePerByte uint64) uint64 {
	// not yet implemented
	return 0
}

// Build and broadcast a transaction that sweeps all coins from an address. If it is a p2sh multisig, the redeemScript must be included
func (w *BtcElectrumWallet) SweepAddress(utxos []wallet.Utxo, address *btcutil.Address, key *hd.ExtendedKey, redeemScript *[]byte, feeLevel wallet.FeeLevel) (*chainhash.Hash, error) {
	// not yet implemented
	return nil, wallet.ErrWalletFnNotImplemented
}

// Create a signature for a multisig transaction
func (w *BtcElectrumWallet) CreateMultisigSignature(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, key *hd.ExtendedKey, redeemScript []byte, feePerByte uint64) ([]wallet.Signature, error) {
	// not yet implemented
	return nil, wallet.ErrWalletFnNotImplemented
}

// Combine signatures and optionally broadcast
func (w *BtcElectrumWallet) Multisign(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, sigs1 []wallet.Signature, sigs2 []wallet.Signature, redeemScript []byte, feePerByte uint64, broadcast bool) ([]byte, error) {
	// not yet implemented
	return nil, wallet.ErrWalletFnNotImplemented
}

// Generate a multisig script from public keys. If a timeout is included the returned script should be a timelocked escrow which releases using the timeoutKey.
func (w *BtcElectrumWallet) GenerateMultisigScript(keys []hd.ExtendedKey, threshold int, timeout time.Duration, timeoutKey *hd.ExtendedKey) (addr btcutil.Address, redeemScript []byte, err error) {
	// not yet implemented
	return nil, nil, wallet.ErrWalletFnNotImplemented

}

// Add a script to the wallet and get notifications back when coins are received or spent from it
func (w *BtcElectrumWallet) AddWatchedScript(script []byte) error {
	// not yet implemented
	return wallet.ErrWalletFnNotImplemented
}

// AddTransactionListener
func (w *BtcElectrumWallet) AddTransactionListener(listener func(wallet.TransactionCallback)) {
	// not yet implemented
}

// NotifyTransactionListners
func (w *BtcElectrumWallet) NotifyTransactionListners(cb wallet.TransactionCallback) {
	// not yet implemented
}

func (w *BtcElectrumWallet) ReSyncBlockchain(fromHeight uint64) {
	panic("ReSyncBlockchain: Not implemented - Non-SPV wallet")
}

func (w *BtcElectrumWallet) AddWatchedAddresses(addrs ...btcutil.Address) error {
	var err error
	var watchedScripts [][]byte

	for _, addr := range addrs {
		script, err := w.AddressToScript(addr)
		if err != nil {
			return err
		}
		watchedScripts = append(watchedScripts, script)
	}

	err = w.txstore.WatchedScripts().PutAll(watchedScripts)
	w.txstore.PopulateAdrs()

	// w.wireService.MsgChan() <- updateFiltersMsg{} // not SPV

	return err
}

func (w *BtcElectrumWallet) DumpHeaders(writer io.Writer) {
	// w.blockchain.db.Print(writer)
	panic("DumpHeaders: Non-SPV wallet")
}

func (w *BtcElectrumWallet) Close() {
	if w.running {
		// Any other tear down here
		w.running = false
	}
}
