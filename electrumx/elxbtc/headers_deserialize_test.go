package elxbtc

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/dev-warrior777/go-electrum-client/electrumx"
)

// Data taken from btc regtest which has a block size 0f 80 and nonce 0

var hdr = []byte{
	0x00, 0x00, 0x00, 0x20,
	0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59, 0xca, 0xaf, 0x12, 0x60, 0x43, 0xeb, 0x5b, 0xbf,
	0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f, 0xc7, 0xb2, 0xb7, 0x3c, 0xf1, 0x88, 0x91, 0x0f,
	0x95, 0x2d, 0xaa, 0x4d, 0x5d, 0x1c, 0x84, 0x66, 0x7f, 0xb4, 0x86, 0x30, 0x0a, 0x63, 0x20, 0x8a,
	0x05, 0xe5, 0x0e, 0xbf, 0x41, 0xd8, 0xc3, 0x4a, 0x9f, 0x3b, 0xd3, 0x7b, 0xd0, 0x45, 0x26, 0x4c,
	0x42, 0x78, 0x33, 0x65,
	0xff, 0xff, 0x7f, 0x20,
	0x00, 0x00, 0x00, 0x00,
}

var hdrBadLenLess = []byte{
	0x00, 0x00, 0x00, 0x20,
	0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59, 0xca, 0xaf, 0x12, 0x60, 0x43, 0xeb, 0x5b, 0xbf,
	0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f, 0xc7, 0xb2, 0xb7, 0x3c, 0xf1, 0x88, 0x91, 0x0f,
	0x95, 0x2d, 0xaa, 0x4d, 0x5d, 0x1c, 0x84, 0x66, 0x7f, 0xb4, 0x86, 0x30, 0x0a, 0x63, 0x20, 0x8a,
	0x05, 0xe5, 0x0e, 0xbf, 0x41, 0xd8, 0xc3, 0x4a, 0x9f, 0x3b, 0xd3, 0x7b, 0xd0, 0x45, 0x26, 0x4c,
	0x42, 0x78, 0x33, 0x65,
	0xff, 0xff, 0x7f, 0x20,
	// less
}

var hdrBadLenMore = []byte{
	0x00, 0x00, 0x00, 0x20,
	0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59, 0xca, 0xaf, 0x12, 0x60, 0x43, 0xeb, 0x5b, 0xbf,
	0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f, 0xc7, 0xb2, 0xb7, 0x3c, 0xf1, 0x88, 0x91, 0x0f,
	0x95, 0x2d, 0xaa, 0x4d, 0x5d, 0x1c, 0x84, 0x66, 0x7f, 0xb4, 0x86, 0x30, 0x0a, 0x63, 0x20, 0x8a,
	0x05, 0xe5, 0x0e, 0xbf, 0x41, 0xd8, 0xc3, 0x4a, 0x9f, 0x3b, 0xd3, 0x7b, 0xd0, 0x45, 0x26, 0x4c,
	0x42, 0x78, 0x33, 0x65,
	0xff, 0xff, 0x7f, 0x20,
	0x00, 0x00, 0x00, 0x00,
	/* more */
	0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA,
}

func mkCfg() *electrumx.ElectrumXConfig {
	return &electrumx.ElectrumXConfig{
		Coin:    "btc",
		NetType: "regtest",
		DataDir: "/tmp",
	}
}

func TestHeaderDeserializer(t *testing.T) {
	cfg := mkCfg()

	iface, _ := NewElectrumXInterface(cfg)
	d := iface.config.HeaderDeserializer

	rdr := bytes.NewBuffer(hdr)
	blkHdr, err := d.Deserialize(rdr)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%x\n", blkHdr.Hash.String())
	if !bytes.Equal([]byte(blkHdr.Hash[:]), []byte{0x73, 0x07, 0x97, 0x74, 0x17, 0xea, 0x9f, 0x17, 0x90, 0xc8, 0x03, 0x88, 0x64, 0x8e, 0xd8, 0x16, 0x26, 0x50, 0xbe, 0x04, 0x45, 0x2b, 0x6b, 0x1d, 0xe8, 0xff, 0x9a, 0xd4, 0x2b, 0x36, 0x45, 0x23}) {
		t.Fatal("sha256 doublehash error")
	}

	// not a full header
	rdrLess := bytes.NewBuffer(hdrBadLenLess)
	_, err = d.Deserialize(rdrLess)
	if err == nil {
		t.Fatal(err)
	}
	// wire.BlockHash reader only reads 80 bytes so no fail on the deserialization

	rdrMore := bytes.NewBuffer(hdrBadLenMore)
	_, err = d.Deserialize(rdrMore)
	if err != nil {
		t.Fatal(err)
	}
}

