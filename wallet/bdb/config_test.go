package bdb

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"
)

var config *CfgDB

func setupCfg() error {
	bdb, err := bolt.Open("test.bdb", 0600, nil)
	if err != nil {
		return nil
	}
	err = initDatabaseBuckets(bdb)
	if err != nil {
		return nil
	}
	config = &CfgDB{
		db:   bdb,
		lock: new(sync.RWMutex),
	}
	return nil
}

func teardownCfg() {
	if config == nil {
		return
	}
	config.db.Close()
	os.RemoveAll("test.bdb")
}

func TestConfig(t *testing.T) {
	if err := setupCfg(); err != nil {
		t.Fatal(err)
	}
	defer teardownCfg()
	time := time.Now()
	fmt.Println(time.String())
	err := config.PutCreationDate(time)
	if err != nil {
		t.Fatal(err)
	}
	time2, err := config.GetCreationDate()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(time2.String())
}
