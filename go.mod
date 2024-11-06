module github.com/dev-warrior777/go-electrum-client

go 1.19

require (
	github.com/btcsuite/btcd v0.24.0
	github.com/btcsuite/btcd/btcec/v2 v2.3.2
	github.com/btcsuite/btcd/btcutil v1.1.5
	github.com/btcsuite/btcd/chaincfg/chainhash v1.1.0
	github.com/btcsuite/btcwallet/wallet/txauthor v1.3.4
	github.com/decred/dcrd/crypto/rand v1.0.0
	github.com/decred/go-socks v1.1.0
	github.com/mattn/go-sqlite3 v1.14.22
	github.com/spf13/cast v1.6.0
	github.com/tyler-smith/go-bip39 v1.1.0
	go.etcd.io/bbolt v1.3.9
	golang.org/x/net v0.25.0
)

replace go.etcd.io/bbolt => github.com/etcd-io/bbolt v1.3.9

require (
	github.com/aead/siphash v1.0.1 // indirect
	github.com/btcsuite/btcwallet/wallet/txsizes v1.2.4 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/kkdai/bstream v1.0.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
)

require (
	github.com/btcsuite/btclog v0.0.0-20170628155309-84c8d2346e9f // indirect
	github.com/btcsuite/btcwallet/wallet/txrules v1.2.1
	github.com/decred/dcrd/crypto/blake256 v1.0.1 // indirect
	github.com/decred/dcrd/crypto/ripemd160 v1.0.2
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0
	golang.org/x/crypto v0.24.0
	golang.org/x/sys v0.21.0 // indirect
)
