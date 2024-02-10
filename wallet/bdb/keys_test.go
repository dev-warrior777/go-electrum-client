package bdb

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"os"
	"sync"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

var kdb *KeysDB

func setupKdb() error {
	bdb, err := bolt.Open("test.bdb", 0600, nil)
	if err != nil {
		return nil
	}
	err = initDatabaseBuckets(bdb)
	if err != nil {
		return nil
	}
	kdb = &KeysDB{
		db:   bdb,
		lock: new(sync.RWMutex),
	}
	return nil
}

func teardownKdb() {
	if kdb == nil {
		return
	}
	kdb.db.Close()
	os.RemoveAll("test.bdb")
}

func TestGetAll(t *testing.T) {
	if err := setupKdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownKdb()
	for i := 0; i < 100; i++ {
		b := make([]byte, 20)
		rand.Read(b)
		// fmt.Println(hex.EncodeToString(b))
		err := kdb.Put(b, wallet.KeyPath{
			Purpose: wallet.EXTERNAL,
			Index:   i,
		})
		if err != nil {
			t.Error(err)
		}
	}
	all, err := kdb.GetAll()
	if err != nil || len(all) != 100 {
		t.Error("Failed to fetch all keys")
	}
}

func TestPutKey(t *testing.T) {
	if err := setupKdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownKdb()
	b := make([]byte, 20)
	rand.Read(b)
	err := kdb.Put(b, wallet.KeyPath{
		Purpose: wallet.EXTERNAL,
		Index:   0,
	})
	if err != nil {
		t.Error(err)
	}
	krec, err := kdb.get(b)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(b, krec.ScriptAddress) {
		t.Errorf(`Expected %s got %s`, hex.EncodeToString(b), krec.ScriptAddress)
	}
	if krec.Purpose != 0 {
		t.Errorf(`Expected 0 got %d`, krec.Purpose)
	}
	if krec.KeyIndex != 0 {
		t.Errorf(`Expected 0 got %d`, krec.KeyIndex)
	}
	if krec.Used != false {
		t.Errorf(`Expected 0 got %v`, krec.Used)
	}
}

func TestPutDuplicateKey(t *testing.T) {
	if err := setupKdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownKdb()
	b := make([]byte, 20)
	rand.Read(b)
	kdb.Put(b, wallet.KeyPath{
		Purpose: wallet.EXTERNAL,
		Index:   0,
	})
	err := kdb.Put(b, wallet.KeyPath{
		Purpose: wallet.INTERNAL,
		Index:   0,
	})
	if err != nil {
		// Unlike a relational db you can put using the same key.
		t.Error("Expected No duplicate key error")
	}
}

func TestMarkKeyAsUsed(t *testing.T) {
	if err := setupKdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownKdb()
	b := make([]byte, 20)
	rand.Read(b)
	err := kdb.Put(b, wallet.KeyPath{
		Purpose: wallet.EXTERNAL,
		Index:   0,
	})
	if err != nil {
		t.Error(err)
	}
	err = kdb.MarkKeyAsUsed(b)
	if err != nil {
		t.Error(err)
	}
	used, err := kdb.isKeyUsed(b)
	if err != nil {
		t.Error(err)
	}
	if !used {
		t.Errorf(`Expected 1 got %v`, used)
	}
}

