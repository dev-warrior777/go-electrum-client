///go:build harness

package elxbtc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"os/signal"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	ex "github.com/dev-warrior777/go-electrum-client/electrumx"
)

var (
	// https://github.com/spesmilo/electrum/blob/cffbe44c07a59a7d6a3d5183181659a57de8d2c0/electrum/servers_testnet.json
	testServerAddr = "testnet.aranguren.org:51001"
	// testServerAddr = "blockstream.info:993"
	// testServerAddr = "gsw6sn27quwf6u3swgra6o7lrp5qau6kt3ymuyoxgkth6wntzm2bjwyd.onion:51001"
	testTx      = "581d837b8bcca854406dc5259d1fb1e0d314fcd450fb2d4654e78c48120e0135"
	testGenesis = "000000000933ea01ad0ee984209779baaec3ced90fa3f408719526f8d77f4943"
	// https://github.com/spesmilo/electrum/blob/cffbe44c07a59a7d6a3d5183181659a57de8d2c0/electrum/servers.json
	mainServerAddr = "elx.bitske.com:50002"
	// mainServerAddr = "eai.coincited.net:50002"
	// mainServerAddr = "bitcoin.aranguren.org:50002" // IPv6
	// mainServerAddr = "blockstream.info:700"
	mainTx      = "f53a8b83f85dd1ce2a6ef4593e67169b90aaeb402b3cf806b37afc634ef71fbc"
	mainGenesis = "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f"
)

func TestRunMainnetNode(t *testing.T) {
	RunNode(t, ex.Mainnet, mainServerAddr, mainTx, mainGenesis, true)
}

func TestRunTestnetNode(t *testing.T) {
	RunNode(t, ex.Testnet, testServerAddr, testTx, testGenesis, false)
}

func RunNode(t *testing.T, network ex.Network, addr, tx, genesis string, useTls bool) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	// ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	ex.DebugMode = true

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}

	rootCAs, _ := x509.SystemCertPool()
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		RootCAs:            rootCAs,
		MinVersion:         tls.VersionTLS12, // works ok
		ServerName:         host,
	}

	if !useTls {
		tlsConfig = nil
	}

	opts := &ex.ConnectOpts{
		TLSConfig:   tlsConfig,
		DebugLogger: ex.StdoutPrinter,
	}

	sc, err := ex.ConnectServer(ctx, addr, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(sc.Proto())

	fmt.Printf("\n\n ** Connected to %s **\n\n", network)

	_, err = sc.Banner(ctx)
	if err != nil {
		t.Fatal(err)
	}

	feats, err := sc.Features(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if feats.Genesis != genesis {
		t.Fatalf("wrong genesis hash for Bitcoin on %s: %s",
			feats.Genesis, network)
	}
	t.Log("Genesis correct")

	_, err = sc.GetTransaction(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}

	const until = time.Second * 7
	fmt.Printf("leaving connection open for %s\n", until)
	select {
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	case <-time.After(until):
	}
}

var (
	// bitcoin genesis mainnet
	bgen           = "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa"
	bgenScriptHash = "8b01df4e368ea28f8dc0423bcf7a4923e3a12d307c875e47a0cfbf90b5c39161"
	// goele wallet regtest
	a1           = "mvP2UeXooRghYvsX7H7XVj78FY49jJw6Sq"
	a1Scripthash = "6036b7e9dcb352f2d7bb4ad0efe0f06e03ba58fad4d16e943a25ae41082d1934"
	// electrum wallet regtest
	ab           = "bcrt1q3fx029uese6mrhvq68u4l6me49refj8maqxvfv"
	abScripthash = "02c21ac0ef859617cbb7ae68943b9af8fb99699d32ea35cb449384aac17b93d5"
)

func TestScripthash(t *testing.T) {
	_, err := addrToScripthash("", &chaincfg.MainNetParams)
	if err == nil {
		t.Fatal(err)
	}

	shGen, err := addrToScripthash(bgen, &chaincfg.MainNetParams)
	if err != nil {
		t.Fatal(err)
	}
	if shGen != bgenScriptHash {
		t.Fatal(err)
	}

	sh1, err := addrToScripthash(a1, &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatal(err)
	}
	if sh1 != a1Scripthash {
		t.Fatal(err)
	}

	shb, err := addrToScripthash(ab, &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatal(err)
	}
	if shb != abScripthash {
		t.Fatal(err)
	}
}
