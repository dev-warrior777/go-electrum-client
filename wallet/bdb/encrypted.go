// This code is available on the terms of the project LICENSE.md file,
// also available online at https://blueoakcouncil.org/license/1.0.0.

package bdb

import (
	"errors"
	"runtime"
	"sync"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/decred/dcrd/crypto/rand"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/nacl/secretbox"
)

const Storage = "storage"

var (
	ErrBadPw = errors.New("bad password")
	// Argon2 params
	Salt    = []byte("2977958431d29f2d") // TODO: a good random for dev
	Time    = uint32(1)
	Mem     = uint32(64 * 1024)
	Threads = uint8(runtime.NumCPU())
	ThrdMax = uint8(255)
	KeyLen  = uint32(32)
)

type EncDB struct {
	db   *bolt.DB
	lock *sync.RWMutex
}

var storageKey = []byte(Storage)

func (e *EncDB) PutEncrypted(b []byte, pw string) error {
	// encrypt
	eb, err := encryptBytes(b, pw)
	if err != nil {
		return err
	}
	// store in db
	e.lock.Lock()
	defer e.lock.Unlock()
	value := eb
	err_ok := e.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(encBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		err := b.Put(storageKey, value)
		return err
	})
	return err_ok
}

func (e *EncDB) GetDecrypted(pw string) ([]byte, error) {
	// retreive from db , if exist
	e.lock.RLock()
	defer e.lock.RUnlock()
	var value []byte
	err := e.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(encBkt)
		if b == nil {
			return ErrBucketNotFound
		}
		value = b.Get(storageKey)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return decryptBytes(value, pw)
}

func encryptBytes(unencrypted []byte, password string) ([]byte, error) {
	secretKey := getEncryptionKey32(password)
	var nonce [24]byte
	rand.Read(nonce[:])
	encrypted := secretbox.Seal(nonce[:], unencrypted, &nonce, &secretKey)
	// nonce is the first 24 bytes of encrypted. the rest is the actual
	// encryption result. [nonce 24][ ...the encryption result...]
	return encrypted, nil
}

func decryptBytes(encrypted []byte, password string) ([]byte, error) {
	secretKey := getEncryptionKey32(password)
	var decryptNonce [24]byte
	copy(decryptNonce[:], encrypted[:24])
	decrypted, ok := secretbox.Open(nil, encrypted[24:], &decryptNonce, &secretKey)
	if !ok {
		return nil, errors.New("secretbox decryption error")
	}
	// decrypted is the decryption of the encrypted bytes with the pre-pended
	// plaintext nonce stripped out
	return decrypted, nil
}

func getEncryptionKey32(password string) [32]byte {
	threads := Threads
	if threads > ThrdMax {
		threads = ThrdMax
	}
	b := argon2.IDKey([]byte(password), Salt, Time, Mem, threads, KeyLen)
	// revert to go19
	// return ([32]byte)(b)
	var arr32 [32]byte
	copy(arr32[:], b)
	return arr32
}

/////////////////////////////////
// Testing

func PrivKeyToWif() error {
	var key *btcec.PrivateKey
	key, err := btcec.NewPrivateKey()
	if err != nil {
		return err
	}
	wif, err := btcutil.NewWIF(key, &chaincfg.MainNetParams, false)
	if err != nil {
		return err
	}
	wifStr := wif.String()

	_, err = btcutil.DecodeWIF(wifStr)
	if err != nil {
		return err
	}
	// Can also do this
	key.ToECDSA()
	return nil
}