func TestGetLastKeyIndex(t *testing.T) {
	if err := setupKdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownKdb()
	var lastExternal []byte
	for i := 0; i < 50; i++ {
		b := make([]byte, 20)
		rand.Read(b)
		err := kdb.Put(b, wallet.KeyPath{
			Purpose: wallet.EXTERNAL,
			Index:   i,
		})
		if err != nil {
			t.Error(err)
		}
		lastExternal = b
	}
	var lastInternal []byte
	for i := 0; i < 50; i++ {
		b := make([]byte, 20)
		rand.Read(b)
		err := kdb.Put(b, wallet.KeyPath{
			Purpose: wallet.INTERNAL,
			Index:   i,
		})
		if err != nil {
			t.Error(err)
		}
		lastInternal = b
	}
	idx, used, err := kdb.GetLastKeyIndex(wallet.EXTERNAL)
	if err != nil || idx != 49 || used != false {
		t.Error("Failed to fetch correct last index")
	}
	kdb.MarkKeyAsUsed(lastExternal)
	_, used, err = kdb.GetLastKeyIndex(wallet.EXTERNAL)
	if err != nil || used != true {
		t.Error("Failed to fetch correct last index")
	}

	idx, used, err = kdb.GetLastKeyIndex(wallet.INTERNAL)
	if err != nil || idx != 49 || used != false {
		t.Error("Failed to fetch correct last index")
	}
	kdb.MarkKeyAsUsed(lastInternal)
	_, used, err = kdb.GetLastKeyIndex(wallet.INTERNAL)
	if err != nil || used != true {
		t.Error("Failed to fetch correct last index")
	}
}

func TestGetPathForKey(t *testing.T) {
	if err := setupKdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownKdb()
	b := make([]byte, 20)
	rand.Read(b)
	err := kdb.Put(b, wallet.KeyPath{
		Purpose: wallet.EXTERNAL,
		Index:   15,
	})
	if err != nil {
		t.Error(err)
	}
	path, err := kdb.GetPathForKey(b)
	if err != nil {
		t.Error(err)
	}
	if path.Index != 15 || path.Purpose != wallet.EXTERNAL {
		t.Error("Returned incorrect key path")
	}
}

func TestKeyNotFound(t *testing.T) {
	if err := setupKdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownKdb()
	b := make([]byte, 20)
	rand.Read(b)
	_, err := kdb.GetPathForKey(b)
	if err == nil {
		t.Error("Return key when it shouldn't have")
	}
}

func TestGetUnsed(t *testing.T) {
	if err := setupKdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownKdb()
	for i := 0; i < 100; i++ {
		b := make([]byte, 20)
		rand.Read(b)
		err := kdb.Put(b, wallet.KeyPath{
			Purpose: wallet.INTERNAL,
			Index:   i,
		})
		if err != nil {
			t.Error(err)
		}
	}
	idx, err := kdb.GetUnused(wallet.INTERNAL)
	if err != nil {
		t.Error("Failed to fetch correct unused")
	}
	if len(idx) != 100 {
		t.Error("Failed to fetch correct unused")
	}
}

func TestGetLookaheadWindows(t *testing.T) {
	if err := setupKdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownKdb()
	for i := 0; i < 100; i++ {
		b := make([]byte, 20)
		rand.Read(b)
		err := kdb.Put(b, wallet.KeyPath{
			Purpose: wallet.EXTERNAL,
			Index:   i,
		})
		if err != nil {
			t.Error(err)
		}
		if i < 50 {
			kdb.MarkKeyAsUsed(b)
		}
		b = make([]byte, 20)
		rand.Read(b)
		err = kdb.Put(b, wallet.KeyPath{
			Purpose: wallet.INTERNAL,
			Index:   1,
		})
		if err != nil {
			t.Error(err)
		}
		if i < 50 {
			kdb.MarkKeyAsUsed(b)
		}
	}
	windows := kdb.GetLookaheadWindows()
	if windows[wallet.EXTERNAL] != 50 || windows[wallet.INTERNAL] != 50 {
		t.Error("Fetched incorrect lookahead windows")
	}
}

func TestDeleteKey(t *testing.T) {
	if err := setupKdb(); err != nil {
		t.Fatal(err)
	}
	defer teardownKdb()
	b := make([]byte, 20)
	rand.Read(b)
	err := kdb.Put(b, wallet.KeyPath{
		Purpose: wallet.EXTERNAL,
		Index:   0,
	})
	if err != nil {
		t.Error(err)
	}
	err = kdb.delete(b)
	if err != nil {
		t.Error(err)
	}
	_, err = kdb.get(b)
	if err == nil {
		t.Error(err)
	}
}
