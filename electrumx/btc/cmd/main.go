package main

// <https://electrumx.readthedocs.io/en/latest/protocol-methods.html>

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
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

var headerFilePath string

func openBlockchainHeadersForAppend(config *client.Config) (*os.File, error) {
	headerFilePath = filepath.Join(config.DataDir, headerFilename)
	headerFile, err := os.OpenFile(headerFilePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0664)
	if err != nil {
		return nil, err
	}
	return headerFile, nil
}

func openBlockchainHeadersForReadWrite(config *client.Config) (*os.File, error) {
	headerFilePath = filepath.Join(config.DataDir, headerFilename)
	headerFile, err := os.OpenFile(headerFilePath, os.O_CREATE|os.O_RDWR, 0664)
	if err != nil {
		return nil, err
	}
	return headerFile, nil
}

// // reverse - reverses bytes received in network byte order
// func reverse(s []byte) []byte {
// 	var d = make([]byte, len(s))
// 	for i, j := 0, len(s)-1; i < len(s); i, j = i+1, j-1 {
// 		d[j] = s[i]
// 	}
// 	return d
// }

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

	fmt.Println("\n\n================= Running =================")

	////////////////////////////////////////////////////////////////////////
	// Get stored blocks
	////////////////

	// create if noexist
	headerFile, err := openBlockchainHeadersForReadWrite(config)
	if err != nil {
		log.Fatal(err)
	}
	defer headerFile.Close()

	fi, err := os.Stat(headerFilePath)
	if err != nil {
		fmt.Println(err.Error())
		log.Fatal(err)
	}

	var numHeaders = 0
	fsize := fi.Size()

	numHeaders = int(fsize / 80)
	er := fsize % 80
	if er != 0 {
		log.Fatal("corrupted file")
	}

	fmt.Println("blockchain_headers - Size: ", fsize, " Heasers: ", numHeaders)

	headerBuf := make([]byte, fsize)
	n, err := headerFile.Read(headerBuf)
	if err != nil {
		log.Fatal(err)
	}
	if int64(n) != fsize {
		log.Fatal("corrupted file")
	}
	// store locally, etc...

	headerFile.Close()

	////////////////////////////////////////////////////////////////////////
	// Gather blocks - catch up from previous stored height
	////////////////
	headerFile, err = openBlockchainHeadersForAppend(config)
	if err != nil {
		log.Fatal(err)
	}
	defer headerFile.Close()

	// Do not make block count too big or electrumX may throttle response
	// as an anti ddos measure. Magic number 2016 from electrum code
	const blockDelta = 2016
	var done_gathering = false
	var startHeight = uint32(numHeaders) // or wallet birthday
	var blockCount = uint32(2016)
	hdrsRes, err := sc.BlockHeaders(ctx, startHeight, blockCount)
	if err != nil {
		log.Fatal(err)
	}
	count := hdrsRes.Count

	fmt.Println("Count: ", count, " read from server at Height: ", startHeight)

	if count > 0 {
		b, err := hex.DecodeString(hdrsRes.HexConcat)
		if err != nil {
			log.Fatal(err)
		}
		_, err = headerFile.Write(b)
		if err != nil {
			log.Fatal(err)
		}
	}

	if count < blockDelta {
		fmt.Println("Done gathering")
		done_gathering = true
	}

	startHeight += blockDelta

	if !done_gathering {
		var nxtHdr = time.Millisecond * 30
	outCtxDone:
		for {
			select {
			case <-ctx.Done():
				break outCtxDone
			case <-time.After(nxtHdr):
				hdrsRes, err := sc.BlockHeaders(ctx, startHeight, blockCount)
				if err != nil {
					fmt.Println(err)
					goto out1
				}
				count = hdrsRes.Count

				fmt.Println("Count: ", count, " read from Height: ", startHeight)

				if count > 0 {
					b, err := hex.DecodeString(hdrsRes.HexConcat)
					if err != nil {
						fmt.Println(err)
						goto out1
					}
					_, err = headerFile.Write(b)
					if err != nil {
						fmt.Println(err)
						goto out1
					}
				}

				if count < blockDelta {
					fmt.Println("Done gathering")
					goto out1
				}

				startHeight += blockDelta
				nxtHdr = time.Second
			}
		}
	}
out1:
	headerFile.Close()

	// debug: read back stored raw headers, deserialize & print ->
	headerFile, err = openBlockchainHeadersForReadWrite(config)
	if err != nil {
		log.Fatal(err)
	}
	defer headerFile.Close()

	fi, _ = os.Stat(headerFilePath)
	fsize = fi.Size()
	fmt.Println("blockchain_headers - Size: ", fsize)

	b := make([]byte, fi.Size())
	numBytes, err := headerFile.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("read blockchain_headers - n: ", numBytes)

	hdrBuf := bytes.NewBuffer(b)
	hdr := wire.BlockHeader{}

	var i int
	for i = 0; i < numBytes; i += 80 {
		err = hdr.Deserialize(hdrBuf)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Hash: ", hdr.BlockHash(), "Height: ", i/80)
		fmt.Println("--------------------------")
		fmt.Printf("Version: 0x%08x\n", hdr.Version)
		fmt.Println("Previous Hash: ", hdr.PrevBlock)
		fmt.Println("Merkle Root: ", hdr.MerkleRoot)
		fmt.Println("Time Stamp: ", hdr.Timestamp)
		fmt.Printf("Bits: 0x%08x\n", hdr.Bits)
		fmt.Println("Nonce: ", hdr.Nonce)
		fmt.Println()
		fmt.Println("============================")
	}
	headerFile.Close()
	// debug: End debug <-

	////////////////////////////////////////////////////
	// 	// read whatever is in the queue
	// 	_, hdrResNotify, err := sc.SubscribeHeaders(ctx)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// out:
	// 	for {
	// 		select {
	// 		case <-ctx.Done():
	// 			break out
	// 		case <-hdrResNotify:
	// 			// read whatever is in the queue
	// 			for x := range hdrResNotify {
	// 				fmt.Println("New Block: ", x.Height, x.Hex)
	// 			}
	// 		}
	// 	}

	// server shutdown
	sc.Shutdown()
	<-sc.Done()
}