// func TestReadStoreHeaderFile(t *testing.T) {
// 	f, n, err := mkHdrFile()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer f.Close()
// 	f.Seek(0, 0)

// 	// read all from file as bytes
// 	b := make([]byte, n)
// 	read, err := f.Read(b)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	if read != n {
// 		log.Fatal("read truncated")
// 	}
// 	// store headers
// 	h := headers{
// 		headerSize:  80,
// 		hdrFilePath: f.Name(),
// 		p:           &chaincfg.RegressionNetParams,
// 		hdrs:        make(map[int64]*wire.BlockHeader),
// 		blkHdrs:     make(map[chainhash.Hash]int64),
// 		tip:         0,
// 		synced:      false,
// 	}
// 	if read%h.headerSize != 0 {
// 		log.Fatal("invalid file size")
// 	}
// 	numHdrs := read / h.headerSize
// 	fmt.Printf("read %d bytes %d headers from headerfile\n", read, numHdrs)

// 	err = h.store(b, 0)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	h.tip = int64(numHdrs - 1)
// 	fmt.Printf("stored %d headers into hdrs map at height %d\n", numHdrs, 0)
// 	// verify chain
// 	err = h.verifyAll()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	// just visual confirmation
// 	var height int64
// 	for height = h.tip; height > 0; height-- {
// 		thisHdr := h.hdrs[height]
// 		prevHdr := h.hdrs[height-1]
// 		prevHdrBlkHash := prevHdr.BlockHash()
// 		if prevHdr.BlockHash() != thisHdr.PrevBlock {
// 			log.Fatal("header chain verify failed")
// 		}
// 		fmt.Printf("verified header at height %d has blockhash %s\n", height-1, prevHdrBlkHash.String())
// 	}
// }

// func TestTruncateHeadersFile(t *testing.T) {
// 	f, size, err := mkHdrFile()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	fname := f.Name()
// 	defer f.Close()
// 	h := headers{
// 		headerSize:  80,
// 		hdrFilePath: fname,
// 		p:           &chaincfg.RegressionNetParams,
// 		hdrs:        make(map[int64]*wire.BlockHeader),
// 		blkHdrs:     make(map[chainhash.Hash]int64),
// 		tip:         0,
// 		synced:      false,
// 	}
// 	numHeaders, err := h.bytesToNumHdrs(int64(size))
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	_, err = h.truncateHeadersFile(1)
// 	if err == nil {
// 		log.Fatal(err)
// 	}
// 	h.synced = true
// 	_, err = h.truncateHeadersFile(-1)
// 	if err == nil {
// 		log.Fatal(err)
// 	}
// 	newNumHeaders, err := h.truncateHeadersFile(1)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	if newNumHeaders != numHeaders-1 {
// 		log.Fatalf("invaid number of headers returned %d", newNumHeaders)
// 	}
// 	newNumHeaders, err = h.truncateHeadersFile(5)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	if newNumHeaders != numHeaders-1-5 {
// 		log.Fatalf("invaid number of headers returned %d", newNumHeaders)
// 	}
// 	newNumHeaders, err = h.truncateHeadersFile(1)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	if newNumHeaders != numHeaders-1-5-1 {
// 		log.Fatalf("invaid number of headers returned %d", newNumHeaders)
// 	}

// 	newNumHeaders, err = h.truncateHeadersFile(1)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	if newNumHeaders != 0 {
// 		log.Fatalf("invaid number of headers returned %d", newNumHeaders)
// 	}
// }

// var hdr = []byte{
// 	0x00, 0x00, 0x00, 0x20, 0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59, 0xca, 0xaf, 0x12, 0x60,
// 	0x43, 0xeb, 0x5b, 0xbf, 0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f, 0xc7, 0xb2, 0xb7, 0x3c,
// 	0xf1, 0x88, 0x91, 0x0f, 0x95, 0x2d, 0xaa, 0x4d, 0x5d, 0x1c, 0x84, 0x66, 0x7f, 0xb4, 0x86, 0x30,
// 	0x0a, 0x63, 0x20, 0x8a, 0x05, 0xe5, 0x0e, 0xbf, 0x41, 0xd8, 0xc3, 0x4a, 0x9f, 0x3b, 0xd3, 0x7b,
// 	0xd0, 0x45, 0x26, 0x4c, 0x42, 0x78, 0x33, 0x65, 0xff, 0xff, 0x7f, 0x20, 0x00, 0x00, 0x00, 0x00,
// }

