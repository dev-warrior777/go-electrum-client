package electrumx

// This is the blockchain headers for a blockchain.

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

const (
	// All coins will use this file-name under ../<coin>/<net>/
	HEADER_FILE_NAME = "blockchain_headers"
	// Bitcoin ElectrumX chunk size - may vary for other coins' ElectrumX servers.
	ELECTRUM_MAGIC_NUMHDR = 2016
)

var ErrSyncing = errors.New("syncing in progress")

func reverseHash(arr [HashSize]byte) [HashSize]byte {
	var newArr [HashSize]byte
	left := 0
	right := HashSize - 1
	for left < right {
		arr[left], arr[right] = arr[right], arr[left]
		left++
		right--
	}
	copy(newArr[:], arr[:])
	return newArr
}

func (wh WireHash) StringRev() string {
	revHash := reverseHash(wh)
	hexStr := hex.EncodeToString(revHash[:])
	return hexStr
}

type headers struct {
	// blockchain header size - per coin.
	headerSize int
	// blockchain_headers file to persist headers we know in datadir
	hdrFilePath string
	// chain parameters for genesis.
	// for each coin/nettype we use the most recent checkpoint (or a specific
	// but arbitrary one) for the start of the block_headers file. For regtest
	// that is block 0.
	startPoint        int64
	headerDeserialzer HeaderDeserializer
	// decoded headers stored by height
	hdrs        map[int64]*BlockHeader
	blkHdrs     map[WireHash]int64
	hdrsMtx     sync.RWMutex
	tip         atomic.Int64
	synced      bool
	recovery    bool
	recoveryTip int64
}

func newHeaders(cfg *ElectrumXConfig) *headers {
	filePath := filepath.Join(cfg.DataDir, HEADER_FILE_NAME)
	startPoint := getStartPointHeight(cfg)
	hdrsMapInitSize := 2 * ELECTRUM_MAGIC_NUMHDR //4032
	hdrsMap := make(map[int64]*BlockHeader, hdrsMapInitSize)
	bhdrsMap := make(map[WireHash]int64, hdrsMapInitSize)
	headerDeserialzer := cfg.HeaderDeserializer
	hdrs := headers{
		headerSize:        cfg.BlockHeaderSize,
		hdrFilePath:       filePath,
		startPoint:        startPoint,
		headerDeserialzer: headerDeserialzer,
		hdrs:              hdrsMap,
		blkHdrs:           bhdrsMap,
		synced:            false,
		recovery:          false,
		recoveryTip:       0,
	}
	return &hdrs
}

func (h *headers) getTip() int64 {
	return h.tip.Load()
}

func (h *headers) setTip(n int64) {
	h.tip.Store(n)
}

func (h *headers) incTip(delta int64) {
	h.tip.Add(delta)
}

func (h *headers) decTip(delta int64) {
	h.tip.Add(-delta)
}

// stored headers start from here
func getStartPointHeight(cfg *ElectrumXConfig) int64 {
	var startAtHeight int64 = 0
	switch cfg.NetType {
	case MAINNET:
		startAtHeight = cfg.StartPoints[MAINNET]
	case TESTNET:
		startAtHeight = cfg.StartPoints[TESTNET]
	case REGTEST:
		startAtHeight = cfg.StartPoints[REGTEST]
	}
	return startAtHeight
}

// ----------------------------------------------------------------------------
// Headers file
// ----------------------------------------------------------------------------

