package raft

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestInmemTransport_RequestVote(t *testing.T) {
	network := NewInmemNetwork(2)
	
	transport1 := network.CreateTransport(1)
	transport2 := network.CreateTransport(2)
	
	var handler2Mu sync.Mutex
	var receivedReq *RequestVoteRequest
	handler2 := &mockHandler{
		onRequestVote: func(req *RequestVoteRequest) *RequestVoteResponse {
			handler2Mu.Lock()
			defer handler2Mu.Unlock()
			receivedReq = req
			return &RequestVoteResponse{Term: 2, VoteGranted: true}
		},
	}
	transport2.RegisterHandler(handler2)
	
	req := &RequestVoteRequest{
		Term:         1,
		CandidateId:  1,
		LastLogIndex: 10,
		LastLogTerm:  1,
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	
	endpoint := transport1.GetEndpoint(2)
	resp, err := endpoint.SendRequestVote(ctx, req)
	
	if err != nil {
		t.Fatalf("SendRequestVote failed: %v", err)
	}
	
	if resp.Term != 2 {
		t.Errorf("expected term 2, got %d", resp.Term)
	}
	
	if !resp.VoteGranted {
		t.Errorf("expected vote granted")
	}
	
	handler2Mu.Lock()
	if receivedReq == nil {
		t.Error("handler did not receive request")
	} else if receivedReq.CandidateId != 1 {
		t.Errorf("expected candidate id 1, got %d", receivedReq.CandidateId)
	}
	handler2Mu.Unlock()
}

func TestInmemTransport_AppendEntries(t *testing.T) {
	network := NewInmemNetwork(2)
	
	transport1 := network.CreateTransport(1)
	transport2 := network.CreateTransport(2)
	
	handler2 := &mockHandler{
		onAppendEntries: func(req *AppendEntriesRequest) *AppendEntriesResponse {
			return &AppendEntriesResponse{Term: 2, Success: true}
		},
	}
	transport2.RegisterHandler(handler2)
	
	req := &AppendEntriesRequest{
		Term:         2,
		LeaderId:     1,
		PrevLogIndex: 5,
		PrevLogTerm:  1,
		Entries: []LogEntry{
			{Term: 2, CommandValid: true, Command: "test"},
		},
		LeaderCommit: 5,
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	
	endpoint := transport1.GetEndpoint(2)
	resp, err := endpoint.SendAppendEntries(ctx, req)
	
	if err != nil {
		t.Fatalf("SendAppendEntries failed: %v", err)
	}
	
	if !resp.Success {
		t.Errorf("expected success")
	}
}

func TestInmemTransport_InstallSnapshot(t *testing.T) {
	network := NewInmemNetwork(2)
	
	transport1 := network.CreateTransport(1)
	transport2 := network.CreateTransport(2)
	
	handler2 := &mockHandler{
		onInstallSnapshot: func(req *InstallSnapshotRequest) *InstallSnapshotResponse {
			return &InstallSnapshotResponse{Term: 2}
		},
	}
	transport2.RegisterHandler(handler2)
	
	req := &InstallSnapshotRequest{
		Term:              2,
		LeaderId:          1,
		LastIncludedIndex: 100,
		LastIncludedTerm:  2,
		Snapshot:          []byte("snapshot-data"),
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	
	endpoint := transport1.GetEndpoint(2)
	resp, err := endpoint.SendInstallSnapshot(ctx, req)
	
	if err != nil {
		t.Fatalf("SendInstallSnapshot failed: %v", err)
	}
	
	if resp.Term != 2 {
		t.Errorf("expected term 2, got %d", resp.Term)
	}
}

func TestInmemTransport_GetPeerCount(t *testing.T) {
	network := NewInmemNetwork(3)
	
	transport1 := network.CreateTransport(1)
	_ = network.CreateTransport(2)
	_ = network.CreateTransport(3)
	
	if transport1.GetPeerCount() != 3 {
		t.Errorf("expected 3 peers, got %d", transport1.GetPeerCount())
	}
	
	if transport1.GetPeerID() != 1 {
		t.Errorf("expected peer id 1, got %d", transport1.GetPeerID())
	}
}

func TestInmemTransport_NoHandler(t *testing.T) {
	network := NewInmemNetwork(2)
	
	transport1 := network.CreateTransport(1)
	_ = network.CreateTransport(2)
	
	req := &RequestVoteRequest{
		Term:        1,
		CandidateId: 1,
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	
	endpoint := transport1.GetEndpoint(2)
	_, err := endpoint.SendRequestVote(ctx, req)
	
	if err == nil {
		t.Error("expected error when no handler registered")
	}
}

type mockHandler struct {
	onRequestVote     func(req *RequestVoteRequest) *RequestVoteResponse
	onAppendEntries   func(req *AppendEntriesRequest) *AppendEntriesResponse
	onInstallSnapshot func(req *InstallSnapshotRequest) *InstallSnapshotResponse
}

func (h *mockHandler) HandleRequestVote(req *RequestVoteRequest) *RequestVoteResponse {
	if h.onRequestVote != nil {
		return h.onRequestVote(req)
	}
	return &RequestVoteResponse{}
}

func (h *mockHandler) HandleAppendEntries(req *AppendEntriesRequest) *AppendEntriesResponse {
	if h.onAppendEntries != nil {
		return h.onAppendEntries(req)
	}
	return &AppendEntriesResponse{}
}

func (h *mockHandler) HandleInstallSnapshot(req *InstallSnapshotRequest) *InstallSnapshotResponse {
	if h.onInstallSnapshot != nil {
		return h.onInstallSnapshot(req)
	}
	return &InstallSnapshotResponse{}
}
