package bdb

import (
	"sync"
	"time"

	"github.com/boltdb/bolt"
)

type CfgDB struct {
	db   *bolt.DB
	lock *sync.RWMutex
}

var creationKey = []byte("creationDate")

func (c *CfgDB) PutCreationDate(creationDate time.Time) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	creationValue, err := creationDate.GobEncode()
	if err != nil {
		return err
	}
	return c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(configBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		err := b.Put(creationKey, creationValue)
		return err
	})
}

func (c *CfgDB) GetCreationDate() (time.Time, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	t := time.Time{}
	var creationValue []byte
	e := c.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(configBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		creationValue = b.Get(creationKey)
		return t.GobDecode(creationValue)
	})
	return t, e
}
