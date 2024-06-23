package electrumx

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"unicode/utf8"
)

///////////////////////////////////////////////////
// ELECTRUMX ANTI DDOS Resource Usage Calculator //
///////////////////////////////////////////////////

// session.py from electrum client similar to session.py in electrumX
//
// # Multiply this by bandwidth bytes used to get resource usage cost
// bw_cost_per_byte = 1 / 100000
// # If cost is over this requests begin to get delayed and concurrency is reduced
// cost_soft_limit = 2000
// # If cost is over this the session is closed
// cost_hard_limit = 10000
// # Resource usage is reduced by this every second
// cost_decay_per_sec = cost_hard_limit / 3600
// # Request delay ranges from 0 to this between cost_soft_limit and cost_hard_limit
// cost_sleep = 2.0
// # Base cost of an error.  Errors that took resources to discover incur additional costs
// error_base_cost = 100.0
// # Initial number of requests that can be concurrently processed
// initial_concurrent = 20
// # Send a "server busy" error if processing a request takes longer than this seconds
// processing_timeout = 30.0
// # Force-close a connection if its socket send buffer stays full this long
// max_send_delay = 20.0

// Each message to and each reply from the server has a cost and as costs rise
// the server will start to throttle the client responses at cost_soft_limit.
// It will increasingly delay responses until cost_hard_limit is reached at
// which point it will disconnect from the tcpip client.
// In mitigation the cost is reduced for by cost_decay_per_sec for periods when
// it is not processing a message. Even errors and database lookups have a cost,
// as do the number of concurrent messages from the same client address.
//
// There is no way to ask the server how much a session has cost at any point in
// time so a client has to calculate the cost itself to avoid being throttled
// then disconnected. Worse a server can set different values for the limits above.
//
// Here we make a simplified session cost calculator based only data bytes sent and
// received with some concurrent calculation of the negative cost decay so that we
// can have some warning and terminate this session and start a new node with a
// new connection/session.
//
// This is just a naive PoC that will have to be better implemented.

const (
	COST_SOFT_LIMIT    = 2000.00
	COST_HARD_LIMIT    = 10000.00
	BW_COST_PER_BYTE   = 1.0 / 100000
	COST_DECAY_PER_SEC = (COST_HARD_LIMIT / 3600.00)
	// Adjust frequency of cost reductions
	TuningFactor = 10
)

type session struct {
	cost    float32
	costMtx sync.Mutex
}

func newSession() *session {
	return &session{cost: float32(0)}
}

func (s *session) start(nodeCtx context.Context) {
	go s.runCostDecayLoop(nodeCtx)
}

func (s *session) runCostDecayLoop(nodeCtx context.Context) {
	var tune int = 0
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-nodeCtx.Done():
			fmt.Println("nodeCtx.Done in runCostDecayLoop - exiting thread")
			return
		case <-t.C:
			if tune%TuningFactor == 0 {
				s.reduceCost()
			}
			tune++
		}
	}
}

func (s *session) reduceCost() {
	s.costMtx.Lock()
	defer s.costMtx.Unlock()
	s.cost -= COST_DECAY_PER_SEC
}

func (s *session) bumpCost(incurred float32) {
	s.costMtx.Lock()
	defer s.costMtx.Unlock()
	s.cost += incurred
	fmt.Printf("incurred: %f, total-cost: %f\n", incurred, s.cost)
}

func (s *session) bumpCostBytes(numBytes int) {
	incurred := float32(numBytes) * BW_COST_PER_BYTE
	s.bumpCost(incurred)
}

func (s *session) bumpCostString(str string) {
	s.bumpCostBytes(utf8.RuneCountInString(str))
}

func (s *session) bumpCostStruct(v any) {
	ba, _ := json.Marshal(v)
	s.bumpCostBytes(len(ba))
}

func (s *session) bumpCostError() {
	// errors are punished
	s.bumpCost(100.0)
}
