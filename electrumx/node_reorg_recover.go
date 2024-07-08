package electrumx

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/wire"
)

const REWIND = 8

// ElectrumX *does* fixup reorgs when it sees them but like us it cannot know
// until the next block so may send us an unconnectable block in the notification
// or block headers we ask for on that notification.
//
// So this is not a question of looking on other peers' chains for chains with more
// proof of work - ElectrumX does that.
//
// When we cannot connect a block header we wind back our tip, hdrs map and truncate
// blockchain_headers file by REWIND block headers so that next time a notification
// comes in we ask for the last REWIND blocks. If still unconnectable on the next
// headers notification we wind back again until startPoint.
func (n *Node) reorgRecovery() {
	h := n.networkHeaders
	tip := h.getTip()

	fmt.Printf("reorgRecovery: tip: %d startPoint: %d\n", tip, h.startPoint)

	if tip <= h.startPoint+REWIND {
		panic("reorgRecovery: tip <= startPoint+REWIND - cannot recover further")
	}

	// truncate and remove from map atomically
	newNumHeaders, err := h.truncateHeadersFile(REWIND)
	if err != nil {
		errMsg := fmt.Sprintf("reorgRecovery: truncateHeadersFile returned: %v", err)
		panic(errMsg)
	}

	fmt.Printf("truncateHeadersFile: new num headers is %d\n", newNumHeaders)

	for i := 0; i < REWIND; i++ {
		h.removeHdrFromTip() // (sets tip--)
	}

	h.recovery = true
	h.recoveryTip = tip // what we  send back to users in getTip() during recovery
}

// ----------------------------------------------------------------------------
// debug remove
// ----------------------------------------------------------------------------

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