// Get the 'blockchain_headers' file size. Error is returned unexamined as
// we assume the file exists and ENOENT will not be valid.
func (h *headers) statFileSize() (int64, error) {
	fi, err := os.Stat(h.hdrFilePath)
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

// appendHeadersFile appends headers from server 'blockchain.block.header(s)' calls
// to 'blockchain_headers' file. Also appends headers received from the
// 'blockchain.headers.subscribe' events. Returns the number of headers written.
func (h *headers) appendHeadersFile(rawHdrs []byte) (int64, error) {
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

func (h *headers) truncateHeadersFile(numHeaders int64) (int64, error) {
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
	headerSize := int64(h.headerSize)
	if size%headerSize != 0 {
		return 0, errors.New("headers file corrupt - size not a multiple of h.headerSize")
	}
	newByteLen := size - (numHeaders * headerSize)
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

func (h *headers) readAllBytesFromFile() ([]byte, error) {
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

// ----------------------------------------------------------------------------
// Maps
// ----------------------------------------------------------------------------

// Store blockHeaders starting at startHeight in the 'hdrs' map
func (h *headers) store(b []byte, startHeight int64) error {
	numHdrs, err := h.bytesToNumHdrs(int64(len(b)))
	if err != nil {
		return err
	}
	rdr := bytes.NewBuffer(b)
	h.hdrsMtx.Lock()
	defer h.hdrsMtx.Unlock()
	var i int64
	for i = 0; i < numHdrs; i++ {
		blkHdr, err := h.headerDeserialzer.Deserialize(rdr)
		if err != nil {
			return err
		}
		at := startHeight + i
		h.hdrs[at] = blkHdr
		blkHash := blkHdr.Hash
		h.blkHdrs[blkHash] = at
	}
	return nil
}

func (h *headers) removeOneHdrFromTip() {
	h.hdrsMtx.Lock()
	defer h.hdrsMtx.Unlock()
	delete(h.hdrs, h.getTip())
	h.decTip(1)
}

// getClientTip returns the stored block headers last tip height or .
//
// If we are in a reorg recovery our tip has been re-wound back to a previous
// known good (probably) header so send the client back the recoveryTip which
// is the last one sent back before reorg was noticed.
func (h *headers) getClientTip() int64 {
	if h.recovery {
		return h.recoveryTip
	}
	return h.getTip()
}

// getClientSynced returns the headers synced status - not locked
func (h *headers) getClientSynced() bool {
	return h.synced
}

// // reverse lookup
// func (n *Node) getHeaderForBlockHash(blkHash *WireHash) *BlockHeader {
// 	h := n.networkHeaders
// 	height := h.blkHdrs[*blkHash]
// 	return h.hdrs[height]
// }

func (h *headers) getTipHash() WireHash {
	hdr := h.hdrs[h.getTip()]
	return hdr.Hash
}

func (h *headers) checkCanConnect(incomingHdr *BlockHeader) bool {
	ourTipHash := h.getTipHash()
	return ourTipHash == incomingHdr.Prev
}

// storeOneHdr stores one block header at h.tip+1 and updates h.tip
// the header is assumed to be valid and can connect
func (h *headers) storeOneHdr(blkHdr *BlockHeader) {
	h.hdrsMtx.Lock()
	defer h.hdrsMtx.Unlock()
	at := h.getTip() + 1
	// at := h.tip + 1
	h.hdrs[at] = blkHdr
	blkHash := blkHdr.Hash
	h.blkHdrs[blkHash] = at
	h.incTip(1)
}

// Verify headers prev hash back from tip. If 'all' is true 'depth' is ignored
// and the whole chain is verified
func (h *headers) verifyFromTip(depth int64, all bool) error {
	h.hdrsMtx.RLock()
	defer h.hdrsMtx.RUnlock()
	downTo := h.getTip() - depth
	if downTo < 0 || all {
		downTo = h.startPoint
	}
	var height int64
	for height = h.getTip(); height > downTo; height-- {
		thisHdr := h.hdrs[height]
		prevHdr := h.hdrs[height-1]
		prevHdrBlkHash := prevHdr.Hash
		if prevHdrBlkHash != thisHdr.Prev {
			return fmt.Errorf("verify failed: height %d", height)
		}
		// fmt.Printf("verified header at height %d has blockhash %s\n",
		// 	height-1, prevHdrBlkHash.StringRev())
	}
	return nil
}

func (h *headers) verifyAll() error {
	return h.verifyFromTip(0, true)
}

// how many headers given a number of bytes .. panics if header size is 0
func (h *headers) bytesToNumHdrs(numBytes int64) (int64, error) {
	if numBytes == 0 {
		return 0, nil
	}
	headerSize := int64(h.headerSize)
	if headerSize <= 0 {
		panic("block header size is zero")
	}
	if numBytes%headerSize != 0 {
		return 0, errors.New(
			"invalid bytes length - not a multiple of header size")
	}
	return numBytes / headerSize, nil
}

// ----------------------------------------------------------------------------
// Client api
// ----------------------------------------------------------------------------

// getBlockHeader returns the block header for height. If out of range will
// return nil.
func (h *headers) getBlockHeader(height int64) (*ClientBlockHeader, error) {
	h.hdrsMtx.RLock()
	defer h.hdrsMtx.RUnlock()
	blkHdr := h.hdrs[height]
	if blkHdr == nil {
		return nil, fmt.Errorf("no block header stored for height %d", height)
	}
	hdr := &ClientBlockHeader{
		Hash:   blkHdr.Hash.StringRev(),
		Prev:   blkHdr.Prev.StringRev(),
		Merkle: blkHdr.Merkle.StringRev(),
	}
	return hdr, nil
}

// getBlockHeaders returns the stored block headers for the requested range.
// If startHeight < startPoint or startHeight > tip or startHeight+count > tip
// will return error.
func (h *headers) getBlockHeaders(startHeight, count int64) ([]*ClientBlockHeader, error) {
	h.hdrsMtx.RLock()
	defer h.hdrsMtx.RUnlock()
	if h.startPoint > startHeight {
		// error for now:
		//   If there is a need for blocks before the last checkpoint consider
		//   making a server call if api users need that.
		return nil, errors.New("requested start height < start of stored block headers")
	}
	if startHeight > h.getTip() {
		return nil, errors.New("requested start height > local tip")
	}
	blkEndRange := startHeight + count
	if blkEndRange > h.getTip() {
		return nil, errors.New("requested range is past the local tip")
	}
	var hdrs = make([]*ClientBlockHeader, 0, 3)
	for i := startHeight; i < blkEndRange; i++ {
		blkHdr := h.hdrs[i]
		hdr := &ClientBlockHeader{
			Hash:   blkHdr.Hash.StringRev(),
			Prev:   blkHdr.Prev.StringRev(),
			Merkle: blkHdr.Merkle.StringRev(),
		}
		hdrs = append(hdrs, hdr)
	}
	return hdrs, nil
}

// ----------------------------------------------------------------------------
// test
// ----------------------------------------------------------------------------

// Use *only* in headers_test.go
func (h *headers) ClearMaps() {
	h.hdrs = nil
	h.blkHdrs = nil
	h.hdrs = make(map[int64]*BlockHeader, 10)
	h.blkHdrs = make(map[WireHash]int64, 10)
}

// dump the top 'depth' hash - prev hashes
func (h *headers) dbgDumpTipHashes(depth int64) {
	tip := h.getTip()
	fmt.Printf("--- Dump of the top %d stored headers ---\n", depth)
	for i := tip; i > tip-depth; i-- {
		hash := h.hdrs[i].Hash.StringRev()
		prev := h.hdrs[i].Prev.StringRev()
		fmt.Printf("height: %d hash: %s prev: %s\n", i, hash, prev)
	}
}

// dump one decoded block header
func (h *headers) dumpAt(height int64) {
	h.hdrsMtx.Lock()
	defer h.hdrsMtx.Unlock()
	hdr := h.hdrs[height]
	fmt.Println("Hash:          ", hdr.Hash.StringRev(), "Height: ", height)
	fmt.Println("--------------------------")
	fmt.Println("Previous Hash: ", hdr.Prev.StringRev())
	fmt.Println("Merkle Root:   ", hdr.Merkle.StringRev())
	fmt.Println()
	fmt.Println("============================")
}

func (h *headers) dumpAll() {
	var k int64
	for k = h.startPoint; k < h.getTip(); k++ {
		h.dumpAt(k)
	}
}
