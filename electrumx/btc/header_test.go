package btc

import (
	"bytes"
	"log"
	"testing"

	"github.com/btcsuite/btcd/wire"
)

var hdr = []byte{
	0x00, 0x00, 0x00, 0x20, 0x06, 0x22, 0x6e, 0x46, 0x11, 0x1a, 0x0b, 0x59, 0xca, 0xaf, 0x12, 0x60,
	0x43, 0xeb, 0x5b, 0xbf, 0x28, 0xc3, 0x4f, 0x3a, 0x5e, 0x33, 0x2a, 0x1f, 0xc7, 0xb2, 0xb7, 0x3c,
	0xf1, 0x88, 0x91, 0x0f, 0x95, 0x2d, 0xaa, 0x4d, 0x5d, 0x1c, 0x84, 0x66, 0x7f, 0xb4, 0x86, 0x30,
	0x0a, 0x63, 0x20, 0x8a, 0x05, 0xe5, 0x0e, 0xbf, 0x41, 0xd8, 0xc3, 0x4a, 0x9f, 0x3b, 0xd3, 0x7b,
	0xd0, 0x45, 0x26, 0x4c, 0x42, 0x78, 0x33, 0x65, 0xff, 0xff, 0x7f, 0x20, 0x00, 0x00, 0x00, 0x00,
}

func TestDeserializeHeader(t *testing.T) {
	blkHdr := wire.BlockHeader{}
	r := bytes.NewBuffer(hdr)
	err := blkHdr.Deserialize(r)
	if err != nil {
		log.Fatal(err)
	}
}
