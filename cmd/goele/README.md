# The Goele BTC Test App

The `goele` app can be used for functional testing of the `go-electrum-client` library. The app  loads a BTC regtest client wallet by default. Testnet can also be used. 

Wallets are at `/home/<user>/.config/goele/btc/<network>`. Regtest wallets should be deleted as necessary.

Use the harness scripts at `client/btc/test_harness`. When goele starts navigate to `client/btc/rpctest` and use the rpc test client.

## RPC Test client

```bash
rpctest v0.1.0

usage:
  cmd [positional args]

  help 					            This help
  echo <any> 				        Echo any input args - test only
  tip 					            Get blockchain tip
  getbalance 				        Get wallet confirmed & unconfirmed balance
  listunspent 				        List all wallet utxos
  getunusedaddress 			        Get a new unused wallet receive address
  getchangeaddress 			        Get a new unused wallet change address
  spend pw amount address feeType   Make signed transaction from wallet utxos
  broadcast rawTx changeIndex 		Broadcast rawTx to ElectrumX
```
