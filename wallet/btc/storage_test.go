package btc

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
)

func createStorageManager() *StorageManager {

	return NewStorageManager(&mockStorage{}, &chaincfg.MainNetParams)
}

func TestStoreRetreiveBlob(t *testing.T) {
	sm := createStorageManager()
	fmt.Println(sm)
	var req = "ABC"
	err := sm.datastore.Encrypt([]byte(req), "abc")
	if err != nil {
		t.Fatal(err)
	}

	ret, err := sm.datastore.Decrypt("abc")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(req))
	fmt.Println(string(ret))
}

var xprv = "tprv8ZgxMBicQKsPfJU6JyiVdmFAtAzmWmTeEv85nTAHjLQyL35tdP2fAPWDSBBnFqGhhfTHVQMcnZhZDFkzFmCjm1bgf5UDwMAeFUWhJ9Dr8c4"
var xpub = "tpubD6NzVbkrYhZ4YmVtCdP63AuHTCWhg6eYpDis4yCb9cDNAXLfFmrFLt85cLFTwHiDJ9855NiE7cgQdiTGt5mb2RS9RfaxgVDkwBybJWm54Gh"

func TestStoreRetrieveEncryptedStore(t *testing.T) {
	sm := createStorageManager()
	fmt.Println(sm)

	sm.store = &Storage{
		Version: "0.1",
		Xprv:    xprv,
		Xpub:    xpub,
	}

	before := sm.store.String()
	fmt.Print("req: ", before)

	req, err := json.Marshal(sm.store)
	if err != nil {
		t.Fatal(err)
	}

	err = sm.datastore.Encrypt(req, "abc")
	if err != nil {
		t.Fatal(err)
	}

	ret, err := sm.datastore.Decrypt("abc")
	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal(ret, sm.store)
	if err != nil {
		t.Fatal(err)
	}

	after := sm.store.String()

	fmt.Println("ret: ", after)

	if before != after {
		t.Fatal("Storage before != Storage after")
	}
}