// var hdrBadLenLess = []byte{
// 	0x00, 0x00, 0x00, 0x20, 0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59, 0xca, 0xaf, 0x12, 0x60,
// 	0x43, 0xeb, 0x5b, 0xbf, 0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f, 0xc7, 0xb2, 0xb7, 0x3c,
// 	0xf1, 0x88, 0x91, 0x0f, 0x95, 0x2d, 0xaa, 0x4d, 0x5d, 0x1c, 0x84, 0x66, 0x7f, 0xb4, 0x86, 0x30,
// 	0x0a, 0x63, 0x20, 0x8a, 0x05, 0xe5, 0x0e, 0xbf, 0x41, 0xd8, 0xc3, 0x4a, 0x9f, 0x3b, 0xd3, 0x7b,
// 	0xd0, 0x45, 0x26, 0x4c, 0x42, 0x78, 0x33, 0x65, 0xff, 0xff, 0x7f, 0x20, // less
// }

// var hdrBadLenMore = []byte{
// 	0x00, 0x00, 0x00, 0x20, 0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59, 0xca, 0xaf, 0x12, 0x60,
// 	0x43, 0xeb, 0x5b, 0xbf, 0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f, 0xc7, 0xb2, 0xb7, 0x3c,
// 	0xf1, 0x88, 0x91, 0x0f, 0x95, 0x2d, 0xaa, 0x4d, 0x5d, 0x1c, 0x84, 0x66, 0x7f, 0xb4, 0x86, 0x30,
// 	0x0a, 0x63, 0x20, 0x8a, 0x05, 0xe5, 0x0e, 0xbf, 0x41, 0xd8, 0xc3, 0x4a, 0x9f, 0x3b, 0xd3, 0x7b,
// 	0xd0, 0x45, 0x26, 0x4c, 0x42, 0x78, 0x33, 0x65, 0xff, 0xff, 0x7f, 0x20, 0x00, 0x00, 0x00, 0x00,
// 	/* more */
// 	0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA,
// }

// var hdr3 = []byte{
// 	0x00, 0x00, 0x00, 0x20, 0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59, 0xca, 0xaf, 0x12, 0x60,
// 	0x43, 0xeb, 0x5b, 0xbf, 0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f, 0xc7, 0xb2, 0xb7, 0x3c,
// 	0xf1, 0x88, 0x91, 0x0f, 0x95, 0x2d, 0xaa, 0x4d, 0x5d, 0x1c, 0x84, 0x66, 0x7f, 0xb4, 0x86, 0x30,
// 	0x0a, 0x63, 0x20, 0x8a, 0x05, 0xe5, 0x0e, 0xbf, 0x41, 0xd8, 0xc3, 0x4a, 0x9f, 0x3b, 0xd3, 0x7b,
// 	0xd0, 0x45, 0x26, 0x4c, 0x42, 0x78, 0x33, 0x65, 0xff, 0xff, 0x7f, 0x20, 0x00, 0x00, 0x00, 0x00,
// 	0x00, 0x00, 0x00, 0x20, 0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59, 0xca, 0xaf, 0x12, 0x60,
// 	0x43, 0xeb, 0x5b, 0xbf, 0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f, 0xc7, 0xb2, 0xb7, 0x3c,
// 	0xf1, 0x88, 0x91, 0x0f, 0x95, 0x2d, 0xaa, 0x4d, 0x5d, 0x1c, 0x84, 0x66, 0x7f, 0xb4, 0x86, 0x30,
// 	0x0a, 0x63, 0x20, 0x8a, 0x05, 0xe5, 0x0e, 0xbf, 0x41, 0xd8, 0xc3, 0x4a, 0x9f, 0x3b, 0xd3, 0x7b,
// 	0xd0, 0x45, 0x26, 0x4c, 0x42, 0x78, 0x33, 0x65, 0xff, 0xff, 0x7f, 0x20, 0x00, 0x00, 0x00, 0x00,
// 	0x00, 0x00, 0x00, 0x20, 0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59, 0xca, 0xaf, 0x12, 0x60,
// 	0x43, 0xeb, 0x5b, 0xbf, 0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f, 0xc7, 0xb2, 0xb7, 0x3c,
// 	0xf1, 0x88, 0x91, 0x0f, 0x95, 0x2d, 0xaa, 0x4d, 0x5d, 0x1c, 0x84, 0x66, 0x7f, 0xb4, 0x86, 0x30,
// 	0x0a, 0x63, 0x20, 0x8a, 0x05, 0xe5, 0x0e, 0xbf, 0x41, 0xd8, 0xc3, 0x4a, 0x9f, 0x3b, 0xd3, 0x7b,
// 	0xd0, 0x45, 0x26, 0x4c, 0x42, 0x78, 0x33, 0x65, 0xff, 0xff, 0x7f, 0x20, 0x00, 0x00, 0x00, 0x00,
// }

