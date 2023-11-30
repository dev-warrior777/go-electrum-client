package wltbtc

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

func createStorageManager() *StorageManager {
	return NewStorageManager(&mockStorage{}, &chaincfg.MainNetParams)
}

func TestStoreRetreiveBlob(t *testing.T) {
	sm := createStorageManager()
	var req = "ABC"
	err := sm.datastore.PutEncrypted([]byte(req), "abc")
	if err != nil {
		t.Fatal(err)
	}

	ret, err := sm.datastore.GetDecrypted("abc")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(req))
	fmt.Println(string(ret))
}

var pw = "abc" // tested
var xprv = "tprv8ZgxMBicQKsPfJU6JyiVdmFAtAzmWmTeEv85nTAHjLQyL35tdP2fAPWDSBBnFqGhhfTHVQMcnZhZDFkzFmCjm1bgf5UDwMAeFUWhJ9Dr8c4"
var xpub = "tpubD6NzVbkrYhZ4YmVtCdP63AuHTCWhg6eYpDis4yCb9cDNAXLfFmrFLt85cLFTwHiDJ9855NiE7cgQdiTGt5mb2RS9RfaxgVDkwBybJWm54Gh"
var shaPw = chainhash.HashB([]byte(pw))
var seed = []byte{0x01, 0x02, 0x03}
var imported = [][]byte{{0x01, 0x01, 0x01}, {0x02, 0x02, 0x02}, {0x03, 0x03, 0x03}}

func TestStoreRetrieveEncryptedStore(t *testing.T) {
	sm := createStorageManager()

	sm.store = &Storage{
		Version:  "0.1",
		Xprv:     xprv,
		Xpub:     xpub,
		ShaPw:    shaPw,
		Seed:     seed,
		Imported: imported,
	}

	before := sm.store.String()
	fmt.Print("req: ", before)

	err := sm.Put(pw)
	if err != nil {
		t.Fatal(err)
	}

	sm.store.blank()

	err = sm.Get(pw)
	if err != nil {
		t.Fatal(err)
	}

	after := sm.store.String()
	fmt.Println("ret: ", after)

	if before != after {
		t.Fatal("Storage before != Storage after")
	}

	shaPw := chainhash.HashB([]byte(pw))
	if !bytes.Equal(sm.store.ShaPw, shaPw) {
		t.Fatal("pw check failed")
	}

	sm.store.blank()
	afterBlank := sm.store.String()
	fmt.Println("ret: ", afterBlank)

	sm.Get("abc")
	fmt.Println(sm.store.String())
}

func TestValidPw(t *testing.T) {
	sm := createStorageManager()

	sm.store = &Storage{
		Version:  "0.1",
		Xprv:     xprv,
		Xpub:     xpub,
		ShaPw:    shaPw,
		Seed:     seed,
		Imported: imported,
	}

	err := sm.Put(pw)
	if err != nil {
		t.Fatal(err)
	}

	sm.store.blank()

	valid := sm.ValidPw("abc")
	if !valid {
		t.Fatal("invalid pw")
	}
	fmt.Println("valid pw")
	fmt.Println(sm.store.String())
}
