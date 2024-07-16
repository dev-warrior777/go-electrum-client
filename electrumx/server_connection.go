// This code is available on the terms of the project LICENSE.md file,
// also available online at https://blueoakcouncil.org/license/1.0.0.

// Package electrum provides a client for an ElectrumX server. Not all methods
// are implemented. For the methods and their request and response types, see
// https://electrumx.readthedocs.io/en/latest/protocol-methods.html.
package electrumx

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/decred/go-socks/socks"
)

// Thanks to Chappjc for the original source code.

// printer is a function with the signature of a logger method.
type printer func(format string, params ...any)

var (
	stderrPrinter = printer(func(format string, params ...any) {
		fmt.Fprintf(os.Stderr, format+"\n", params...)
	})
)

// from electrum code - a ping should be about 50% default server timeout for
// ping which is ~10m .. so should be around 300s with a margin for error.
// Unfortunately many servers have different time outs much shorter than this.
//
// We also do read deadline extension every 10s so do a ping there. Pings have
// a high server anti-ddos session cost of 0.1 though .. so it's a balancing act.
const keepAliveInterval = 10 * time.Second

type serverConn struct {
	conn       net.Conn
	nodeCancel context.CancelCauseFunc
	done       chan struct{}
	addr       string // kept for debug
	debug      printer

	reqID uint64

	// Response handlers per request with id. Closed in the 'listen' func below.
	respHandlers    map[uint64]chan *response // reqID => requestor
	respHandlersMtx sync.Mutex

	// The single scripthash notification channel. The channel will be made on
	// the connectServer call and lasts until connection is terminated. It is
	// closed in the 'listen' func below.
	scripthashNotify    chan *ScripthashStatusResult
	scripthashNotifyMtx sync.Mutex

	// The single headers notification channel. The channel will be made on
	// the connectServer call and lasts until connection is terminated. It is
	// closed in the 'listen' below.
	headersNotify    chan *headersNotifyResult
	headersNotifyMtx sync.Mutex
}

func (sc *serverConn) nextID() uint64 {
	return atomic.AddUint64(&sc.reqID, 1)
}

const newline = byte('\n')

// listen reads from the tcpip stream forwards replies to the response and notification
// channels.
func (sc *serverConn) listen(nodeCtx context.Context) {
	// Only listen should close these channels, and only after the read loop has finished
	// so that requests are passed back to the caller after the connection is terminated.
	defer sc.cancelRequests()        // close the response chans
	defer sc.closeHeadersNotify()    // close the headers notify channel
	defer sc.closeScripthashNotify() // close the scripthash notify channel

	// make a reader with a buffer big enough to handle initial sync download
	// of block headers from ElectrumX -> client in chunks of 2016 headers for
	// btc on each request. Chunks * Header size * safety margin.
	reader := bufio.NewReaderSize(sc.conn, 2016*80*16)

	for {
		if nodeCtx.Err() != nil {
			return
		}

		// read msg chunk from stream
		msg, err := reader.ReadBytes(newline)
		if err != nil {
			if nodeCtx.Err() == nil { // unexpected
				sc.debug("ReadBytes: %v - conn closed\n", err)
			}
			sc.nodeCancel(errServerCanceled)
			return
		}

		var jsonResp response
		err = json.Unmarshal(msg, &jsonResp)
		if err != nil {
			continue
		}

		// sc.debug("[Debug] ", string(msg), "\n[<-Debug]\n\n")

		// Notifications
		if jsonResp.Method != "" {
			var ntfnParams ntfnData // the ntfn payload
			err = json.Unmarshal(msg, &ntfnParams)
			if err != nil {
				sc.debug("notification Unmarshal error: %v", err)
				continue
			}

			if jsonResp.Method == "blockchain.headers.subscribe" {
				fmt.Println()
				fmt.Println("------- debug headers change ----------------------------------------------")
				sc.headersTipChangeNotify(ntfnParams.Params)
				continue
			}

			if jsonResp.Method == "blockchain.scripthash.subscribe" {
				fmt.Println()
				fmt.Println("------- debug -------------------------------------------------------------")
				fmt.Println(" --- blockchain.scripthash.subscribe")
				sc.scripthashStatusNotify(ntfnParams.Params)
				continue
			}
			sc.debug("Received notification for unknown method %s", jsonResp.Method)
			continue
		}

		// Responses
		c := sc.responseChan(jsonResp.ID)
		if c == nil {
			sc.debug("Received response for unknown request ID %d", jsonResp.ID)
			continue
		}
		c <- &jsonResp // buffered and single use => cannot block
	}
}

