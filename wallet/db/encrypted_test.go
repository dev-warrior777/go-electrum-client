package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sync"
	"testing"
)

const pw = "abc"

var enc EncDB

func init() {
	conn, _ := sql.Open("sqlite3", ":memory:")
	initDatabaseTables(conn)
	enc = EncDB{
		db:   conn,
		lock: new(sync.RWMutex),
	}
}

func TestEncryptBytes(t *testing.T) {
	b := make([]byte, 32)
	rand.Read(b)
	sb := hex.EncodeToString(b)
	fmt.Println(sb)
	err := enc.Encrypt(b, pw)
	if err != nil {
		t.Fatal(err)
	}
	ret, err := enc.Decrypt(pw)
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
