# go-electrum-client

__Goele__ - Golang Electrum Client

## Electrum client wallet in Go

- Electrum client multi coin wallet library
- ElectrumX chain server interface
- No GUI

## Purpose

- Goele is a simplified functional version of an electrum wallet.

- Goele is golang code intended to be used directly by other golang projects; for example a lite trading wallet.

- Goele wallets __do__ have a simple RPC Server to call some of the python console-like methods such as ListUnspent, Spend & Broadcast, etc. But this is for testing only.

## Development example

The __goele__ example app can be found at `cmd/goele/`. This is the tool I have been using for functional testing.

If running against regtest network then it is dependent upon the __BTC ElectrumX RegTest Simnet Harness__ found in `client/btc/test_harness`

It will also run against btc testnet.
