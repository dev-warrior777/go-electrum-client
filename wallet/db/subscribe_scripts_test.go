package db

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"sync"
	"testing"
)

var ssdb SubscribeScriptsDB

func init() {
	conn, _ := sql.Open("sqlite3", ":memory:")
	initDatabaseTables(conn)
	ssdb = SubscribeScriptsDB{
		db:   conn,
		lock: new(sync.RWMutex),
	}
}

func TestSubscribeScriptsDB_Put(t *testing.T) {
	err := ssdb.Put([]byte("test"))
	if err != nil {
		t.Error(err)
	}
	stmt, _ := ssdb.db.Prepare("select * from subscribeScripts")
	defer stmt.Close()

	var out string
	err = stmt.QueryRow().Scan(&out)
	if err != nil {
		t.Error(err)
	}
	if hex.EncodeToString([]byte("test")) != out {
		t.Error("Failed to inserted watched script into db")
	}
}

func TestSubscribeScriptsDB_GetAll(t *testing.T) {
	err := ssdb.Put([]byte("test"))
	if err != nil {
		t.Error(err)
	}
	err = ssdb.Put([]byte("test2"))
	if err != nil {
		t.Error(err)
	}
	scripts, err := ssdb.GetAll()
	if err != nil {
		t.Error(err)
	}
	if len(scripts) != 2 {
		t.Error("Returned incorrect number of subscribe scripts")
	}
	if !bytes.Equal(scripts[0], []byte("test")) {
		t.Error("Returned incorrect subscribe script")
	}
	if !bytes.Equal(scripts[1], []byte("test2")) {
		t.Error("Returned incorrect subscribe script")
	}
}

func TestSubscribeScriptsDB_Delete(t *testing.T) {
	err := ssdb.Put([]byte("test"))
	if err != nil {
		t.Error(err)
	}
	err = ssdb.Delete([]byte("test"))
	if err != nil {
		t.Error(err)
	}
	scripts, err := ssdb.GetAll()
	if err != nil {
		t.Error(err)
	}
	for _, script := range scripts {
		if bytes.Equal(script, []byte("test")) {
			t.Error("Failed to delete subscribe script")
		}
	}
}
