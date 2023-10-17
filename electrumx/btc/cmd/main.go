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

	ex "github.com/dev-warrior777/go-electrum-client/electrumx"
)

var (
	simnetServerAddr = "127.0.0.1:54002"
	simnetTx         = "9298133b60ac679b01e4d407c552d9ac0866ea40c52501dc1d39fdd57b9e9b5f"
	simnetGenesis    = "0f9188f13cb7b2c71f2a335e3a4fc328bf5beb436012afca590b1a11466e2206"
)

func main() {
	RunNode(ex.Regtest, simnetServerAddr, simnetTx, simnetGenesis, true)
}

func RunNode(network ex.Network, addr, tx, genesis string, useTls bool) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	// ctx, cancel := contexlog.WithTimeout(contexlog.Background(), 45*time.Second)
	defer cancel()

	ex.DebugMode = true

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
