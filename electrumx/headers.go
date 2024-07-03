package electrumx

// This is the Client's copy of the blockchain headers for a blockchain.
// Backed by a file in the datadir of the chain (main, test, reg nets)
// We store here as a map and not a tree so we must trust the server if
// SingleNode. When grabbing new blocks some attempt is made to understand
// reorgs but the true longest chain with the most work cannot be known
// without connecting to many servers using MultiNode which is a TODO:

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

const (
	HEADER_SIZE           = 80 // TODO: move to ElectrumXInterface
	HEADER_FILE_NAME      = "blockchain_headers"
	ELECTRUM_MAGIC_NUMHDR = 2016 // chunk size
)

type Headers struct {
	// blockchain headers file to persist headers we know
	hdrFilePath string
	// chain parameters for genesis and checkpoint. We always use the latest
	// checkpoint height to start the file. For regtest that is genesis.
	net *chaincfg.Params
	// decoded headers stored by height
	hdrsMtx    sync.RWMutex
	hdrs       map[int64]*wire.BlockHeader
	blkHdrs    map[chainhash.Hash]int64
	startPoint int64
	tip        int64
	synced     bool
	tipChange  chan int64
}

func NewHeaders(cfg *ElectrumXConfig) *Headers {
	filePath := filepath.Join(cfg.DataDir, HEADER_FILE_NAME)
	hdrsMapInitSize := 2 * ELECTRUM_MAGIC_NUMHDR //4032
	hdrsMap := make(map[int64]*wire.BlockHeader, hdrsMapInitSize)
	bhdrsMap := make(map[chainhash.Hash]int64, hdrsMapInitSize)
	hdrs := Headers{
		hdrFilePath: filePath,
		net:         cfg.Params,
		hdrs:        hdrsMap,
		blkHdrs:     bhdrsMap,
		startPoint:  getStartPointHeight(cfg),
		tip:         0,
		synced:      false,
		tipChange:   nil,
	}
	return &hdrs
}

// stored Headers start from here - TODO: move to ElectrumXInterface
func getStartPointHeight(cfg *ElectrumXConfig) int64 {
	var startAtHeight int64 = 0
	chain := cfg.Chain.String()
	switch chain {
	case "Bitcoin":
		switch cfg.Params {
		case &chaincfg.RegressionNetParams:
			startAtHeight = 0
		case &chaincfg.TestNet3Params:
			startAtHeight = int64(2560000)
		case &chaincfg.MainNetParams:
			startAtHeight = int64(823000)
		}
	default:
		fmt.Printf("headers.go: unimplemented chain: %s\n", chain)
	}
	return startAtHeight
}

// Get the 'blockchain_headers' file size. Error is returned unexamined as
// we assume the file exists and ENOENT will not be valid.
func (h *Headers) statFileSize() (int64, error) {
	fi, err := os.Stat(h.hdrFilePath)
	if err != nil {
		fmt.Println(err.Error())
		return 0, err
	}
	return fi.Size(), nil
}

// // Read num headers from offset in 'blockchain_headers' file
// func (h *Headers) readHeaders(num, height int64) (int32, error) {
// 	begin := int64(height * HEADER_SIZE)
// 	f, err := os.OpenFile(h.hdrFilePath, os.O_CREATE|os.O_RDWR, 0664)
// 	if err != nil {
// 		return 0, err
// 	}
// 	f.Seek(begin, 0)
// 	bytesToRead := num * HEADER_SIZE
// 	b := make([]byte, bytesToRead)
// 	bytesRead, err := f.Read(b)
// 	if err != nil {
// 		return 0, err
// 	}
// 	if bytesRead == 0 { // empty
// 		return 0, nil
// 	}
// 	if bytesRead < int(bytesToRead) {
// 		if bytesRead%HEADER_SIZE != 0 {
// 			return 0, errors.New("corrupt file")
// 		}
// 		headersRead := int32(bytesRead / HEADER_SIZE)
// 		err = h.store(b, height)
// 		return headersRead, err
// 	}
// 	return 0, nil
// }

