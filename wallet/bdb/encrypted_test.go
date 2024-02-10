package bdb

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/boltdb/bolt"
)

const pw = "abc"

var enc *EncDB

func setupEnc() error {
	bdb, err := bolt.Open("test.bdb", 0600, nil)
	if err != nil {
		return nil
	}
	err = initDatabaseBuckets(bdb)
	if err != nil {
		return nil
	}
	enc = &EncDB{
		db:   bdb,
		lock: new(sync.RWMutex),
	}
	return nil
}

func teardownEnc() {
	if enc == nil {
		return
	}
	enc.db.Close()
	os.RemoveAll("test.bdb")
}

func TestEncryptDecryptBytes(t *testing.T) {
	if err := setupEnc(); err != nil {
		t.Fatal("cannot create database")
	}
	defer teardownEnc()
	b := make([]byte, 32)
	rand.Read(b)
	sb := hex.EncodeToString(b)
	fmt.Println(sb)
	err := enc.PutEncrypted(b, pw)
	if err != nil {
		t.Fatal(err)
	}

	ret, err := enc.GetDecrypted(pw)
	if err != nil {
		t.Fatal(err)
	}
	sret := hex.EncodeToString(ret)
	fmt.Println(sret)
	if sb != sret {
		t.Fatalf("before: %s\n Not equal to\nafter:  %s\n", sb, sret)
	}
}

func TestWif(t *testing.T) {
	err := PrivKeyToWif()
	if err != nil {
		t.Fatal(err)
	}
}
