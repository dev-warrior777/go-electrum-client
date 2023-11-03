package btc

// This is the Client's copy of the blockchain headers for a blockchain
// Backed by a file in the datadir of the chain (main, test, reg nets)
// We store here as a map and not a tree so we must trust the server if
// SingleNode. When grabbing new blocks some attempt is made to understand
// forks but the true longest chain with the most work cannot be known
// without connecting to many servers using MultiNode.

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/dev-warrior777/go-electrum-client/client"
)

const (
	HEADER_SIZE           = 80
	HEADER_FILE_NAME      = "blockchain_headers"
	ELECTRUM_MAGIC_NUMHDR = 2016
)

var (
	pver     = wire.ProtocolVersion
	maybeTip = int32(0)
)

type Headers struct {
	// blockchain headers file to persist headers we know
	hdrFilePath string
	// chain parameters for genesis and checkpoint. We always use the latest
	// checkpoint height to start the file. For regtest that is genesis.
	net *chaincfg.Params
	// decoded headers stored by height
	hdrsMtx sync.RWMutex
	hdrs    map[int32]wire.BlockHeader
	hdrsTip int32
	synced  bool
}

func NewHeaders(cfg *client.ClientConfig) (*Headers, error) {
	filePath := filepath.Join(cfg.DataDir, HEADER_FILE_NAME)
	hdrsMapInitSize := 2 * ELECTRUM_MAGIC_NUMHDR //4032
	hdrsMap := make(map[int32]wire.BlockHeader, hdrsMapInitSize)
	hdrs := Headers{
		hdrFilePath: filePath,
		net:         cfg.Params,
		hdrs:        hdrsMap,
		hdrsTip:     0,
		synced:      false,
	}
	return &hdrs, nil
}

func (h *Headers) clearMap() {
	h.hdrs = nil // gc
	h.hdrs = make(map[int32]wire.BlockHeader)
}

// Get the 'blockchin_headers' file size. Error is returned unexamined as
// we assume the file exists and ENOENT will not be valid.
func (h *Headers) StatFileSize() (int64, error) {
	fi, err := os.Stat(h.hdrFilePath)
	if err != nil {
		fmt.Println(err.Error())
		return 0, err
	}
	return fi.Size(), nil
}

// Read num headers from offset in 'blockchain_headers' file
func (h *Headers) ReadHeaders(num, height int32) (int32, error) {
	begin := int64(height * HEADER_SIZE)
	f, err := os.OpenFile(h.hdrFilePath, os.O_CREATE|os.O_RDWR, 0664)
	if err != nil {
		return 0, err
	}
	f.Seek(begin, 0)
	bytesToRead := num * HEADER_SIZE
	b := make([]byte, bytesToRead)
	bytesRead, err := f.Read(b)
	if err != nil {
		return 0, err
	}
	if bytesRead == 0 { // empty
		return 0, nil
	}
	if bytesRead < int(bytesToRead) {
		if bytesRead%HEADER_SIZE != 0 {
			return 0, errors.New("corrupt file")
		}
		headersRead := int32(bytesRead / HEADER_SIZE)
		err = h.store(b, height)
		return headersRead, err
	}
	return 0, nil
}

// AppendHeaders appends headers from server 'blockchain.block.header(s)' calls
// to 'blockchain_headers' file. Also appends headers received from the
// 'blockchain.headers.subscribe' events. Returns the number of headers written.
func (h *Headers) AppendHeaders(rawHdrs []byte) (int32, error) {
	numBytes := len(rawHdrs)
	numHdrs, err := bytesToNumHdrs(numBytes)
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

	return int32(numHdrs), nil
}

func (h *Headers) ReadAllBytesFromFile() ([]byte, error) {
	hdrFile, err := os.OpenFile(h.hdrFilePath, os.O_CREATE|os.O_RDWR, 0664)
	if err != nil {
		return nil, err
	}
	defer hdrFile.Close()
	fsize, err := h.StatFileSize()
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
// 'b' should have exactly 'numHdrs' x 'HEADER_SIZE' bytes. A few more
// bytes will be ignored here though. A few less will error EOF. Caveat emptor!
func (h *Headers) store(b []byte, height int32) error {
	numHdrs, err := bytesToNumHdrs(len(b))
	if err != nil {
		return err
	}
	rdr := bytes.NewBuffer(b)
	h.hdrsMtx.Lock()
	defer h.hdrsMtx.Unlock()
	var i int32
	for i = 0; i < numHdrs; i++ {
		blkHdr := wire.BlockHeader{}
		err := blkHdr.Deserialize(rdr)
		if err != nil {
			return err
		}
		at := height + i
		h.hdrs[at] = blkHdr
	}
	return nil
}

func bytesToNumHdrs(numBytes int) (int32, error) {
	if numBytes%HEADER_SIZE != 0 {
		return 0, errors.New("invalid bytes length - not a multiple of header size")
	}
	return int32(numBytes / HEADER_SIZE), nil
}

func (h *Headers) BytesToNumHdrs(numBytes int) (int32, error) {
	return bytesToNumHdrs(numBytes)
}
