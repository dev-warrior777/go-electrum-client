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

const STORAGE = "storage"

var (
	ErrBadPw = errors.New("bad password")
	// Argon2 params
	SALT    = []byte("2977958431d29f2d") // TODO: a good random for dev
	TIME    = uint32(1)
	MEM     = uint32(64 * 1024)
	THREADS = uint8(runtime.NumCPU())
	THRDMAX = uint8(255)
	KEYLEN  = uint32(32)
)

type EncDB struct {
	db   *bolt.DB
	lock *sync.RWMutex
}

var storageKey = []byte(STORAGE)

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
	threads := THREADS
	if threads > THRDMAX {
		threads = THRDMAX
	}
	b := argon2.IDKey([]byte(password), SALT, TIME, MEM, threads, KEYLEN)
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
