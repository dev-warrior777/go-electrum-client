package electrumx

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/btcsuite/btcd/wire"
)

// syncHeaders uodates the network's headers and then subscribes for new update
// tip notifications and listens for them
func (n *Node) syncHeaders(nodeCtx context.Context) error {
	err := n.syncNetworkHeaders(nodeCtx)
	if err != nil {
		return err
	}
	return nil
}

// syncNetworkHeaders reads blockchain_headers file, then gets any missing block
// from end of file to current tip from server. The current set of headers is
// also stored in headers map and the chain verified by checking previous block
// hashes backwards from local Tip.
func (n *Node) syncNetworkHeaders(nodeCtx context.Context) error {
	h := n.networkHeaders

	// we start from a recent height for testnet/mainnet
	startPointHeight := h.startPoint

	// 1. Read last stored blockchain_headers file for this network

	b, err := h.readAllBytesFromFile()
	if err != nil {
		return err
	}
	lb := int64(len(b))
	fmt.Println("read:", lb, " bytes from header file")
	numHeaders, err := h.bytesToNumHdrs(lb)
	if err != nil {
		return err
	}

	var maybeTip int64 = startPointHeight + numHeaders - 1

	// 2. Gather new block headers we did not have in file up to current tip

	// Do not make block count too big or electrumX may throttle response
	// as an anti ddos measure. ElectrumX Doc.: "Recommended to be at least one
	// bitcoin difficulty retarget period, i.e. 2016."
	var blockCount = 2016
	var doneGathering = false
	var startHeight = startPointHeight + numHeaders

	hdrsRes, err := n.blockHeaders(nodeCtx, startHeight, blockCount)
	if err != nil {
		return err
	}
	count := hdrsRes.Count

	fmt.Printf("read: %d from server at height %d max chunk size %d\n", count, startHeight, hdrsRes.Max)

	if count > 0 {
		b, err := hex.DecodeString(hdrsRes.HexConcat)
		if err != nil {
			log.Fatal(err)
		}
		nh, err := h.appendHeadersFile(b)
		if err != nil {
			log.Fatal(err)
		}
		maybeTip += int64(count)

		fmt.Println(" appended: ", nh, " headers at ", startHeight, " maybeTip ", maybeTip)
	}

	if count < blockCount {
		doneGathering = true
	}

	for !doneGathering {

		startHeight += int64(blockCount)

		select {

		case <-nodeCtx.Done():
			n.setState(DISCONNECTED)
			<-n.server.conn.Done()
			fmt.Printf("nodeCtx.Done - gathering - %s\n", n.serverAddr)
			<-n.server.conn.Done()
			return nil

		case <-time.After(time.Second):
			hdrsRes, err := n.blockHeaders(nodeCtx, startHeight, blockCount)
			if err != nil {
				return err
			}
			count = hdrsRes.Count

			if count > 0 {
				b, err := hex.DecodeString(hdrsRes.HexConcat)
				if err != nil {
					return err
				}
				nh, err := h.appendHeadersFile(b)
				if err != nil {
					return err
				}
				maybeTip += int64(count)

				fmt.Println(" Appended: ", nh, " headers at ", startHeight, " maybeTip ", maybeTip)
			}

			if count < blockCount {
				// fmt.Println("\nDone gathering")
				doneGathering = true
			}
		}
	}

	// 3. Read up to date blockchain_headers file
	b2, err := h.readAllBytesFromFile()
	if err != nil {
		return err
	}

	// 4. Store all headers in the hdrs map
	err = h.store(b2, startPointHeight)
	if err != nil {
		return err
	}
	h.tip = maybeTip

	// 5. Verify headers in headers map
	fmt.Printf("starting verify at height %d\n", h.tip)
	err = h.verifyAll()
	if err != nil {
		return err
	}
	fmt.Println("header chain verified")

	h.synced = true
	fmt.Println("headers synced up to tip ", h.tip)
	return nil
}

