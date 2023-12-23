package wltbtc

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/dev-warrior777/go-electrum-client/wallet"
	"github.com/tyler-smith/go-bip39"
)

var test_mnemonic = "jungle pair grass super coral bubble tomato sheriff pulp cancel luggage wagon"

func makeRegtestSeed(mnemonic string) ([]byte, error) {
	return bip39.NewSeedWithErrorChecking(test_mnemonic, "")
}

func createTxStore() (*TxStore, *StorageManager) {
	mockDb := MockDatastore{
		&mockConfig{creationDate: time.Now()},
		&mockStorage{blob: make([]byte, 10)},
		&mockKeyStore{make(map[string]*keyStoreEntry)},
		&mockUtxoStore{make(map[string]*wallet.Utxo)},
		&mockStxoStore{make(map[string]*wallet.Stxo)},
		&mockTxnStore{make(map[string]*wallet.Txn)},
		&mockSubscriptionsStore{make(map[string]*wallet.Subscription)},
	}

	var seed []byte
	seed, err := makeRegtestSeed(test_mnemonic)
	if err != nil {
		// make random seed
		seed := make([]byte, 32)
		rand.Read(seed)
		fmt.Println("Made random seed")
	}
	key, _ := hdkeychain.NewMaster(seed, &chaincfg.RegressionNetParams)
	km, _ := NewKeyManager(mockDb.Keys(), &chaincfg.RegressionNetParams, key)
	sm := NewStorageManager(mockDb.Enc(), &chaincfg.RegressionNetParams)
	txStore, _ := NewTxStore(&chaincfg.RegressionNetParams, &mockDb, km)
	return txStore, sm
}

func MockWallet() *BtcElectrumWallet {
	txstore, storageMgr := createTxStore()

	storageMgr.store.Xprv = "tprv8ZgxMBicQKsPfJU6JyiVdmFAtAzmWmTeEv85nTAHjLQyL35tdP2fAPWDSBBnFqGhhfTHVQMcnZhZDFkzFmCjm1bgf5UDwMAeFUWhJ9Dr8c4"
	storageMgr.store.Xpub = "tpubD6NzVbkrYhZ4YmVtCdP63AuHTCWhg6eYpDis4yCb9cDNAXLfFmrFLt85cLFTwHiDJ9855NiE7cgQdiTGt5mb2RS9RfaxgVDkwBybJWm54Gh"
	storageMgr.store.ShaPw = chainhash.HashB([]byte(pw))
	storageMgr.store.Seed = []byte{0x01, 0x02, 0x03}
	storageMgr.store.Imported = [][]byte{{0x01, 0x01, 0x01}, {0x02, 0x02, 0x02}, {0x03, 0x03, 0x03}}

	return &BtcElectrumWallet{
		txstore:        txstore,
		keyManager:     txstore.keyManager,
		storageManager: storageMgr,
		params:         &chaincfg.RegressionNetParams,
		feeProvider:    wallet.DefaultFeeProvider(),
	}
}