// keepAlive pushes the stream read deadline further into the future every 10s
// then pings the server.
func (sc *serverConn) keepAlive(nodeCtx context.Context) {
	t := time.NewTicker(keepAliveInterval)
	defer t.Stop()

	for {
		// listen => ReadBytes cannot wait forever.
		newTime := time.Now().Add(keepAliveInterval * 5 / 4)
		err := sc.conn.SetReadDeadline(newTime)
		if err != nil {
			return
		}
		if err = sc.ping(nodeCtx); err != nil {
			return
		}

		select {
		case <-nodeCtx.Done():
			return
		case <-t.C:
		}
	}
}

type connectOpts struct {
	TLSConfig *tls.Config
	TorProxy  string
}

// connectServer connects to the electrumx server at the given address. To close
// the connection and shutdown serverConn cancel the context then wait on the
// channel from Done() to ensure a clean shutdown (connection closed and
// all incoming responses handled).
// There is no automatic reconnection functionality, as the caller should handle
// dropped connections by cycling to a different server.
func connectServer(
	nodeCtx context.Context,
	nodeCancel context.CancelCauseFunc,
	addr string,
	opts *connectOpts) (*serverConn, error) {

	var dial func(nodeCtx context.Context, network, addr string) (net.Conn, error)
	var dialCtx context.Context
	var dialCancel context.CancelFunc

	if opts.TorProxy != "" {
		proxy := &socks.Proxy{
			Addr:         opts.TorProxy,
			TorIsolation: true,
		}
		fmt.Printf("using tor isolation proxy: %s\n - to connect to %s \n", proxy.Addr, addr)
		dial = proxy.DialContext
		dialCtx, dialCancel = context.WithTimeout(nodeCtx, 20*time.Second)
		defer dialCancel()
	} else {
		dial = new(net.Dialer).DialContext
		dialCtx, dialCancel = context.WithTimeout(nodeCtx, 5*time.Second)
		defer dialCancel()
	}

	conn, err := dial(dialCtx, "tcp", addr)
	if err != nil {
		fmt.Printf("dial - %v\n", err)
		return nil, err
	}

	if opts.TLSConfig != nil {
		conn = tls.Client(conn, opts.TLSConfig)
		err = conn.(*tls.Conn).HandshakeContext(nodeCtx)
		if err != nil {
			conn.Close()
			return nil, err
		}
	}

	sc := &serverConn{
		conn:         conn,
		nodeCancel:   nodeCancel,
		done:         make(chan struct{}),
		addr:         addr,
		debug:        stderrPrinter,
		respHandlers: make(map[uint64]chan *response),
		// 128 bytes - unbuffered because we have a queue downstream
		scripthashNotify: make(chan *ScripthashStatusResult),
		// 168 bytes - unbuffered because we have a queue downstream
		headersNotify: make(chan *headersNotifyResult),
	}

	go sc.listen(nodeCtx)
	go sc.keepAlive(nodeCtx)
	go func() {
		<-nodeCtx.Done()
		cause := context.Cause(nodeCtx)
		sc.debug("nodeCtx.Done in connectServer for %s - cause %v\n", sc.addr, cause)
		conn.Close()
		close(sc.done)
	}()

	return sc, nil
}

// Done returns a channel that is closed when serverConn is *fully* shut down.
func (sc *serverConn) Done() <-chan struct{} {
	return sc.done
}

func (sc *serverConn) send(msg []byte) error {
	err := sc.conn.SetWriteDeadline(time.Now().Add(7 * time.Second))
	if err != nil {
		return err
	}
	_, err = sc.conn.Write(msg)
	return err
}

func (sc *serverConn) registerRequest(id uint64) chan *response {
	c := make(chan *response, 1)
	sc.respHandlersMtx.Lock()
	sc.respHandlers[id] = c
	sc.respHandlersMtx.Unlock()
	return c
}

