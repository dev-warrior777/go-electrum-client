package btc

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/dev-warrior777/go-electrum-client/electrumx"
)

// syncHeaders uodates the client headers and then subscribes for new update
// tip notifications and listens for them
// syncHeaders is part of the ElectrumClient interface inmplementation
func (ec *BtcElectrumClient) syncHeaders(ctx context.Context) error {
	err := ec.syncClientHeaders(ctx)
	if err != nil {
		return err
	}
	return ec.headersNotify(ctx)
}

// syncClientHeaders reads blockchain_headers file, then gets any missing block
// from end of file to current tip from server. The current set of headers is
// also stored in headers map and the chain verified by checking previous block
// hashes backwards from local Tip.
func (ec *BtcElectrumClient) syncClientHeaders(ctx context.Context) error {
	h := ec.clientHeaders

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

	node := ec.GetNode()

	hdrsRes, err := node.BlockHeaders(ctx, startHeight, blockCount)
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

		case <-ctx.Done():
			fmt.Println("shutdown - gathering")
			return nil

		case <-time.After(time.Second):
			hdrsRes, err := node.BlockHeaders(ctx, startHeight, blockCount)
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

	// 3. Read up to date blockchain_headers file - this can be improved since
	//    we already read most of it but for now: simplicity
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
	ec.updateWalletTip()
	ec.tipChanged()
	return nil
}

// headersNotify subscribes to new block tip notifications from the
// electrumx server and queues them as they arrive. The client local 'blockhain-
// _headers' file is appended and the headers map updated and verified.
//
// ElectrumX Documentation Note:
//   - should a new block arrive quickly, perhaps while the server is still processing
//     prior blocks, the server may only notify of the most recent chain tip. The
//     protocol does not guarantee notification of all intermediate block headers
func (ec *BtcElectrumClient) headersNotify(ctx context.Context) error {
	h := ec.clientHeaders
	node := ec.GetNode()
	if node == nil {
		return ErrNoNode
	}
	// in case of network restart we want to cancel these notification processing
	// threads and restart new ones for a new SingleNode connection
	notifyCtx, cancelHeaders := context.WithCancel(ctx)
	// store cancel func
	ec.cancelHeadersThreads = cancelHeaders
	// get a channel to receive notifications from this electrumx connection
	hdrsNotifyChan, err := node.GetHeadersNotify()
	if err != nil {
		return err
	}
	// start headers queue
	hdrsQueueChan := make(chan *electrumx.HeadersNotifyResult, 1)
	go ec.headerQueue(notifyCtx, hdrsQueueChan)
	// subscribe headers returns latest header the server knows
	hdrRes, err := node.SubscribeHeaders(ctx)
	if err != nil {
		return err
	}
	ourTip := h.getTip()
	fmt.Println("subscribe headers - height", hdrRes.Height, "our tip", ourTip, "diff", hdrRes.Height-ourTip)
	hdrsQueueChan <- hdrRes

	go func() {
		fmt.Println("=== Waiting for headers ===")
		for {
			select {
			case <-notifyCtx.Done():
				fmt.Println("notifyCtx.Done - in headers notify - exiting thread")
				close(hdrsQueueChan)
				return
			case hdrNotifyRes, ok := <-hdrsNotifyChan:
				if !ok {
					fmt.Println("headers notify channel was closed - exiting thread")
					return
				}
				hdrsQueueChan <- hdrNotifyRes
			}
		}
	}()

	return nil
}

func (ec *BtcElectrumClient) headerQueue(notifyCtx context.Context, hdrsQueueChan <-chan *electrumx.HeadersNotifyResult) {
	h := ec.clientHeaders
	fmt.Println("headrs queue started")
	for {
		select {
		case <-notifyCtx.Done():
			fmt.Println("notifyCtx.Done - in headers queue - exiting thread")
			return
		case hdrRes, ok := <-hdrsQueueChan:
			if !ok {
				fmt.Println("headers queue channel was closed - exiting thread")
				return
			}
			// process at leisure? can sleep
			fmt.Printf("incoming header notification height: %d hex: %s\n", hdrRes.Height, hdrRes.Hex)
			ourTip := h.getTip()
			if hdrRes.Height < h.startPoint {
				// earlier than our starting checkpoint
				// this condition would just trigger server change for MultiNode. SingleNode is 'trusted' ..maybe ;-)
				fmt.Printf(" - server height %d is below our starting check point %d\n", hdrRes.Height, h.startPoint)
				continue
			}
			if hdrRes.Height <= ourTip {
				// we have it - usually this is the the special case of the very first
				// hdr passedin from 'blockchain.subscribe.headers' - as we just did a
				// headers sync.
				//
				// Note: maybe some verification here that it is the same as our stored one TODO:
				fmt.Printf(" - we already have a header for height %d\n", hdrRes.Height)
				continue
			}
			fmt.Printf(" - we do not yet have header for height %d, our tip is %d\n", hdrRes.Height, ourTip)
			if hdrRes.Height == (ourTip + 1) {
				// simple case one header incoming .. try connect on top of our chain
				if !ec.connectTip(hdrRes.Hex) {
					fmt.Printf(" - possible reorg at server height %d - ignoring for now - TODO:\n", hdrRes.Height)
				}
				// connected the block & updated our headers tip
				fmt.Printf(" - updated 1 header - our new tip is %d\n", h.getTip())
				continue
			}
			// two or more headers that we do not have yet
			n, err := ec.syncHeadersOntoOurTip(notifyCtx, hdrRes.Height)
			if err != nil {
				panic(err)
			}
			// updating less hdrs than requested is not an error - we hope to get them next time
			fmt.Printf(" - updated %d headers - our new tip is %d\n", n, h.getTip())
		}
	}
}

