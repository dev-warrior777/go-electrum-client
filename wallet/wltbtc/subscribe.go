package wltbtc

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

type SubscribeManager struct {
	datastore wallet.SubscribeScripts
	params    *chaincfg.Params
}

func NewSubscribeManager(db wallet.SubscribeScripts, params *chaincfg.Params) *SubscribeManager {
	sm := &SubscribeManager{
		datastore: db,
		params:    params,
	}
	return sm
}

func (sm *SubscribeManager) Put(scriptPubKey []byte) error {
	return sm.datastore.Put(scriptPubKey)
}

func (sm *SubscribeManager) GetAll() ([][]byte, error) {
	return sm.datastore.GetAll()
}

func (sm *SubscribeManager) Delete(scriptPubKey []byte) error {
	return sm.datastore.Delete(scriptPubKey)
}