func (sc *serverConn) responseChan(id uint64) chan *response {
	sc.respHandlersMtx.Lock()
	defer sc.respHandlersMtx.Unlock()
	c := sc.respHandlers[id]
	delete(sc.respHandlers, id)
	return c
}

// cancelRequests deletes all response handlers from the respHandlers map and
// closes all of the channels. This method MUST be called the listen thread.
func (sc *serverConn) cancelRequests() {
	sc.respHandlersMtx.Lock()
	defer sc.respHandlersMtx.Unlock()
	for id, c := range sc.respHandlers {
		close(c)
		delete(sc.respHandlers, id)
	}
}

// scripthashStatusNotify is called from the listen thread when a
// scripthash status notification has been received. The raw bytes
// are 2 non-json strings.
//
// Incoming data from the server:
// raw '\[s1, s2\]'
//
// Which we decode into:
// statusResult [ScriptHash, Status]
func (sc *serverConn) scripthashStatusNotify(raw json.RawMessage) {
	var strs [2]string
	if err := json.Unmarshal(raw, &strs); err == nil && len(strs) == 2 {
		statusResult := ScripthashStatusResult{
			Scripthash: strs[0],
			Status:     strs[1],
		}
		sc.scripthashNotifyMtx.Lock()
		defer sc.scripthashNotifyMtx.Unlock()
		sc.scripthashNotify <- &statusResult
	} else {
		sc.debug("Scripthash Status Notify\nError: %v\nRaw: %s\n", err, string(raw))
	}
}

// closeScripthashNotify closes the scripthash subscription notify channel.
func (sc *serverConn) closeScripthashNotify() {
	sc.scripthashNotifyMtx.Lock()
	defer sc.scripthashNotifyMtx.Unlock()
	close(sc.scripthashNotify)
}

// headersTipChangeNotify is called from the listen thread when a header
// tip change notification has been received.
//
// Incoming data from the server:
// raw '\[\{...\}\{...\}   ...   \{...\}\]'
//
// Which we decode into:
// headersResults [{Height,Hex}{Height,Hex}...{Height,Hex}]
func (sc *serverConn) headersTipChangeNotify(raw json.RawMessage) {
	var headersResults []*headersNotifyResult
	if err := json.Unmarshal(raw, &headersResults); err == nil {
		sc.headersNotifyMtx.Lock()
		defer sc.headersNotifyMtx.Unlock()
		for _, r := range headersResults {
			sc.headersNotify <- r
		}
	} else {
		sc.debug("Headers Notify\nError: %v\nRaw: %s\n", err, string(raw))
	}
}

// closeScripthashNotify closes the scripthash subscription notify channel.
func (sc *serverConn) closeHeadersNotify() {
	sc.headersNotifyMtx.Lock()
	defer sc.headersNotifyMtx.Unlock()
	close(sc.headersNotify)
}

// request performs a request to the remote server for the given method using
// the provided arguments, which may either be positional (e.g.
// []interface{arg1, arg2}), named (any struct), or nil if there are no
// arguments. args may not be any other basic type. The the response does not
// include an error, the result will be unmarshalled into result, unless the
// provided result is nil in which case the response payload will be ignored.
func (sc *serverConn) request(nodeCtx context.Context, method string, args any, result any) error {
	id := sc.nextID()
	reqMsg, err := prepareRequest(id, method, args)
	if err != nil {
		return err
	}
	reqMsg = append(reqMsg, newline)

	c := sc.registerRequest(id)

	if err = sc.send(reqMsg); err != nil {
		sc.nodeCancel(errServerCanceled)
		return err
	}

	var resp *response
	select {
	case <-nodeCtx.Done():
		return nodeCtx.Err()
	case resp = <-c:
	}

	if resp == nil {
		return errors.New("response channel closed")
	}

	if resp.Error != nil {
		return resp.Error
	}

	if result != nil {
		return json.Unmarshal(resp.Result, result)
	}
	return nil
}

// ----------------------------------------------------------------------------
// Server API
// ----------------------------------------------------------------------------

// ping the remote server.
func (sc *serverConn) ping(nodeCtx context.Context) error {
	return sc.request(nodeCtx, "server.ping", nil, nil)
}

