package raft

import (
	"context"
	"sync"
	"time"
)

type InmemTransport struct {
	mu       sync.RWMutex
	peers    map[int]*InmemTransport
	me       int
	peerCount int
	handler  RPCHandler
	
	timeout time.Duration
	closed  bool
	closeCh chan struct{}
}

func NewInmemTransport(peerID int, peerCount int) *InmemTransport {
	return &InmemTransport{
		peers:    make(map[int]*InmemTransport),
		me:       peerID,
		peerCount: peerCount,
		timeout:  100 * time.Millisecond,
		closeCh:  make(chan struct{}),
	}
}

func (t *InmemTransport) Connect(peerID int, peer *InmemTransport) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.peers[peerID] = peer
}

func (t *InmemTransport) GetPeerCount() int {
	return t.peerCount
}

func (t *InmemTransport) GetPeerID() int {
	return t.me
}

func (t *InmemTransport) GetEndpoint(peerID int) RPCEndpoint {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if peerID == t.me {
		return t
	}
	return t.peers[peerID]
}

func (t *InmemTransport) RegisterHandler(handler RPCHandler) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.handler = handler
}

func (t *InmemTransport) Start() error {
	return nil
}

func (t *InmemTransport) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return
	}
	t.closed = true
	close(t.closeCh)
}

func (t *InmemTransport) SendRequestVote(ctx context.Context, req *RequestVoteRequest) (*RequestVoteResponse, error) {
	t.mu.RLock()
	handler := t.handler
	t.mu.RUnlock()
	
	if handler == nil {
		return nil, context.DeadlineExceeded
	}
	
	resp := handler.HandleRequestVote(req)
	return resp, nil
}

func (t *InmemTransport) SendAppendEntries(ctx context.Context, req *AppendEntriesRequest) (*AppendEntriesResponse, error) {
	t.mu.RLock()
	handler := t.handler
	t.mu.RUnlock()
	
	if handler == nil {
		return nil, context.DeadlineExceeded
	}
	
	resp := handler.HandleAppendEntries(req)
	return resp, nil
}

func (t *InmemTransport) SendInstallSnapshot(ctx context.Context, req *InstallSnapshotRequest) (*InstallSnapshotResponse, error) {
	t.mu.RLock()
	handler := t.handler
	t.mu.RUnlock()
	
	if handler == nil {
		return nil, context.DeadlineExceeded
	}
	
	resp := handler.HandleInstallSnapshot(req)
	return resp, nil
}

type InmemNetwork struct {
	mu       sync.Mutex
	trans    map[int]*InmemTransport
	handlers map[int]RPCHandler
	peerCount int
}

func NewInmemNetwork(peerCount int) *InmemNetwork {
	return &InmemNetwork{
		trans:     make(map[int]*InmemTransport),
		handlers:  make(map[int]RPCHandler),
		peerCount: peerCount,
	}
}

func (n *InmemNetwork) CreateTransport(peerID int) *InmemTransport {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	t := NewInmemTransport(peerID, n.peerCount)
	n.trans[peerID] = t
	
	for id, other := range n.trans {
		if id != peerID {
			t.Connect(id, other)
			other.Connect(peerID, t)
		}
	}
	
	return t
}

func (n *InmemNetwork) RegisterHandler(peerID int, handler RPCHandler) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.handlers[peerID] = handler
	if t, ok := n.trans[peerID]; ok {
		t.RegisterHandler(handler)
	}
}

func (n *InmemNetwork) GetTransport(peerID int) *InmemTransport {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.trans[peerID]
}