func (ec *BtcElectrumClient) syncHeadersOntoOurTip(notifyCtx context.Context, serverHeight int64) (int64, error) {
	h := ec.clientHeaders
	ourTip := h.getTip()
	numMissing := serverHeight - ourTip
	from := ourTip + 1
	to := serverHeight
	fmt.Printf("syncHeadersFromTip: ourTip %d server height %d\n", ourTip, serverHeight)
	if numMissing > 10 { // per electrum
		return ec.updateFromChunk(notifyCtx, from, to)
	}
	return ec.updateFromBlocks(notifyCtx, from, to)
}

func (ec *BtcElectrumClient) updateFromBlocks(notifyCtx context.Context, from, to int64) (int64, error) {
	node := ec.GetNode()
	if node == nil {
		return 0, ErrNoNode
	}
	var headersConnected int64
	fmt.Printf("updateFromBlocks: from %d to %d inclusive\n", from, to)
	for i := from; i <= to; i++ {
		header, err := node.BlockHeader(notifyCtx, i)
		if err != nil {
			break
		}
		if !ec.connectTip(header) {
			break
		}
		headersConnected++
		time.Sleep(time.Second) // for now .. find out how to calculate this
	}
	return headersConnected, nil
}

func (ec *BtcElectrumClient) updateFromChunk(notifyCtx context.Context, from, to int64) (int64, error) {
	node := ec.GetNode()
	if node == nil {
		return 0, ErrNoNode
	}
	var headersConnected int64
	fmt.Printf("updateFromChunk: from %d to %d inclusive\n", from, to)
	reqCount := int(to - from + 1)
	hdrsRes, err := node.BlockHeaders(notifyCtx, from, reqCount)
	if err != nil {
		fmt.Printf("BlockHeaders failed - %v\n", err)
		return 0, nil
	}
	if hdrsRes.Max < 2016 {
		// corrupted
		fmt.Printf("server uses too low 'max' count %d for block.headers\n", hdrsRes.Max)
		return 0, nil
	}
	OneHdrLen := HEADER_SIZE * 2
	allHdrs := hdrsRes.HexConcat
	strLenAll := len(allHdrs)
	if strLenAll != OneHdrLen*hdrsRes.Count {
		// corrupted
		fmt.Printf("inconsistent chunk hex and count")
		return 0, nil
	}
	if reqCount != hdrsRes.Count {
		// maybe corrupt - maybe deliberate - maybe connect here, anyway .. see how often it happens
		fmt.Printf("expected %d headers but only got %d\n", reqCount, hdrsRes.Count)
		return 0, nil
	}
	for i := 0; i < hdrsRes.Count; i++ {
		header := hdrsRes.HexConcat[i*OneHdrLen : (i+1)*OneHdrLen]
		if !ec.connectTip(header) {
			break
		}
		headersConnected++
	}
	return headersConnected, nil
}

func (ec *BtcElectrumClient) connectTip(serverHeader string) bool {
	h := ec.clientHeaders
	incomingHdr, incomingHdrBytes, err := convertStringHdrToBlkHdr(serverHeader)
	if err != nil {
		// this assertion would just trigger server change for MultiNode
		panic(err)
	}
	// check connect block
	if !h.checkCanConnect(incomingHdr) {
		// fork maybe?
		return false
	}
	_, err = h.appendHeadersFile(incomingHdrBytes)
	if err != nil {
		panic(err)
	}
	h.storeOneHdr(incomingHdr)
	return true
}

