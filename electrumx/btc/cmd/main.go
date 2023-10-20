package main

// <https://electrumx.readthedocs.io/en/latest/protocol-methods.html>

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dev-warrior777/go-electrum-client/client"
	ex "github.com/dev-warrior777/go-electrum-client/electrumx"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

const (
	coinDir = "btc"
	// raw bitcoin headers from last checkpoint. For regtest that means from
	// genesis - so no need to check checkpoint merkle proofs
	headerFilename = "blockchain_headers"
)

var (
	simnetServerAddr = "127.0.0.1:53002"
	simnetTx         = ""
	simnetGenesis    = "0f9188f13cb7b2c71f2a335e3a4fc328bf5beb436012afca590b1a11466e2206"
)

func makeBitcoinRegtestConfig() (*client.Config, error) {
	cfg := client.NewDefaultConfig()
	cfg.Chain = wallet.Bitcoin
	cfg.Params = &chaincfg.RegressionNetParams
	cfg.StoreEncSeed = true
	appDir, err := client.GetConfigPath()
	if err != nil {
		return nil, err
	}
	regtestDir := filepath.Join(appDir, coinDir, "regtest")
	err = os.MkdirAll(regtestDir, os.ModeDir|0777)
	if err != nil {
		return nil, err
	}
	cfg.DataDir = regtestDir
	return cfg, nil
}

func openBlockchainHeaders(config *client.Config) (*os.File, error) {
	headerFilePath := filepath.Join(config.DataDir, headerFilename)
	headerFile, err := os.OpenFile(headerFilePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0664)
	if err != nil {
		return nil, err
	}
	return headerFile, nil
}

func main() {
	RunNode(ex.Regtest, simnetServerAddr, simnetTx, simnetGenesis, true)
}

func RunNode(network ex.Network, addr, tx, genesis string, useTls bool) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	// ctx, cancel := contexlog.WithTimeout(contexlog.Background(), 45*time.Second)
	defer cancel()

	ex.DebugMode = true

	config, err := makeBitcoinRegtestConfig()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(config.DataDir)
	headerFile, err := openBlockchainHeaders(config)
	if err != nil {
		log.Fatal(err)
	}
	defer headerFile.Close()

	headerFile.Write([]byte{0x41})
	headerFile.Write([]byte{0x42})
	headerFile.Write([]byte{0x43})

	if true {
		os.Exit(0xff)
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}
	fmt.Println(sc.Proto())

	fmt.Printf("\n\n ** Connected to %s **\n\n", network)

	feats, err := sc.Features(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if feats.Genesis != genesis {
		log.Fatalf("wrong genesis hash for Bitcoin on %s: %s",
			feats.Genesis, network)
	}
	fmt.Println("Genesis correct: ", "0x"+feats.Genesis)

	////////////////////////////////////////////////////////////////////////
	// Gather blocks
	////////////////
	// // Do not make block count too big or electrumX may throttle response
	// // as an anti ddos measure
	// var startHeight = uint32(0) // or wallet birthday
	// var blockCount = uint32(7)
	// hdrsRes, err := sc.BlockHeaders(ctx, startHeight, blockCount)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// spew.Dump(hdrsRes)

	fmt.Println("\n\n================= Running =================")
	// read whatever is in the queue
	_, hdrResNotify, err := sc.SubscribeHeaders(ctx)
	if err != nil {
		log.Fatal(err)
	}

out:
	for {
		select {
		case <-ctx.Done():
			break out
		case <-hdrResNotify:
			// read whatever is in the queue
			for x := range hdrResNotify {
				fmt.Println("New Block: ", x.Height, x.Hex)
			}
		}
	}
	// server shutdown
	sc.Shutdown()
	<-sc.Done()
}
