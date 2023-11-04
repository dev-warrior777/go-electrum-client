package btc

import (
	"encoding/hex"
	"fmt"
	"log"
	"time"
)

// SyncHeaders reads blockchain_headers file, then gets any missing block from
// end of file to current tip from server. The current set of headers is also
// stored in headers map and the chain veirfied by checking previous block
// hashes backwards from Tip.
func (ec *BtcElectrumClient) SyncHeaders() error {
	headers, err := NewHeaders(ec.ClientConfig)
	if err != nil {
		return err
	}

	// 1. Read last stored blockchain_headers file for this network

	b, err := headers.ReadAllBytesFromFile()
	if err != nil {
		return err
	}
	lb := len(b)
	fmt.Println("read header bytes", lb)
	numHeaders, err := headers.BytesToNumHdrs(lb)
	if err != nil {
		return err
	}
	b = nil // gc

	maybeTip := numHeaders - 1

	// 2. Gather new block headers we did not have in file up to current tip

	// Do not make block count too big or electrumX may throttle response
	// as an anti ddos measure. Magic number 2016 from electrum code
	const blockDelta = 20 // 20 dev 2016 pro
	var doneGathering = false
	var startHeight = uint32(numHeaders)
	var blockCount = uint32(20)

	n := ec.GetNode()

	hdrsRes, err := n.BlockHeaders(startHeight, blockCount)
	if err != nil {
		return err
	}
	count := hdrsRes.Count

	fmt.Print("Count: ", count, " read from server at Height: ", startHeight)

	if count > 0 {
		b, err := hex.DecodeString(hdrsRes.HexConcat)
		if err != nil {
			log.Fatal(err)
		}
		nh, err := headers.AppendHeaders(b)
		if err != nil {
			log.Fatal(err)
		}
		maybeTip += int32(count)

		fmt.Println(" Appended: ", nh, " headers at ", startHeight, " maybeTip ", maybeTip)
	}

	if count < blockDelta {
		fmt.Println("\nDone gathering")
		doneGathering = true
	}

	sc, err := n.GetServerConn()
	if err != nil {
		return err
	}
	svrCtx := sc.SvrCtx

	for !doneGathering {

		startHeight += blockDelta

		select {
		case <-svrCtx.Done():
			fmt.Println("Server shutdown - gathering")
			n.Stop()
			return nil
		case <-time.After(time.Millisecond * 33):
			hdrsRes, err := n.BlockHeaders(startHeight, blockCount)
			if err != nil {
				return err
			}
			count = hdrsRes.Count

			fmt.Print("Count: ", count, " read from server at Height: ", startHeight)

			if count > 0 {
				b, err := hex.DecodeString(hdrsRes.HexConcat)
				if err != nil {
					return err
				}
				nh, err := headers.AppendHeaders(b)
				if err != nil {
					return err
				}
				maybeTip += int32(count)

				fmt.Println(" Appended: ", nh, " headers at ", startHeight, " maybeTip ", maybeTip)
			}

			if count < blockDelta {
				fmt.Println("\nDone gathering")
				doneGathering = true
			}
		}
	}

	// 3. Read up to date blockchain_headers file - this can be improved since
	//    we already read most of it but for now; simplicity
	b2, err := headers.ReadAllBytesFromFile()
	if err != nil {
		return err
	}

	// 4. Store all headers in a map
	err = headers.Store(b2, 0)
	if err != nil {
		return err
	}
	headers.hdrsTip = maybeTip

	// 5. Verify headers in headers map
	fmt.Printf("starting verify at height %d\n", headers.hdrsTip)
	err = headers.VerifyAll()
	if err != nil {
		return err
	}
	fmt.Println("header chain verified")

	headers.synced = true
	fmt.Println("headers synced up to tip ", headers.hdrsTip)
	return nil
}

//////////////////////////////////////////////////////////////////////////////
// Btc
//////
// func (ec *BtcElectrumClient) Foo() error {