func convertStringHdrToBlkHdr(svrHdr string) (*wire.BlockHeader, []byte, error) {
	rawBytes, err := hex.DecodeString(svrHdr)
	if err != nil {
		return nil, nil, err
	}
	if len(rawBytes) != HEADER_SIZE {
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

// // preValidateheaders validates incoming headers before storing in file and map
// func (ec *BtcElectrumClient) preValidateHeaders(gbr *electrumx.GetBlockHeadersResult, tip int64) error {
// 	h := ec.clientHeaders
// 	rawBytes, err := hex.DecodeString(gbr.HexConcat)
// 	if err != nil {
// 		return err
// 	}
// 	// validate incoming bytes as multiple of HEADER_SIZE
// 	rawBytesLen := int64(len(rawBytes))
// 	n, err := h.bytesToNumHdrs(rawBytesLen)
// 	if err != nil {
// 		return err
// 	}
// 	count := int64(gbr.Count)
// 	if n != count {
// 		return fmt.Errorf("bad GetBlockHeadersResult")
// 	}
// 	// convert to BlockHeader
// 	incoming := make([]*wire.BlockHeader, 0)
// 	var i int64
// 	for i = 0; i < count; i++ {
// 		b := rawBytes[i*HEADER_SIZE : (i+1)*HEADER_SIZE]
// 		r := bytes.NewBuffer(b)
// 		blkHdr := &wire.BlockHeader{}
// 		err = blkHdr.Deserialize(r)
// 		if err != nil {
// 			return err
// 		}
// 		incoming = append(incoming, blkHdr)
// 	}
// 	// verify backwards except the first incoming header
// 	for i := count - 1; i > 0; i-- {
// 		prev := incoming[i].PrevBlock
// 		hash := incoming[i-1].BlockHash()
// 		if prev != hash {
// 			return err
// 		}
// 	}
// 	// check no missing by verifying first incoming prev against last stored
// 	// block hash.
// 	h.hdrsMtx.RLock()
// 	lastHdr := h.hdrs[tip]
// 	h.hdrsMtx.RUnlock()
// 	lastHdrHash := lastHdr.BlockHash()
// 	if incoming[0].PrevBlock != lastHdrHash {
// 		return err
// 	}
// 	return nil
// }

///////////////////////////
// ElectrumClient interface

// Tip returns the (local) block headers tip height and client headers sync status.
func (ec *BtcElectrumClient) Tip() (int64, bool) {
	h := ec.clientHeaders
	h.hdrsMtx.RLock()
	defer h.hdrsMtx.RUnlock()
	return h.tip, h.synced
}

// GetBlockHeader returns the client's block header for height. If out of range
// will return nil.
func (ec *BtcElectrumClient) GetBlockHeader(height int64) *wire.BlockHeader {
	h := ec.clientHeaders
	h.hdrsMtx.RLock()
	defer h.hdrsMtx.RUnlock()
	// return nil for now. If there is a need for blocks before the last checkpoint
	// consider making a server call
	return h.hdrs[height]
}

// GetBlockHeaders returns the client's block headers for the requested range.
// If startHeight < startPoint or startHeight > tip or startHeight+count > tip
// will return error.
func (ec *BtcElectrumClient) GetBlockHeaders(startHeight, count int64) ([]*wire.BlockHeader, error) {
	h := ec.clientHeaders
	h.hdrsMtx.RLock()
	defer h.hdrsMtx.RUnlock()
	if h.startPoint > startHeight {
		// error for now. If there is a need for blocks before the last checkpoint
		// consider making a server call
		return nil, errors.New("requested start height < start of stored blocks")
	}
	if startHeight > h.tip {
		return nil, errors.New("requested start height > tip")
	}
	blkEndRange := startHeight + count
	if blkEndRange > h.tip {
		return nil, errors.New("requested range exceeds the tip")
	}
	var headers = make([]*wire.BlockHeader, 0, 3)
	for i := startHeight; i < blkEndRange; i++ {
		headers = append(headers, h.hdrs[i])
	}
	return headers, nil
}

func (ec *BtcElectrumClient) GetHeaderForBlockHash(blkHash *chainhash.Hash) *wire.BlockHeader {
	h := ec.clientHeaders
	h.hdrsMtx.RLock()
	defer h.hdrsMtx.RUnlock()
	height := h.blkHdrs[*blkHash]
	return ec.GetBlockHeader(height)
}

func (ec *BtcElectrumClient) RegisterTipChangeNotify() (<-chan int64, error) {
	h := ec.clientHeaders
	h.tipChangeMtx.Lock()
	defer h.tipChangeMtx.Unlock()
	if !h.synced {
		return nil, errors.New("client's header chain is not synced")
	}
	h.tipChange = make(chan int64, 1)
	return h.tipChange, nil
}

func (ec *BtcElectrumClient) UnregisterTipChangeNotify() {
	h := ec.clientHeaders
	h.tipChangeMtx.Lock()
	defer h.tipChangeMtx.Unlock()
	if h.tipChange != nil {
		close(ec.clientHeaders.tipChange)
		ec.clientHeaders.tipChange = nil
	}
}

func (ec *BtcElectrumClient) tipChanged() {
	h := ec.clientHeaders
	h.tipChangeMtx.Lock()
	defer h.tipChangeMtx.Unlock()
	if h.tipChange != nil {
		h.tipChange <- h.tip
	}
}
