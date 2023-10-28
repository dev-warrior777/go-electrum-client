package btc

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/dev-warrior777/go-electrum-client/client"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

var hdrFileReg = []byte{
	0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x3b, 0xa3, 0xed, 0xfd, 0x7a, 0x7b, 0x12, 0xb2, 0x7a, 0xc7, 0x2c, 0x3e,
	0x67, 0x76, 0x8f, 0x61, 0x7f, 0xc8, 0x1b, 0xc3, 0x88, 0x8a, 0x51, 0x32, 0x3a, 0x9f, 0xb8, 0xaa,
	0x4b, 0x1e, 0x5e, 0x4a, 0xda, 0xe5, 0x49, 0x4d, 0xff, 0xff, 0x7f, 0x20, 0x02, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x20, 0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59, 0xca, 0xaf, 0x12, 0x60,
	0x43, 0xeb, 0x5b, 0xbf, 0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f, 0xc7, 0xb2, 0xb7, 0x3c,
	0xf1, 0x88, 0x91, 0x0f, 0x95, 0x2d, 0xaa, 0x4d, 0x5d, 0x1c, 0x84, 0x66, 0x7f, 0xb4, 0x86, 0x30,
	0x0a, 0x63, 0x20, 0x8a, 0x05, 0xe5, 0x0e, 0xbf, 0x41, 0xd8, 0xc3, 0x4a, 0x9f, 0x3b, 0xd3, 0x7b,
	0xd0, 0x45, 0x26, 0x4c, 0x87, 0x13, 0x39, 0x65, 0xff, 0xff, 0x7f, 0x20, 0x01, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x20, 0x18, 0xb9, 0x0e, 0x2b, 0xc1, 0x1a, 0x45, 0x1c, 0x8b, 0xd7, 0x9f, 0x30,
	0xa7, 0x82, 0xa1, 0xec, 0x7f, 0x68, 0x8f, 0x3c, 0xcd, 0xbe, 0x0e, 0x13, 0xbe, 0x01, 0x28, 0x7e,
	0x31, 0x29, 0x04, 0x5b, 0x8c, 0x41, 0xee, 0x85, 0x3a, 0x25, 0xbc, 0xc6, 0x7e, 0xad, 0xf4, 0x72,
	0xd7, 0x14, 0xa9, 0x06, 0x21, 0xa7, 0xd0, 0x76, 0x52, 0xf8, 0xb9, 0xc1, 0xd1, 0x96, 0x89, 0xdf,
	0x4d, 0x46, 0x79, 0xd4, 0x90, 0x13, 0x39, 0x65, 0xff, 0xff, 0x7f, 0x20, 0x03, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x20, 0x33, 0x74, 0xfa, 0x65, 0x26, 0x0c, 0xd6, 0x27, 0xb2, 0x3c, 0x0e, 0x12,
	0x74, 0xb1, 0x61, 0xb7, 0xab, 0x15, 0x1d, 0x04, 0x55, 0x12, 0xa2, 0x69, 0x57, 0x7a, 0x30, 0x86,
	0x9d, 0x0e, 0x15, 0x6d, 0xc1, 0xd1, 0xd2, 0xb8, 0x7d, 0xf2, 0x57, 0x1d, 0xb2, 0x7e, 0x49, 0x02,
	0xf5, 0xce, 0xef, 0x58, 0x9a, 0x28, 0xb3, 0x15, 0x0f, 0xdb, 0xaa, 0x80, 0x15, 0x2c, 0x0c, 0x19,
	0x9e, 0x11, 0x12, 0x2e, 0x90, 0x13, 0x39, 0x65, 0xff, 0xff, 0x7f, 0x20, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x20, 0x1f, 0x16, 0x0e, 0x22, 0x34, 0xb5, 0x85, 0x80, 0xf9, 0x3e, 0xb1, 0x3e,
	0xe0, 0xbf, 0x3d, 0x92, 0x3a, 0x3f, 0xc4, 0xad, 0x18, 0x89, 0xdb, 0xf3, 0x89, 0xf1, 0x4e, 0xa0,
	0x85, 0x07, 0x99, 0x20, 0xce, 0x0c, 0xe0, 0x96, 0xef, 0xa5, 0xbf, 0xba, 0xcf, 0x3a, 0xfe, 0xaf,
	0xf9, 0x77, 0x88, 0xf7, 0x66, 0xea, 0x0c, 0x66, 0x9c, 0x92, 0x6d, 0xa5, 0x87, 0x81, 0xcd, 0x80,
	0x75, 0xb4, 0x8e, 0x36, 0x91, 0x13, 0x39, 0x65, 0xff, 0xff, 0x7f, 0x20, 0x02, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x20, 0x5a, 0xbb, 0x7e, 0x0b, 0x5a, 0xec, 0x23, 0x01, 0xf2, 0x0f, 0x78, 0xd3,
	0x35, 0xfe, 0xf4, 0x3c, 0xca, 0x7d, 0x8b, 0xdf, 0x34, 0x06, 0x92, 0x87, 0x76, 0xcd, 0x0c, 0xfa,
	0x7a, 0xdb, 0x56, 0x15, 0xc5, 0xd6, 0xe7, 0x8d, 0x50, 0x81, 0x34, 0xc4, 0xd3, 0xca, 0xfe, 0xc1,
	0xf2, 0x0d, 0xbd, 0xcc, 0x99, 0x4e, 0xa6, 0x02, 0xf6, 0xdc, 0xef, 0x15, 0x8b, 0x0a, 0x16, 0x99,
	0x42, 0xa1, 0x03, 0x21, 0x91, 0x13, 0x39, 0x65, 0xff, 0xff, 0x7f, 0x20, 0x02, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x20, 0x4f, 0x4b, 0xf4, 0xe7, 0x0f, 0xa5, 0x4c, 0x87, 0x74, 0x69, 0x7e, 0x67,
	0x16, 0xb1, 0x99, 0x36, 0x74, 0x8a, 0x4c, 0x91, 0xd3, 0x98, 0x12, 0x3f, 0xe5, 0xfd, 0xcf, 0x25,
	0xb2, 0xfa, 0x04, 0x0c, 0x09, 0xad, 0xbd, 0x2e, 0xe9, 0x71, 0x54, 0x65, 0x59, 0x63, 0x11, 0xaf,
	0xb2, 0xb6, 0x7a, 0x6c, 0x40, 0x3c, 0x70, 0x2f, 0x3a, 0x5f, 0x1f, 0x7c, 0x4f, 0xaf, 0xe3, 0x3f,
	0xf6, 0xa9, 0xa1, 0x61, 0x91, 0x13, 0x39, 0x65, 0xff, 0xff, 0x7f, 0x20, 0x02, 0x00, 0x00, 0x00,
}

