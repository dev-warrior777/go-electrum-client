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

// func (w *BtcElectrumWallet) ConnectedPeers() []*peer.Peer {
// 	// return w.peerManager.ConnectedPeers()
// 	panic("ConnectedPeers: Non-SPV wallet")
// }

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
	height, _ := w.ChainTip()
	txns, err := w.txstore.Txns().GetAll(false)
	if err != nil {
		return txns, err
	}
	for i, tx := range txns {
		var confirmations int32
		var status wallet.StatusCode
		confs := int32(height) - tx.Height + 1
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
		tx.Confirmations = int64(confirmations)
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

func (w *BtcElectrumWallet) GetConfirmations(txid chainhash.Hash) (uint32, uint32, error) {
	txn, err := w.txstore.Txns().Get(txid)
	if err != nil {
		return 0, 0, err
	}
	if txn.Height == 0 {
		return 0, 0, nil
	}
	chainTip, _ := w.ChainTip()
	return chainTip - uint32(txn.Height) + 1, uint32(txn.Height), nil
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

func (w *BtcElectrumWallet) AddTransactionListener(callback func(wallet.TransactionCallback)) {
	w.txstore.listeners = append(w.txstore.listeners, callback)
}

func (w *BtcElectrumWallet) ChainTip() (uint32, chainhash.Hash) {
	// var ch chainhash.Hash
	// sh, err := w.blockchain.db.GetBestHeader()
	// if err != nil {
	// 	return 0, ch
	// }
	// return sh.height, sh.header.BlockHash()
	panic("ChainTip: Non-SPV wallet -- get from ElectrumX")
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

	// w.wireService.MsgChan() <- updateFiltersMsg{}

	return err
}

func (w *BtcElectrumWallet) DumpHeaders(writer io.Writer) {
	// w.blockchain.db.Print(writer)
	panic("DumpHeaders: Non-SPV wallet")
}

func (w *BtcElectrumWallet) Close() {
	if w.running {
		// log.Info("Disconnecting from peers and shutting down")
		// w.peerManager.Stop()
		// w.blockchain.Close()
		// w.wireService.Stop()
		w.running = false
	}
}

func (w *BtcElectrumWallet) ReSyncBlockchain(fromDate time.Time) {
	// w.blockchain.Rollback(fromDate)
	// w.txstore.PopulateAdrs()
	// w.wireService.Resync()
	panic("ReSyncBlockchain: Non-SPV wallet")
}

func (w *BtcElectrumWallet) ExchangeRates() wallet.ExchangeRates {
	panic("ExchangeRates: not implemented yet")
}

// AssociateTransactionWithOrder used for ORDER_PAYMENT message
func (w *BtcElectrumWallet) AssociateTransactionWithOrder(cb wallet.TransactionCallback) {
	for _, l := range w.txstore.listeners {
		go l(cb)
	}
}