// headersNotify subscribes to new block tip notifications from the
// electrumx server and queues them as they arrive.
//
// ElectrumX Documentation Note:
//   - should a new block arrive quickly, perhaps while the server is still processing
//     prior blocks, the server may only notify of the most recent chain tip. The
//     protocol does not guarantee notification of all intermediate block headers.
func (n *Node) headersNotify(nodeCtx context.Context) error {
	h := n.networkHeaders
	// get a channel to receive notifications from this node's <- server connection
	hdrsNotifyChan := n.getHeadersNotify()
	if hdrsNotifyChan == nil {
		return errors.New("server headers notify channel is nil")
	}
	// start headers queue with depth 3
	qchan := make(chan *headersNotifyResult, 3)
	go n.headerQueue(nodeCtx, qchan)
	// subscribe headers also returns latest header the server knows
	hdrRes, err := n.subscribeHeaders(nodeCtx)
	if err != nil {
		return err
	}
	ourTip := h.getTip()
	fmt.Println("subscribe headers - height", hdrRes.Height, "our tip", ourTip, "diff", hdrRes.Height-ourTip)
	// qchan <- hdrRes

	go func(nodeCtx context.Context) {
		defer close(qchan)
		fmt.Println("=== Waiting for headers ===")
		for {
			if nodeCtx.Err() != nil {
				n.setState(DISCONNECTED)
				<-n.server.conn.done
				fmt.Printf("nodeCtx.Done - in headers notify %s - exiting thread\n", n.serverAddr)
				return
			}

			// from server into the queue
			hdrs := <-hdrsNotifyChan
			qchan <- hdrs
		}
	}(nodeCtx)

	return nil
}

// headerQueue receives incoming headers notify results from qhan
// - run as a goroutine
// The client local 'blockhain_headers' file is appended and the headers map updated and verified.
func (n *Node) headerQueue(nodeCtx context.Context, qchan <-chan *headersNotifyResult) {
	h := n.networkHeaders
	fmt.Println("headrs queue started")
	for {
		if nodeCtx.Err() != nil {
			n.setState(DISCONNECTED)
			<-n.server.conn.Done()
			fmt.Printf("nodeCtx.Done - in headersQueue %s - exiting thread\n", n.serverAddr)
			return
		}

		// process at leisure? can sleep a bit maybe

		for hdrRes := range qchan {

			if hdrRes == nil {
				fmt.Printf("qchan closed %s - exiting thread\n", n.serverAddr)
				return
			}

			ourTip := h.getTip() // locked .. propagate this value to syncHeadersOntoOurTip

			// dbg: TODO: remove --------------------------------------------->
			fmt.Printf(" --- headerQueue %v\n", time.Now())
			fmt.Printf("qlen %d\n", len(qchan))
			fmt.Printf("our tip is %d\n", ourTip)
			tipHdr := n.networkHeaders.hdrs[ourTip]
			var ourTipHash string
			if tipHdr == nil {
				fmt.Println("our header at tip is nil")
				ourTipHash = "<nil>"
			} else {
				ourTipHash = dbgHashHexFromHdr(tipHdr)
				fmt.Printf("our tip hash: %s\n", ourTipHash)
			}
			fmt.Printf("incoming header notification height: %d hex: %s\n", hdrRes.Height, hdrRes.Hex)
			// enddbg: TODO: remove <------------------------------------------

			if hdrRes.Height < h.startPoint {
				// earlier than our starting checkpoint
				// TODO: this condition should just trigger node/server change for MultiNode
				fmt.Printf(" - server height %d is below our starting check point %d\n", hdrRes.Height, h.startPoint)
				continue
			}
			if hdrRes.Height <= ourTip {
				// we already have it
				//
				// Note: maybe some verification here that it is the same as our stored one TODO:
				fmt.Printf(" - we already have a header for height %d\n", hdrRes.Height)
				continue
			}
			fmt.Printf(" - we do not yet have header for height %d, our tip is %d\n", hdrRes.Height, ourTip)
			if hdrRes.Height == (ourTip + 1) {
				// simple case one header incoming .. try connect on top of our chain
				if !n.connectTip(hdrRes.Hex) {
					fmt.Printf(" - possible reorg at server height %d - skipping for now - PLACE AAA\n", hdrRes.Height)
					n.session.bumpCostError()
					continue
				}
				n.session.bumpCostString(hdrRes.Hex)
				// connected the block & updated our headers tip
				fmt.Printf(" - updated 1 header - our new tip is %d\n", h.getTip())
				// notify client
				fmt.Println("   ..updating client thru network's static clientTipChangeNotify channel")
				n.clientTipChangeNotify <- h.getTip()
				continue
			}
			// two or more headers that we do not have yet
			numHdrs, err := n.syncHeadersOntoOurTip(nodeCtx, hdrRes.Height)
			if err != nil {
				panic(err)
			}
			// updating less hdrs than requested is not an error - we hope to get them next time
			fmt.Printf(" - updated %d headers - our new tip is %d\n", numHdrs, h.getTip())
			if numHdrs > 0 {
				fmt.Println("   ..updating client thru network's clientTipChangeNotify channel")
				n.clientTipChangeNotify <- h.getTip()
			}
		}
	}
}