func mkHdrFile() (*os.File, int, error) {
	f, err := os.CreateTemp("/tmp", "cli_tst_")
	if err != nil {
		return nil, 0, err
	}
	n, err := f.Write(hdrFileReg)
	if err != nil {
		return nil, n, err
	}
	if n != len(hdrFileReg) {
		return nil, n, errors.New("file read truncated")
	}
	fmt.Printf("read and stored %d bytes into new headerfile\n", n)
	return f, n, nil
}

func makeRegtestConfig() (*client.ClientConfig, error) {
	cfg := client.NewDefaultConfig()
	cfg.Chain = wallet.Bitcoin
	cfg.Params = &chaincfg.RegressionNetParams
	cfg.StoreEncSeed = true
	appDir, err := client.GetConfigPath()
	if err != nil {
		return nil, err
	}
	regtestDir := filepath.Join(appDir, "btc", "regtest")
	err = os.MkdirAll(regtestDir, os.ModeDir|0777)
	if err != nil {
		return nil, err
	}
	cfg.DataDir = regtestDir
	return cfg, nil
}

func TestAppendHeaders(t *testing.T) {
	f, err := os.CreateTemp("/tmp", "cli_tst_")
	if err != nil {
		log.Fatal(err)
	}
	fi, err := f.Stat()
	if err != nil {
		fmt.Println(err.Error())
		log.Fatal(err)
	}
	fname := fi.Name()
	f.Close()

	h := Headers{
		hdrFilePath: path.Join("/tmp", fname),
		net:         &chaincfg.RegressionNetParams,
		hdrs:        make(map[int32]wire.BlockHeader),
		hdrsTip:     0,
		synced:      false,
	}

	var totalHdrs int32 = 0
	numHdrs, err := h.AppendHeaders(hdrFileReg[:160])
	if err != nil {
		log.Fatal(err)
	}
	totalHdrs += numHdrs
	fmt.Println(numHdrs, " headers stored")
	numHdrs, err = h.AppendHeaders(hdrFileReg[160:320])
	if err != nil {
		log.Fatal(err)
	}
	totalHdrs += numHdrs
	fmt.Println(numHdrs, " headers stored")
	numHdrs, err = h.AppendHeaders(hdrFileReg[320:])
	if err != nil {
		log.Fatal(err)
	}
	totalHdrs += numHdrs
	fmt.Println(numHdrs, " headers stored")

	fsize, err := h.StatFileSize()
	if err != nil {
		fmt.Println(err.Error())
		log.Fatal(err)
	}
	fmt.Println(fsize, " total bytes stored ", totalHdrs, ", total headers stored")

	if totalHdrs != int32(fsize/HEADER_SIZE) {
		log.Fatal("total headers wrong")
	}

	// 'finished' appending - now grab the bytes back
	maybeTip := totalHdrs - 1

	// read back bytes from file
	b, err := h.ReadAllBytesFromFile()
	if err != nil {
		log.Fatal(err)
	}

	// store all bytes into the map
	err = h.store(b, 0)
	if err != nil {
		log.Fatal(err)
	}
	h.hdrsTip = maybeTip

	// verify chain
	var height int32
	for height = h.hdrsTip; height > 0; height-- {
		thisHdr := h.hdrs[height]
		prevHdr := h.hdrs[height-1]
		prevHdrBlkHash := prevHdr.BlockHash()
		if prevHdr.BlockHash() != thisHdr.PrevBlock {
			log.Fatal("header chain verify failed")
		}
		fmt.Printf("verified header at height %d has blockhash %s\n", height-1, prevHdrBlkHash.String())
	}
}