// serverVersion returns the server's software version and electrumx protocol
// of the connected server
func (sc *serverConn) serverVersion(nodeCtx context.Context, client, proto string) ([]string, error) {
	var vers []string
	err := sc.request(nodeCtx, "server.version", positional{client, proto}, &vers)
	if err != nil {
		return nil, err
	}
	if len(vers) != 2 {
		return nil, fmt.Errorf("unexpected version response: %v", vers)
	}
	return vers, nil
}

// serverFeatures represents the result of a server features request.
type serverFeatures struct {
	Genesis  string                       `json:"genesis_hash"`
	Hosts    map[string]map[string]uint32 `json:"hosts"` // e.g. {"host.com": {"tcp_port": 51001, "ssl_port": 51002}}, may be unset!
	ProtoMax string                       `json:"protocol_max"`
	ProtoMin string                       `json:"protocol_min"`
	Pruning  any                          `json:"pruning,omitempty"`  // supposedly an integer, but maybe a string or even JSON null
	Version  string                       `json:"server_version"`     // server software version, not proto
	HashFunc string                       `json:"hash_function"`      // e.g. sha256
	Services []string                     `json:"services,omitempty"` // e.g. ["tcp://host.com:51001", "ssl://host.com:51002"]
}

// features requests the features claimed by the server.
func (sc *serverConn) features(nodeCtx context.Context) (*serverFeatures, error) {
	var feats serverFeatures
	err := sc.request(nodeCtx, "server.features", nil, &feats)
	if err != nil {
		return nil, err
	}
	return &feats, nil
}

// peersResult represents the results of a peers server request. We further break
// this info down in network_servers.go as this struct is awkward and we want to
// persist some of it in a json file like electrum does.
type peersResult struct {
	Addr  string // IP address or .onion name
	Host  string
	Feats []string
}

// serverPeers requests the known peers from a server (other servers). Occasionaly
// a testnet server such as  "testnet.qtornado.com:51002" doesn't send peers. But
// is still useful as a non-leader.
func (sc *serverConn) serverPeers(nodeCtx context.Context) ([]*peersResult, error) {
	// Note that the Electrum exchange wallet type does not  use this
	// method since it follows the Electrum wallet server peer or one of the
	// wallets other servers. See (*electrumWallet).connect and
	// (*WalletClient).GetServers. We might wish to in the future though.

	// [["ip", "host", ["featA", "featB", ...]], ...]
	// [][]any{string, string, []any{string, ...}}
	var resp [][]any
	err := sc.request(nodeCtx, "server.peers.subscribe", nil, &resp) // not really a subscription!
	if err != nil {
		return nil, err
	}

	peers := make([]*peersResult, 0, len(resp))
	for _, peer := range resp {
		if len(peer) != 3 {
			sc.debug("bad peer data: %v (%T)", peer, peer)
			continue
		}
		addr, ok := peer[0].(string)
		if !ok {
			sc.debug("bad peer IP data: %v (%T)", peer[0], peer[0])
			continue
		}
		host, ok := peer[1].(string)
		if !ok {
			sc.debug("bad peer hostname: %v (%T)", peer[1], peer[1])
			continue
		}
		featsI, ok := peer[2].([]any)
		if !ok {
			sc.debug("bad peer feature data: %v (%T)", peer[2], peer[2])
			continue
		}
		feats := make([]string, len(featsI))
		for i, featI := range featsI {
			feat, ok := featI.(string)
			if !ok {
				sc.debug("bad peer feature data: %v (%T)", featI, featI)
				continue
			}
			feats[i] = feat
		}
		peers = append(peers, &peersResult{
			Addr:  addr,
			Host:  host,
			Feats: feats,
		})
	}
	return peers, nil
}

// sigScript represents the signature script in a Vin returned by a transaction
// request.
type sigScript struct {
	Asm string `json:"asm"` // this is not the sigScript you're looking for ;-)
	Hex string `json:"hex"`
}

// vin represents a transaction input in a requested transaction.
type vin struct {
	TxID      string     `json:"txid"`
	Vout      uint32     `json:"vout"`
	SigScript *sigScript `json:"scriptsig"`
	Witness   []string   `json:"txinwitness,omitempty"`
	Sequence  uint32     `json:"sequence"`
	Coinbase  string     `json:"coinbase,omitempty"`
}