func (n *Node) syncHeadersOntoOurTip(nodeCtx context.Context, serverHeight int64) (int64, error) {
	h := n.networkHeaders
	ourTip := h.getTip()
	numMissing := serverHeight - ourTip
	from := ourTip + 1
	to := serverHeight
	fmt.Printf("syncHeadersFromTip: ourTip %d server height %d num missing %d\n", ourTip, serverHeight, numMissing)
	// per electrum, but I don't think it matters and we could use BlockHeaders once
	if numMissing > 10 {
		return n.updateFromChunk(nodeCtx, from, to)
	}
	return n.updateFromBlocks(nodeCtx, from, to)
}

func (n *Node) updateFromBlocks(nodeCtx context.Context, from, to int64) (int64, error) {
	var headersConnected int64
	fmt.Printf("updateFromBlocks: from %d to %d inclusive\n", from, to)
	for i := from; i <= to; i++ {
		header, err := n.blockHeader(nodeCtx, i)
		if err != nil {
			break
		}

		// dbg
		fmt.Println("incoming:")
		hash := n.dbgHashHexFromHeaderHex(header)
		prev := n.dbgStringHeaderPrev(header)
		fmt.Printf("hdr hash: %s hdr prev: %s\n", hash, prev)
		// enddbg

		if !n.connectTip(header) {

			// debug------------------>
			if i == 0 {
				fmt.Println("dbgTryRecoverHack")
				n.dbgTryRecoverHack()
				fmt.Printf("*** removed 10 stored headers from tip -  new tip is %d - PLACE BBB\n", n.networkHeaders.getTip())
			}
			// <-----------------------

			break
		}
		headersConnected++
		time.Sleep(100 * time.Millisecond) // for reducing session cost at electrumX server
	}
	return headersConnected, nil
}

func (n *Node) updateFromChunk(nodeCtx context.Context, from, to int64) (int64, error) {
	h := n.networkHeaders
	var headersConnected int64
	fmt.Printf("updateFromChunk: from %d to %d inclusive\n", from, to)
	reqCount := int(to - from + 1)
	hdrsRes, err := n.blockHeaders(nodeCtx, from, reqCount)
	if err != nil {
		fmt.Printf("BlockHeaders failed - %v\n", err)
		return 0, nil
	}
	// check max send parameter for validity
	if hdrsRes.Max < 2016 {
		// corrupted or electrumx server code different from expected
		fmt.Printf("server uses too low 'max' count %d for block.headers\n", hdrsRes.Max)
		return 0, nil
	}
	oneHdrLen := int(h.headerSize) * 2
	allHdrs := hdrsRes.HexConcat
	strLenAll := len(allHdrs)
	// check size of returned concatenated blocks
	if strLenAll != oneHdrLen*hdrsRes.Count {
		// corrupted
		fmt.Printf("inconsistent chunk hex and count")
		return 0, nil
	}
	// check the number we asked for has been returned
	if reqCount != hdrsRes.Count {
		// maybe corrupt - maybe deliberate - maybe connect here anyway .. see how often it happens!
		fmt.Printf("expected %d headers but got %d\n", reqCount, hdrsRes.Count)
		return 0, nil
	}

	// dbg
	fmt.Println("incoming:")
	for i := 0; i < hdrsRes.Count; i++ {
		header := hdrsRes.HexConcat[i*oneHdrLen : (i+1)*oneHdrLen]
		hash := n.dbgHashHexFromHeaderHex(header)
		prev := n.dbgStringHeaderPrev(header)
		fmt.Printf("hdr hash: %s hdr prev: %s\n", hash, prev)
	}
	// enddbg

	for i := 0; i < hdrsRes.Count; i++ {
		header := hdrsRes.HexConcat[i*oneHdrLen : (i+1)*oneHdrLen]
		if !n.connectTip(header) {

			// debug------------------>
			if i == 0 {
				fmt.Println("dbgTryRecoverHack")
				n.dbgTryRecoverHack()
				fmt.Printf("*** removed 10 stored headers from tip -  new tip is %d - PLACE CCC\n", n.networkHeaders.getTip())
			}
			// <-----------------------

			break
		}
		headersConnected++
	}
	return headersConnected, nil
}

