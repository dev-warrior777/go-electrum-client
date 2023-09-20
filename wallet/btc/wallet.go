package btc

import (
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
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

	// blockchain  *Blockchain
	// txstore    *TxStore
	// keyManager *KeyManager

	mutex *sync.RWMutex

	creationDate time.Time

	running bool

	exchangeRates wallet.ExchangeRates
}

// TODO:
//var _ = wallet.Wallet(BtcElectrumWallet{})

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

	bpf := exchangerates.NewBitcoinPriceFetcher(config.Proxy)
	w.exchangeRates = bpf
	if !config.DisableExchangeRates {
		go bpf.Run()
	}

	w.keyManager, err = NewKeyManager(config.DB.Keys(), w.params, w.masterPrivateKey)

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
