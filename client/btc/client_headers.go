package btc

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
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

	b, err := h.ReadAllBytesFromFile()
	if err != nil {
		return err
	}
	lb := int64(len(b))
	fmt.Println("read:", lb, " bytes from header file")
	numHeaders, err := h.BytesToNumHdrs(lb)
	if err != nil {
		return err
	}
	b = nil // gc

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
		nh, err := h.AppendHeaders(b)
		if err != nil {
			log.Fatal(err)
		}
		maybeTip += int64(count)

		fmt.Println(" Appended: ", nh, " headers at ", startHeight, " maybeTip ", maybeTip)
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

		case <-time.After(time.Millisecond * 1000):
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
				nh, err := h.AppendHeaders(b)
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
	b2, err := h.ReadAllBytesFromFile()
	if err != nil {
		return err
	}

	// 4. Store all headers in a map
	err = h.Store(b2, startPointHeight)
	if err != nil {
		return err
	}
	h.tip = maybeTip

	// 5. Verify headers in headers map
	fmt.Printf("starting verify at height %d\n", h.tip)
	err = h.VerifyAll()
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
// electrumx server and handles them as they arrive. The client local 'blockhain
// _headers' file is appended and the headers map updated and verified.
//
// Note:
// should a new block arrive quickly, perhaps while the server is still processing
// prior blocks, the server may only notify of the most recent chain tip. The
// protocol does not guarantee notification of all intermediate block headers.
//
// headersNotify is part of the ElectrumClient interface implementation
func (ec *BtcElectrumClient) headersNotify(ctx context.Context) error {
	h := ec.clientHeaders

	// local tip for calculation before storage
	maybeTip := h.tip

	node := ec.GetNode()

	hdrResNotifyCh, err := node.GetHeadersNotify()
	if err != nil {
		return err
	}

	hdrRes, err := node.SubscribeHeaders(ctx)
	if err != nil {
		return err
	}

	fmt.Println("Subscribe Headers")
	fmt.Println(" - height", hdrRes.Height, "maybeTip", maybeTip, "diff", hdrRes.Height-maybeTip)

	// in case of network restart we want to cancel this thread and restart a new one

	// make new cancellable context for network restarts
	notifyCtx, cancelHeaders := context.WithCancel(ctx)
	// store in ec
	ec.cancelHeadersNotify = cancelHeaders

	go func() {

		fmt.Println("=== Waiting for headers ===")

		for {
			select {

			case <-notifyCtx.Done():
				fmt.Println("notifyCtx.Done - in headers notify - exiting thread")
				return

			case _, ok := <-hdrResNotifyCh:
				if !ok {
					fmt.Println("headers notify channel closed - exiting thread")
					return
				}

				// read whatever is in the queue, usually one header at tip
				for x := range hdrResNotifyCh {
					fmt.Printf("new block: height %d %s\n", x.Height, x.Hex)
					if x.Height > maybeTip {
						n := x.Height - maybeTip
						if n == 1 {
							// simple case: just store it
							fmt.Println("Storing header for height: ", x.Height)
							b, err := hex.DecodeString(x.Hex)
							if err != nil {
								panic(err)
							}
							hdrsAppended, err := h.AppendHeaders(b)
							if err != nil {
								panic(err)
							}
							if hdrsAppended != 1 {
								panic("appended less headers than read")
							}
							err = h.Store(b, x.Height)
							if err != nil {
								panic("could not store header in map")
							}

							// update tip / local tip / wallet tip /  notify listener
							h.tip = x.Height
							maybeTip = x.Height
							ec.updateWalletTip()
							ec.tipChanged()

							// verify added header back from new tip
							h.VerifyFromTip(2, false)

						} else {
							// Server can skip any amount of headers but we should
							// trust that this SingleNode's tip is the tip ..maybe
							fmt.Println("More than one header..")
							numMissing := x.Height - maybeTip
							from := maybeTip + 1
							numToGet := int(numMissing)
							fmt.Printf("Filling from height %d to height %d inclusive\n", from, x.Height)
							// go get them with 'block.headers'
							hdrsRes, err := node.BlockHeaders(ctx, from, numToGet)
							if err != nil {
								panic(err)
							}
							count := hdrsRes.Count

							fmt.Println("Storing: ", count, " headers ", from, "..", from+int64(count)-1)

							if count > 0 {
								b, err := hex.DecodeString(hdrsRes.HexConcat)
								if err != nil {
									panic(err)
								}
								hdrsAppended, err := h.AppendHeaders(b)
								if err != nil {
									panic(err)
								}
								if hdrsAppended != int64(count) {
									panic("only appended less headers than read")
								}
								err = h.Store(b, from)
								if err != nil {
									panic("could not store headers in map")
								}

								// update tip / local tip / wallet tip / notify listener
								h.tip = x.Height
								maybeTip = x.Height
								ec.updateWalletTip()
								ec.tipChanged()

								// verify added headers back from new tip
								h.VerifyFromTip(int64(count+1), false)
							}
						}
					} else {
						fmt.Printf("Already got a header for height %d - possible reorg (unhandled)\n", x.Height)
					}
				}
			}
		}
		// serve until ^C
	}()

	return nil
}

// ElectrumClient interface

// Tip returns the (local) block headers tip height and client headers sync status.
func (ec *BtcElectrumClient) Tip() (int64, bool) {
	h := ec.clientHeaders
	return h.tip, h.synced
}

// GetBlockHeader returns the client's block header for height. If out of range
// will return nil.
func (ec *BtcElectrumClient) GetBlockHeader(height int64) *wire.BlockHeader {
	h := ec.clientHeaders
	h.hdrsMtx.Lock()
	defer h.hdrsMtx.Unlock()
	// return nil for now. If there is a need for blocks before the last checkpoint
	// consider making a server call
	return h.hdrs[height]
}

// GetBlockHeaders returns the client's block headers for the requested range.
// If startHeight < startPoint or startHeight > tip or startHeight+count > tip
// will return error.
func (ec *BtcElectrumClient) GetBlockHeaders(startHeight, count int64) ([]*wire.BlockHeader, error) {
	h := ec.clientHeaders
	h.hdrsMtx.Lock()
	defer h.hdrsMtx.Unlock()
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
	height := h.blkHdrs[*blkHash]
	return ec.GetBlockHeader(height)
}

func (ec *BtcElectrumClient) RegisterTipChangeNotify() (<-chan int64, error) {
	h := ec.clientHeaders
	if !h.synced {
		return nil, errors.New("client's header chain is not synced")
	}
	h.tipChange = make(chan int64, 1)
	return h.tipChange, nil
}

func (ec *BtcElectrumClient) UnregisterTipChangeNotify() {
	h := ec.clientHeaders
	if h.tipChange != nil {
		close(ec.clientHeaders.tipChange)
		ec.clientHeaders.tipChange = nil
	}
}

func (ec *BtcElectrumClient) tipChanged() {
	h := ec.clientHeaders
	if h.tipChange != nil {
		h.tipChange <- h.tip
	}
}