func (n *Node) connectTip(serverHeader string) bool {
	h := n.networkHeaders
	incomingHdr, incomingHdrBytes, err := n.convertStringHdrToBlkHdr(serverHeader)
	if err != nil {
		// this assertion should maybe just trigger a server change for MultiNode
		panic(err)
	}
	// check connect block
	if !h.checkCanConnect(incomingHdr) {
		fmt.Printf("connectTip - cannot connect\n -- incoming prev hash: %s\n -- current tip hash:   %s \n",
			incomingHdr.PrevBlock.String(), h.getTipBlock().BlockHash().String())
		// fork maybe?
		h.dbgDumpTipHashes(10)
		return false
	}
	// hard to make writing to file && writing headers map atomic. Consider just using the
	// 'blockchain_headers' file for storage.
	_, err = h.appendHeadersFile(incomingHdrBytes)
	if err != nil {
		panic(err)
	}
	// if here we could write a file: 'last_good_header'
	// but it is the top of the chain we following only
	// which could be a fork :(
	h.storeOneHdr(incomingHdr) // updates tip
	return true
}

func (n *Node) convertStringHdrToBlkHdr(svrHdr string) (*wire.BlockHeader, []byte, error) {
	h := n.networkHeaders
	rawBytes, err := hex.DecodeString(svrHdr)
	if err != nil {
		return nil, nil, err
	}
	if len(rawBytes) != h.headerSize {
		return nil, nil, fmt.Errorf("corrupted header - length %d", len(svrHdr))
	}
	r := bytes.NewBuffer(rawBytes)
	hdr := &wire.BlockHeader{}
	err = hdr.Deserialize(r)
	if err != nil {
		return nil, nil, err
	}
	return hdr, rawBytes, nil
}

// debug ------------------------------------------------------------------------

// -------- hack -------------------

func (n *Node) dbgTryRecoverHack() {
	h := n.networkHeaders
	tip := h.getTip()
	if tip > 10 {
		h.truncateHeadersFile(10)
		for i := 0; i < 10; i++ {
			h.removeHdrFromTip()
		}
	}
}

// -------- end hack --------------

func (n *Node) dbgStringHeaderPrev(svrHdr string) string {
	h := n.networkHeaders
	rawBytes, err := hex.DecodeString(svrHdr)
	if err != nil {
		return "<hex decode error>"
	}
	if len(rawBytes) != int(h.headerSize) {
		return "<corrupted header>"
	}
	r := bytes.NewBuffer(rawBytes)
	hdr := &wire.BlockHeader{}
	err = hdr.BtcDecode(r, 0, wire.BaseEncoding)
	if err != nil {
		return "<deserilaize error>"
	}
	return hdr.PrevBlock.String()
}

func (n *Node) dbgHashHexFromHeaderHex(svrHdr string) string {
	hdr, _, err := n.convertStringHdrToBlkHdr(svrHdr)
	if err != nil {
		return "*conversion error*"
	}
	hash := hdr.BlockHash()
	return hash.String()
}

// func dbgPrevHexFromHeaderHex(svrHdr string) string {
// 	hdr, _, err := convertStringHdrToBlkHdr(svrHdr)
// 	if err != nil {
// 		return "*conversion error*"
// 	}
// 	hash := hdr.BlockHash()
// 	return hash.String()
// }

func dbgHashHexFromHdr(hdr *wire.BlockHeader) string {
	return hdr.BlockHash().String()
}