// pkScript represents the pkScript/scriptPubKey of a transaction output
type pkScript struct {
	Asm       string   `json:"asm"`
	Hex       string   `json:"hex"`
	ReqSigs   uint32   `json:"reqsigs"`
	Type      string   `json:"type"`
	Addresses []string `json:"addresses,omitempty"`
}

// vout represents a transaction output in a requested transaction.
type vout struct {
	Value    float64  `json:"value"`
	N        uint32   `json:"n"`
	PkScript pkScript `json:"scriptpubkey"`
}

// GetTransactionResult is the data returned by a transaction request. It is
// exported to Client.
type GetTransactionResult struct {
	TxID string `json:"txid"`
	// Hash          string `json:"hash"` // ??? don't use, not always the txid! witness not stripped?
	Version       uint32 `json:"version"`
	Size          uint32 `json:"size"`
	VSize         uint32 `json:"vsize"`
	Weight        uint32 `json:"weight"`
	LockTime      uint32 `json:"locktime"`
	Hex           string `json:"hex"`
	Vin           []vin  `json:"vin"`
	Vout          []vout `json:"vout"`
	BlockHash     string `json:"blockhash,omitempty"`
	Confirmations int32  `json:"confirmations,omitempty"` // probably uint32 ok because it seems to be omitted, but could be -1?
	Time          int64  `json:"time,omitempty"`
	BlockTime     int64  `json:"blocktime,omitempty"` // same as Time?
	// Merkel // proto 1.5+ // consider upgrading proto
}