func TestReadStoreHeaderFile(t *testing.T) {
	f, n, err := mkHdrFile()
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	f.Seek(0, 0)

	// read all from file as bytes
	b := make([]byte, n)
	read, err := f.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	if read != n {
		log.Fatal("read truncated")
	}
	if read%HEADER_SIZE != 0 {
		log.Fatal("invalid file size")
	}
	numHdrs := int32(read / HEADER_SIZE)
	fmt.Printf("read %d bytes %d headers from headerfile\n", read, numHdrs)

	// store headers
	cfg, _ := makeRegtestConfig()
	h, _ := NewHeaders(cfg)
	err = h.store(b, 0)
	if err != nil {
		log.Fatal(err)
	}
	h.hdrsTip = numHdrs - 1
	fmt.Printf("stored %d headers into hdrs map at height %d\n", numHdrs, 0)

	// verify chain
	var height int32
	for height = h.hdrsTip; height > 0; height-- {
		thisHdr := h.hdrs[height]
		prevHdr := h.hdrs[height-1]
		prevHdrBlkHash := prevHdr.BlockHash()
		if prevHdr.BlockHash() != thisHdr.PrevBlock {
			log.Fatal("header chain verify failed")
		}
		fmt.Printf("verified header at height %d has blockhash %s\n", height-1, prevHdrBlkHash.String())
	}
}

var hdr = []byte{
	0x00, 0x00, 0x00, 0x20, 0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59, 0xca, 0xaf, 0x12, 0x60,
	0x43, 0xeb, 0x5b, 0xbf, 0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f, 0xc7, 0xb2, 0xb7, 0x3c,
	0xf1, 0x88, 0x91, 0x0f, 0x95, 0x2d, 0xaa, 0x4d, 0x5d, 0x1c, 0x84, 0x66, 0x7f, 0xb4, 0x86, 0x30,
	0x0a, 0x63, 0x20, 0x8a, 0x05, 0xe5, 0x0e, 0xbf, 0x41, 0xd8, 0xc3, 0x4a, 0x9f, 0x3b, 0xd3, 0x7b,
	0xd0, 0x45, 0x26, 0x4c, 0x42, 0x78, 0x33, 0x65, 0xff, 0xff, 0x7f, 0x20, 0x00, 0x00, 0x00, 0x00,
}

var hdrBadLenLess = []byte{
	0x00, 0x00, 0x00, 0x20, 0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59, 0xca, 0xaf, 0x12, 0x60,
	0x43, 0xeb, 0x5b, 0xbf, 0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f, 0xc7, 0xb2, 0xb7, 0x3c,
	0xf1, 0x88, 0x91, 0x0f, 0x95, 0x2d, 0xaa, 0x4d, 0x5d, 0x1c, 0x84, 0x66, 0x7f, 0xb4, 0x86, 0x30,
	0x0a, 0x63, 0x20, 0x8a, 0x05, 0xe5, 0x0e, 0xbf, 0x41, 0xd8, 0xc3, 0x4a, 0x9f, 0x3b, 0xd3, 0x7b,
	0xd0, 0x45, 0x26, 0x4c, 0x42, 0x78, 0x33, 0x65, 0xff, 0xff, 0x7f, 0x20,
}