// appendHeadersFile appends headers from server 'blockchain.block.header(s)' calls
// to 'blockchain_headers' file. Also appends headers received from the
// 'blockchain.headers.subscribe' events. Returns the number of headers written.
func (h *Headers) appendHeadersFile(rawHdrs []byte) (int64, error) {
	numBytes := len(rawHdrs)
	numHdrs, err := h.bytesToNumHdrs(int64(numBytes))
	if err != nil {
		return 0, err
	}
	hdrFile, err := os.OpenFile(h.hdrFilePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0664)
	if err != nil {
		return 0, err
	}
	defer hdrFile.Close()
	_, err = hdrFile.Write(rawHdrs)
	if err != nil {
		return 0, err
	}
	return numHdrs, nil
}

func (h *Headers) truncateHeadersFile(numHeaders int64) (int64, error) {
	if !h.synced {
		return 0, errors.New("still syncing")
	}
	if numHeaders <= 0 {
		return 0, errors.New("numHeaders <= 0")
	}
	size, err := h.statFileSize()
	if err != nil {
		return 0, err
	}
	if size%HEADER_SIZE != 0 {
		return 0, errors.New("headers file corrupt - size not a multiple of HEADER_SIZE")
	}
	newByteLen := size - (numHeaders * HEADER_SIZE)
	if newByteLen < 0 {
		newByteLen = 0
	}
	err = os.Truncate(h.hdrFilePath, newByteLen)
	if err != nil {
		return 0, err
	}
	newSize, err := h.statFileSize()
	if err != nil {
		return 0, err
	}
	newNumHeaders, _ := h.bytesToNumHdrs(newSize)
	return newNumHeaders, nil
}

func (h *Headers) readAllBytesFromFile() ([]byte, error) {
	hdrFile, err := os.OpenFile(h.hdrFilePath, os.O_CREATE|os.O_RDWR, 0664)
	if err != nil {
		return nil, err
	}
	defer hdrFile.Close()
	fsize, err := h.statFileSize()
	if err != nil {
		return nil, err
	}
	b := make([]byte, fsize)
	hdrFile.Seek(0, 0)
	n, err := hdrFile.Read(b)
	if err != nil {
		return nil, err
	}
	if n != int(fsize) {
		return nil, errors.New("read less tha file size")
	}
	return b, nil
}

// store 'numHdrs' headers starting at height 'height' in 'hdrs' map
// 'b' should have exactly 'numHdrs' x 'HEADER_SIZE' bytes.
// updates h.Tip for successful additions
func (h *Headers) store(b []byte, startHeight int64) error {
	numHdrs, err := h.bytesToNumHdrs(int64(len(b)))
	if err != nil {
		return err
	}
	rdr := bytes.NewBuffer(b)
	h.hdrsMtx.Lock()
	defer h.hdrsMtx.Unlock()
	var i int64
	for i = 0; i < numHdrs; i++ {
		blkHdr := &wire.BlockHeader{}
		err := blkHdr.Deserialize(rdr)
		if err != nil {
			return err
		}
		at := startHeight + i
		h.hdrs[at] = blkHdr
		blkHash := blkHdr.BlockHash()
		h.blkHdrs[blkHash] = at
	}
	return nil
}

func (h *Headers) removeHdrFromTip() {
	h.hdrsMtx.Lock()
	defer h.hdrsMtx.Unlock()
	delete(h.hdrs, h.tip)
	h.tip--
}

// tip returns the stored block headers tip height.
func (h *Headers) getTip() int64 {
	h.hdrsMtx.RLock()
	defer h.hdrsMtx.RUnlock()
	return h.tip
}

// GetBlockHeader returns the block header for height. If out of range will
// return nil.
func (h *Headers) getBlockHeader(height int64) (*wire.BlockHeader, error) {
	h.hdrsMtx.RLock()
	defer h.hdrsMtx.RUnlock()
	// If there is a need for blocks before the last checkpoint consider making
	// a server call
	hdr := h.hdrs[height]
	if hdr == nil {
		return nil, fmt.Errorf("no block stored for height %d", height)
	}
	return hdr, nil
}

