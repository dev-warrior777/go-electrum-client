package wltbtc

import (
	"fmt"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
)

func createSubscribeManager() *SubscribeManager {
	scripts := make(map[string][]byte)
	return NewSubscribeManager(&mockSubscribeScriptsStore{scripts: scripts}, &chaincfg.MainNetParams)
}

func TestStoreSubscribeScript(t *testing.T) {
	sm := createSubscribeManager()
	req := []byte("paymentscript")
	err := sm.datastore.Put(req)
	if err != nil {
		t.Fatal(err)
	}

	ret, err := sm.datastore.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range ret {
		if string(s) == string(req) {
			return
		}
	}
	t.Fatal("req != ret")
}

func TestDeleteSubscribeScript(t *testing.T) {
	sm := createSubscribeManager()
	req := []byte("paymentscript")
	err := sm.datastore.Put(req)
	if err != nil {
		t.Fatal(err)
	}

	err = sm.datastore.Delete(req)
	if err != nil {
		t.Fatal(err)
	}

	ret, err := sm.datastore.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range ret {
		fmt.Println(ret)
		if string(s) == string(req) {
			t.Fatal("req != ret")
		}
	}
}