// getTransaction requests a transaction with verbose output. Some servers such
// as Blockstream do not support this to save on the heavy resources used.
func (sc *serverConn) getTransaction(nodeCtx context.Context, txid string) (*GetTransactionResult, error) {
	verbose := true
	// verbose result
	var resp GetTransactionResult
	err := sc.request(nodeCtx, "blockchain.transaction.get", positional{txid, verbose}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// getRawTransaction requests a transaction as raw bytes.
func (sc *serverConn) getRawTransaction(nodeCtx context.Context, txid string) (string, error) {
	// non verbose result as a hex string of the raw transaction
	var resp string
	err := sc.request(nodeCtx, "blockchain.transaction.get", positional{txid, false}, &resp)
	if err != nil {
		return "", err
	}
	return resp, nil
}

// ////////////////////////////////////////////////////////////////////////////
// block headers methods
// /////////////////////

// blockHeader requests the block header at the given height, returning
// hex encoded serialized header.
func (sc *serverConn) blockHeader(nodeCtx context.Context, height uint32) (string, error) {
	var resp string
	err := sc.request(nodeCtx, "blockchain.block.header", positional{height}, &resp)
	if err != nil {
		return "", err
	}
	return resp, nil
}

// getBlockHeadersResult represents the result of a batch request for block
// headers via the block.headers method. The serialized block headers are
// concatenated in the HexConcat field, which contains Count headers.
type getBlockHeadersResult struct {
	Count     int    `json:"count"`
	HexConcat string `json:"hex"`
	Max       int64  `json:"max"`
}

// blockHeaders requests a batch of block headers beginning at the given height.
// The sever may respond with a different number of headers, so the caller
// should check the Count field of the result.
func (sc *serverConn) blockHeaders(nodeCtx context.Context, startHeight int64, count int) (*getBlockHeadersResult, error) {
	var resp getBlockHeadersResult
	err := sc.request(nodeCtx, "blockchain.block.headers", positional{startHeight, count}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// headersNotifyResult is the contents of a block header notification.
type headersNotifyResult struct {
	Height int64  `json:"height"`
	Hex    string `json:"hex"`
}

// getHeadersNotify returns this connection owned recv channel for headers
// tip change notifications.
func (sc *serverConn) getHeadersNotify() chan *headersNotifyResult {
	return sc.headersNotify
}

// subscribeHeaders subscribes for block header notifications. There is no
// guarantee that we will be notified of all new blocks when the electrumx
// server is busy processing blocks.
func (sc *serverConn) subscribeHeaders(nodeCtx context.Context) (*headersNotifyResult, error) {
	const method = "blockchain.headers.subscribe"

	var resp headersNotifyResult
	err := sc.request(nodeCtx, method, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// ////////////////////////////////////////////////////////////////////////////
// scripthash methods (exported) to Client
// ///////////////////////////////////////

// ScripthashStatusResult is the contents of a scripthash notification.
// Raw bytes with no json key names or json [] {} delimiters
type ScripthashStatusResult struct {
	Scripthash string // 32 byte scripthash - the id of the watched address
	Status     string // 32 byte sha256 hash of entire history to date or null
}

// GetScripthashNotify returns this connection owned recv channel for scripthash
// status change notifications.
func (sc *serverConn) GetScripthashNotify() chan *ScripthashStatusResult {
	return sc.scripthashNotify
}

// SubscribeScripthash subscribes for notifications of changes for an address
// (scripthash) in wallet. We send the electrum 'scripthash' of the address
// rather than the base58 encoded string. See also client_wallet.go.
func (sc *serverConn) SubscribeScripthash(nodeCtx context.Context, scripthash string) (*ScripthashStatusResult, error) {
	const method = "blockchain.scripthash.subscribe"

	var status string // no json - sha256 of address history expected, as hex string
	err := sc.request(nodeCtx, method, positional{scripthash}, &status)
	if err != nil {
		return nil, err
	}

	statusResult := ScripthashStatusResult{
		Scripthash: scripthash,
		Status:     status,
	}

	return &statusResult, nil
}

// UnsubscribeScripthash unsubscribes from a script hash, preventing future
// status change notifications.
func (sc *serverConn) UnsubscribeScripthash(nodeCtx context.Context, scripthash string) {
	const method = "blockchain.scripthash.unsubscribe"

	var resp string
	err := sc.request(nodeCtx, method, positional{scripthash}, &resp)
	if err != nil {
		sc.debug("UnsubscribeScripthash: %v\n", err)
	}
}

type History struct {
	Height int64  `json:"height"`
	TxHash string `json:"tx_hash"`
	Fee    int    `json:"fee,omitempty"` // satoshis; iff in mempool
}

type HistoryResult []History

// GetHistory gets a list of [{height, txid and fee (only mempool)},...] for the
// scripthash of an address.
func (sc *serverConn) GetHistory(nodeCtx context.Context, scripthash string) (HistoryResult, error) {
	var resp HistoryResult
	err := sc.request(nodeCtx, "blockchain.scripthash.get_history", positional{scripthash}, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type ListUnspent struct {
	Height int64  `json:"height"`
	TxPos  int64  `json:"tx_pos"`
	TxHash string `json:"tx_hash"`
	Value  int64  `json:"value"` // satoshis
}

type ListUnspentResult []ListUnspent

// GetListUnspent gets a list of [{height, txid tx_pos and value},...] for the
// scripthash of an address.
func (sc *serverConn) GetListUnspent(nodeCtx context.Context, scripthash string) (ListUnspentResult, error) {
	var resp ListUnspentResult
	err := sc.request(nodeCtx, "blockchain.scripthash.listunspent", positional{scripthash}, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ////////////////////////////////////////////////////////////////////////////
// Other wallet methods (exported to Client)
// /////////////////////////////////////////

// Broadcast broadcasts a raw tx as a hexadecimal string to the network. The tx
// hash is returned as a hexadecimal string.
func (sc *serverConn) Broadcast(nodeCtx context.Context, rawTx string) (string, error) {
	var resp string
	err := sc.request(nodeCtx, "blockchain.transaction.broadcast", positional{rawTx}, &resp)
	if err != nil {
		return "", err
	}
	return resp, nil
}

// Estimated transaction fee in coin units per kilobyte, as a floating point number string.
// If the daemon does not have enough information to make an estimate, the integer -1
// is returned.
func (sc *serverConn) EstimateFee(nodeCtx context.Context, confTarget int64) (int64, error) {
	var resp float64
	err := sc.request(nodeCtx, "blockchain.estimatefee", positional{confTarget}, &resp)
	if err != nil {
		return 0, err
	}
	if resp == -1 {
		return -1, errors.New("server cannot estimate a feerate")
	}
	return int64(resp * 1e8), nil
}
