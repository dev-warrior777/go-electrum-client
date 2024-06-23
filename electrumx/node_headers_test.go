package electrumx

import "testing"

func TestNode_connectTip(t *testing.T) {
	type fields struct {
		state               nodeState
		serverAddr          string
		connectOpts         *ConnectOpts
		server              *Server
		leader              bool
		syncingHeaders      bool
		networkHeaders      *Headers
		rcvTipChangeNotify  chan int64
		rcvScriptHashNotify chan *ScripthashStatusResult
	}
	type args struct {
		serverHeader string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{
				state:               tt.fields.state,
				serverAddr:          tt.fields.serverAddr,
				connectOpts:         tt.fields.connectOpts,
				server:              tt.fields.server,
				leader:              tt.fields.leader,
				syncingHeaders:      tt.fields.syncingHeaders,
				networkHeaders:      tt.fields.networkHeaders,
				rcvTipChangeNotify:  tt.fields.rcvTipChangeNotify,
				rcvScriptHashNotify: tt.fields.rcvScriptHashNotify,
			}
			if got := n.connectTip(tt.args.serverHeader); got != tt.want {
				t.Errorf("Node.connectTip() = %v, want %v", got, tt.want)
			}
		})
	}
}
