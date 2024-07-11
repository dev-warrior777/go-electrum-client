package electrumx

// ----------------------------------------------------------------------------
// debug remove
// ----------------------------------------------------------------------------

// func (n *Node) dbgStringHeaderPrev(svrHdr string) string {
// 	h := n.networkHeaders
// 	rawBytes, err := hex.DecodeString(svrHdr)
// 	if err != nil {
// 		return "<hex decode error>"
// 	}
// 	if len(rawBytes) != int(h.headerSize) {
// 		return "<corrupted header>"
// 	}
// 	r := bytes.NewBuffer(rawBytes)
// 	hdr := &wire.BlockHeader{}
// 	err = hdr.BtcDecode(r, 0, wire.BaseEncoding)
// 	if err != nil {
// 		return "<deserilaize error>"
// 	}
// 	return hdr.PrevBlock.String()
// }

// func (n *Node) dbgHashHexFromHeaderHex(svrHdr string) string {
// 	hdr, _, err := n.convertStringHdrToBlkHdr(svrHdr)
// 	if err != nil {
// 		return "*conversion error*"
// 	}
// 	hash := hdr.BlockHash()
// 	return hash.String()
// }

// func dbgHashHexFromHdr(hdr *wire.BlockHeader) string {
// 	return hdr.BlockHash().String()
// }

// updateFromBlocks

// // dbg
// fmt.Println("incoming:")
// hash := n.dbgHashHexFromHeaderHex(header)
// prev := n.dbgStringHeaderPrev(header)
// fmt.Printf("hdr hash: %s hdr prev: %s\n", hash, prev)
// // enddbg

// updateFromChunk

// // dbg
// fmt.Println("incoming:")
// for i := 0; i < hdrsRes.Count; i++ {
// 	header := hdrsRes.HexConcat[i*oneHdrLen : (i+1)*oneHdrLen]
// 	hash := n.dbgHashHexFromHeaderHex(header)
// 	prev := n.dbgStringHeaderPrev(header)
// 	fmt.Printf("hdr hash: %s hdr prev: %s\n", hash, prev)
// }
// // enddbg