var hdrBadLenMore = []byte{
	0x00, 0x00, 0x00, 0x20, 0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59, 0xca, 0xaf, 0x12, 0x60,
	0x43, 0xeb, 0x5b, 0xbf, 0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f, 0xc7, 0xb2, 0xb7, 0x3c,
	0xf1, 0x88, 0x91, 0x0f, 0x95, 0x2d, 0xaa, 0x4d, 0x5d, 0x1c, 0x84, 0x66, 0x7f, 0xb4, 0x86, 0x30,
	0x0a, 0x63, 0x20, 0x8a, 0x05, 0xe5, 0x0e, 0xbf, 0x41, 0xd8, 0xc3, 0x4a, 0x9f, 0x3b, 0xd3, 0x7b,
	0xd0, 0x45, 0x26, 0x4c, 0x42, 0x78, 0x33, 0x65, 0xff, 0xff, 0x7f, 0x20, 0x00, 0x00, 0x00, 0x00,
	0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA,
}

var hdr3 = []byte{
	0x00, 0x00, 0x00, 0x20, 0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59, 0xca, 0xaf, 0x12, 0x60,
	0x43, 0xeb, 0x5b, 0xbf, 0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f, 0xc7, 0xb2, 0xb7, 0x3c,
	0xf1, 0x88, 0x91, 0x0f, 0x95, 0x2d, 0xaa, 0x4d, 0x5d, 0x1c, 0x84, 0x66, 0x7f, 0xb4, 0x86, 0x30,
	0x0a, 0x63, 0x20, 0x8a, 0x05, 0xe5, 0x0e, 0xbf, 0x41, 0xd8, 0xc3, 0x4a, 0x9f, 0x3b, 0xd3, 0x7b,
	0xd0, 0x45, 0x26, 0x4c, 0x42, 0x78, 0x33, 0x65, 0xff, 0xff, 0x7f, 0x20, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x20, 0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59, 0xca, 0xaf, 0x12, 0x60,
	0x43, 0xeb, 0x5b, 0xbf, 0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f, 0xc7, 0xb2, 0xb7, 0x3c,
	0xf1, 0x88, 0x91, 0x0f, 0x95, 0x2d, 0xaa, 0x4d, 0x5d, 0x1c, 0x84, 0x66, 0x7f, 0xb4, 0x86, 0x30,
	0x0a, 0x63, 0x20, 0x8a, 0x05, 0xe5, 0x0e, 0xbf, 0x41, 0xd8, 0xc3, 0x4a, 0x9f, 0x3b, 0xd3, 0x7b,
	0xd0, 0x45, 0x26, 0x4c, 0x42, 0x78, 0x33, 0x65, 0xff, 0xff, 0x7f, 0x20, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x20, 0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59, 0xca, 0xaf, 0x12, 0x60,
	0x43, 0xeb, 0x5b, 0xbf, 0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f, 0xc7, 0xb2, 0xb7, 0x3c,
	0xf1, 0x88, 0x91, 0x0f, 0x95, 0x2d, 0xaa, 0x4d, 0x5d, 0x1c, 0x84, 0x66, 0x7f, 0xb4, 0x86, 0x30,
	0x0a, 0x63, 0x20, 0x8a, 0x05, 0xe5, 0x0e, 0xbf, 0x41, 0xd8, 0xc3, 0x4a, 0x9f, 0x3b, 0xd3, 0x7b,
	0xd0, 0x45, 0x26, 0x4c, 0x42, 0x78, 0x33, 0x65, 0xff, 0xff, 0x7f, 0x20, 0x00, 0x00, 0x00, 0x00,
}

func TestStore(t *testing.T) {
	cfg, _ := makeRegtestConfig()
	h, _ := NewHeaders(cfg)
	err := h.store(hdr, 0)
	if err != nil {
		log.Fatal(err)
	}
	h.clearMap()
	err = h.store(hdr3, 0)
	if err != nil {
		log.Fatal(err)
	}
	h.clearMap()
	err = h.store(hdr3, 100)
	if err != nil {
		log.Fatal(err)
	}
	h.clearMap()
	err = h.store(hdr, 0)
	if err != nil {
		log.Fatal("error expected")
	}
	h.clearMap()
	err = h.store(hdr3, 0)
	if err != nil {
		log.Fatal(err)
	}
	// extra bytes are ignored
	h.clearMap()
	err = h.store(hdrBadLenMore, 0)
	if err == nil {
		log.Fatal("error expected")
	}
	// less than expected size is not ignored
	h.clearMap()
	err = h.store(hdrBadLenLess, 0)
	if err == nil {
		log.Fatal("error expected")
	}
}