// getBlockHeaders returns the stored block headers for the requested range.
// If startHeight < startPoint or startHeight > tip or startHeight+count > tip
// will return error.
func (h *Headers) getBlockHeaders(startHeight, count int64) ([]*wire.BlockHeader, error) {
	h.hdrsMtx.RLock()
	defer h.hdrsMtx.RUnlock()
	if h.startPoint > startHeight {
		// error for now. If there is a need for blocks before the last checkpoint
		// consider making a server call if api users need that.
		return nil, errors.New("requested start height < start of stored blocks")
	}
	if startHeight > h.tip {
		return nil, errors.New("requested start height > local tip")
	}
	blkEndRange := startHeight + count
	if blkEndRange > h.tip {
		return nil, errors.New("requested range is past the local tip")
	}
	var headers = make([]*wire.BlockHeader, 0, 3)
	for i := startHeight; i < blkEndRange; i++ {
		headers = append(headers, h.hdrs[i])
	}
	return headers, nil
}

// func (n *Node) getHeaderForBlockHash(blkHash *chainhash.Hash) *wire.BlockHeader {
// 	h := n.networkHeaders
// 	h.hdrsMtx.RLock()
// 	defer h.hdrsMtx.RUnlock()
// 	height := h.blkHdrs[*blkHash]
// 	return h.hdrs[height]
// }

func (h *Headers) getTipBlock() *wire.BlockHeader {
	h.hdrsMtx.RLock()
	defer h.hdrsMtx.RUnlock()
	return h.hdrs[h.tip]
}

func (h *Headers) checkCanConnect(incomingHdr *wire.BlockHeader) bool {
	ourTipHeader := h.getTipBlock()
	ourHash := ourTipHeader.BlockHash()
	return ourHash == incomingHdr.PrevBlock
}

// storeOneHdr stores one wire block header at h.tip+1 and updates h.tip
// the header is assumed to be valid and can connect
func (h *Headers) storeOneHdr(blkHdr *wire.BlockHeader) {
	h.hdrsMtx.Lock()
	defer h.hdrsMtx.Unlock()
	at := h.tip + 1
	h.hdrs[at] = blkHdr
	blkHash := blkHdr.BlockHash()
	h.blkHdrs[blkHash] = at
	h.tip = at
}

// Verify headers prev hash back from tip. If 'all' is true 'depth' is ignored
// and the whole chain is verified
func (h *Headers) verifyFromTip(depth int64, all bool) error {
	h.hdrsMtx.RLock()
	defer h.hdrsMtx.RUnlock()
	downTo := h.tip - depth
	if downTo < 0 || all {
		downTo = h.startPoint
	}
	var height int64
	for height = h.tip; height > downTo; height-- {
		thisHdr := h.hdrs[height]
		prevHdr := h.hdrs[height-1]
		prevHdrBlkHash := prevHdr.BlockHash()
		if prevHdrBlkHash != thisHdr.PrevBlock {
			return fmt.Errorf("verify failed: height %d", height)
		}
		// fmt.Printf("verified header at height %d has blockhash %s\n",
		// 	height-1, prevHdrBlkHash.String())
	}
	return nil
}

func (h *Headers) verifyAll() error {
	return h.verifyFromTip(0, true)
}

func (h *Headers) dumpAt(height int64) {
	h.hdrsMtx.Lock()
	defer h.hdrsMtx.Unlock()
	hdr := h.hdrs[height]
	fmt.Println("Hash: ", hdr.BlockHash(), "Height: ", height)
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

func (h *Headers) dumpAll() {
	var k int64
	for k = 0; k <= h.tip; k++ {
		h.dumpAt(k)
	}
}

func (h *Headers) bytesToNumHdrs(numBytes int64) (int64, error) {
	if numBytes%HEADER_SIZE != 0 {
		return 0, errors.New(
			"invalid bytes length - not a multiple of header size")
	}
	return numBytes / HEADER_SIZE, nil
}

// Only for TEST in headers_test.go
func (h *Headers) ClearMaps() {
	h.hdrs = nil
	h.blkHdrs = nil
	h.hdrs = make(map[int64]*wire.BlockHeader, 10)
	h.blkHdrs = make(map[chainhash.Hash]int64, 10)
}

// Only debug - dump the top 'depth' hash - prev hashes
func (h *Headers) dbgDumpTipHashes(depth int64) {
	tip := h.getTip()
	fmt.Printf("--- Dump of the top %d stored headers ---\n", depth)
	for i := tip; i > tip-depth; i-- {
		hash := h.hdrs[i].BlockHash().String()
		prev := h.hdrs[i].PrevBlock.String()
		fmt.Printf("height: %d hash: %s prev: %s\n", i, hash, prev)
	}
}