// func TestStore(t *testing.T) {
// 	h := headers{
// 		headerSize:  80,
// 		hdrFilePath: "<empty>",
// 		p:           &chaincfg.RegressionNetParams,
// 		hdrs:        make(map[int64]*wire.BlockHeader),
// 		blkHdrs:     make(map[chainhash.Hash]int64),
// 		tip:         0,
// 		synced:      false,
// 	}
// 	err := h.store(hdr, 0)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	h.ClearMaps()
// 	err = h.store(hdr3, 0)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	h.ClearMaps()
// 	err = h.store(hdr3, 100)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	h.ClearMaps()
// 	err = h.store(hdr, 0)
// 	if err != nil {
// 		log.Fatal("error expected")
// 	}
// 	h.ClearMaps()
// 	err = h.store(hdr3, 0)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	// extra bytes are ignored
// 	h.ClearMaps()
// 	err = h.store(hdrBadLenMore, 0)
// 	if err == nil {
// 		log.Fatal("error expected")
// 	}
// 	// less than expected size is not ignored
// 	h.ClearMaps()
// 	err = h.store(hdrBadLenLess, 0)
// 	if err == nil {
// 		log.Fatal("error expected")
// 	}
// }

// func TestMapIter(t *testing.T) {
// 	h := headers{
// 		headerSize:  80,
// 		hdrFilePath: "<empty>",
// 		p:           &chaincfg.RegressionNetParams,
// 		hdrs:        make(map[int64]*wire.BlockHeader),
// 		blkHdrs:     make(map[chainhash.Hash]int64),
// 		tip:         0,
// 		synced:      false,
// 	}
// 	err := h.store(hdrFileReg, 0)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	numBytes := int64(len(hdrFileReg))
// 	numHeaders, err := h.bytesToNumHdrs(numBytes)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	h.tip = numHeaders - 1
// 	h.dumpAll()
// }

// func TestStoreHashes(t *testing.T) {
// 	h := headers{
// 		headerSize:  80,
// 		hdrFilePath: "<empty>",
// 		p:           &chaincfg.RegressionNetParams,
// 		hdrs:        make(map[int64]*wire.BlockHeader),
// 		blkHdrs:     make(map[chainhash.Hash]int64),
// 		tip:         0,
// 		synced:      false,
// 	}
// 	err := h.store(hdrFileReg, 0)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	numBytes := int64(len(hdrFileReg))
// 	numHeaders, err := h.bytesToNumHdrs(numBytes)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	h.tip = numHeaders - 1
// 	var i int64
// 	for i = 0; i <= h.tip; i++ {
// 		hdr := h.hdrs[i]
// 		if hdr == nil {
// 			log.Fatalf("nil header returned from map at %d", i)
// 		}
// 		blkHash := hdr.BlockHash()
// 		height := h.blkHdrs[blkHash]
// 		if i != height {
// 			t.Errorf("height mismatch: wanted %d got %d", i, height)
// 		}
// 	}
// }

// var hdrSerialized = []byte{
// 	0x00, 0x00, 0x00, 0x20, 0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59, 0xca, 0xaf, 0x12, 0x60,
// 	0x43, 0xeb, 0x5b, 0xbf, 0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f, 0xc7, 0xb2, 0xb7, 0x3c,
// 	0xf1, 0x88, 0x91, 0x0f, 0x95, 0x2d, 0xaa, 0x4d, 0x5d, 0x1c, 0x84, 0x66, 0x7f, 0xb4, 0x86, 0x30,
// 	0x0a, 0x63, 0x20, 0x8a, 0x05, 0xe5, 0x0e, 0xbf, 0x41, 0xd8, 0xc3, 0x4a, 0x9f, 0x3b, 0xd3, 0x7b,
// 	0xd0, 0x45, 0x26, 0x4c, 0x42, 0x78, 0x33, 0x65, 0xff, 0xff, 0x7f, 0x20, 0x00, 0x00, 0x00, 0x00,
// }

// func TestDeserializeHeader(t *testing.T) {
// 	blkHdr := wire.BlockHeader{}
// 	r := bytes.NewBuffer(hdrSerialized)
// 	err := blkHdr.Deserialize(r)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// }