func Test_gatherCoins(t *testing.T) {
	w := MockWallet()
	w.blockchainTip = 100
	txid := "6f7a58ad92702601fcbaac0e039943a384f5274a205c16bb8bbab54f9ea2fbad"
	h1, err := chainhash.NewHashFromStr(txid)
	if err != nil {
		t.Error(err)
	}
	key1, err := w.keyManager.GetFreshKey(wallet.EXTERNAL)
	if err != nil {
		t.Error(err)
	}
	addr1, err := key1.Address(&chaincfg.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	script1, err := w.AddressToScript(addr1)
	if err != nil {
		t.Error(err)
	}
	op := wire.OutPoint{
		Hash:  *h1,
		Index: 0,
	}
	utxo := wallet.Utxo{Op: op, ScriptPubkey: script1, AtHeight: 5, Value: 10000}
	err = w.txstore.Utxos().Put(utxo)
	if err != nil {
		t.Error(err)
	}
	coinmap := w.gatherCoins(false)
	for coin, key := range coinmap {
		if !bytes.Equal(coin.PkScript(), script1) {
			t.Error("Pubkey script in coin is incorrect")
		}
		if coin.Index() != 0 {
			t.Error("Returned incorrect index")
		}
		if !coin.Hash().IsEqual(h1) {
			t.Error("Returned incorrect hash")
		}
		if coin.NumConfs() != int64(w.blockchainTip-5) {
			t.Error("Returned incorrect number of confirmations")
		}
		if coin.Value() != 10000 {
			t.Error("Returned incorrect coin value")
		}
		addr2, err := key.Address(&chaincfg.RegressionNetParams)
		if err != nil {
			t.Error(err)
		}
		if addr2.EncodeAddress() != addr1.EncodeAddress() {
			t.Error("Returned incorrect key")
		}
		key.Zero()
	}
	// test freeze
	err = w.txstore.Utxos().Freeze(utxo)
	if err != nil {
		t.Error(err)
	}
	coinmap = w.gatherCoins(false)
	if len(coinmap) > 0 {
		t.Fatal("should be no unfrozen coin in map")
	}
	os.Remove("headers.bin")
}

// We do pure segwit inputs only for now. No mixed inputs. The wallet is segwit
// by default. Later if there is a use case we can add the 30% extra code needed
// for that such as wrapped P2SH-P2WPKH, etc.
func Test_newSegwitTransaction(t *testing.T) {
	w := MockWallet()
	w.blockchainTip = 500

	// A real Tx from harness->goele

	// make one utxo
	txid := "50b636d971e7d4d918d92876d6d53a22ccc960e051f540108056ca4ad6ec080c"
	vout := 0
	// witnessProgram := "0014a30a0cf1da8c0c36ae8d637b674663ccf2b31e45"
	witnessProgram := "a30a0cf1da8c0c36ae8d637b674663ccf2b31e45"
	h1Txid, err := chainhash.NewHashFromStr(txid)
	if err != nil {
		t.Error(err)
	}
	h1OutIndex := uint32(vout)
	h1WitnessProgram, err := hex.DecodeString(witnessProgram)
	if err != nil {
		t.Error(err)
	}

	segwitAddress, swerr := btcutil.NewAddressWitnessPubKeyHash(h1WitnessProgram, w.params)
	if swerr != nil {
		t.Error(err)
	}

	fmt.Println("Segwit address", segwitAddress.EncodeAddress())
	script, err := txscript.PayToAddrScript(segwitAddress)
	if err != nil {
		t.Error(err)
	}
	op := wire.OutPoint{
		Hash:  *h1Txid,
		Index: h1OutIndex,
	}
	err = w.txstore.Utxos().Put(wallet.Utxo{
		Op:           op,
		ScriptPubkey: script,
		AtHeight:     421,
		Value:        10030000})
	if err != nil {
		t.Error(err)
	}

	// to harness ->
	address, err := btcutil.DecodeAddress("bcrt1q322tg0y2hzyp9zztr7d2twdclhqg88anvzxwwr", &chaincfg.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	_, _, err = w.Spend(
		// changeIndex, tx, err := w.Spend(
		"abc",
		int64(3000000),
		address,
		wallet.NORMAL,
		false,
	)
	if err != nil {
		t.Error(err)
	}
	// if tx != nil {
	// 	fmt.Println(tx.TxHash().String(), changeIndex)
	// }
}

func Test_newLegacyTransaction(t *testing.T) {
	w := MockWallet()
	w.blockchainTip = 100
	// make one utxo
	txid := "6f7a58ad92702601fcbaac0e039943a384f5274a205c16bb8bbab54f9ea2fbad"
	h1, err := chainhash.NewHashFromStr(txid)
	if err != nil {
		t.Error(err)
	}
	key1, err := w.keyManager.GetFreshKey(wallet.EXTERNAL)
	if err != nil {
		t.Error(err)
	}
	addr1, err := key1.Address(&chaincfg.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	script1, err := w.AddressToScript(addr1)
	if err != nil {
		t.Error(err)
	}
	op := wire.OutPoint{
		Hash:  *h1,
		Index: 0,
	}
	err = w.txstore.Utxos().Put(wallet.Utxo{Op: op, ScriptPubkey: script1, AtHeight: 5, Value: 200000})
	if err != nil {
		t.Error(err)
	}

	/////////////////////// maybe set up more utxos later ////////////////////

	address, err := btcutil.DecodeAddress("bcrt1q322tg0y2hzyp9zztr7d2twdclhqg88anvzxwwr", &chaincfg.RegressionNetParams)
	if err != nil {
		t.Error(err)
	}
	changeIndex, tx, err := w.Spend(
		"abc",
		int64(100000),
		address,
		wallet.NORMAL,
		false,
	)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(tx.TxHash().String(), changeIndex)
}
