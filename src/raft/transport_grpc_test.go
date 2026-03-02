package raft

import (
	"context"
	"testing"
	"time"
)

func TestGrpcTransport_RequestVote(t *testing.T) {
	peers := []string{
		"127.0.0.1:50051",
		"127.0.0.1:50052",
	}
	
	transport1 := NewGrpcTransport(&TransportConfig{
		Peers:      peers,
		PeerID:     0,
		ListenAddr: peers[0],
	})
	
	transport2 := NewGrpcTransport(&TransportConfig{
		Peers:      peers,
		PeerID:     1,
		ListenAddr: peers[1],
	})
	
	handler2 := &mockHandler{
		onRequestVote: func(req *RequestVoteRequest) *RequestVoteResponse {
			return &RequestVoteResponse{Term: 2, VoteGranted: true}
		},
	}
	transport2.RegisterHandler(handler2)
	
	if err := transport1.Start(); err != nil {
		t.Fatalf("transport1.Start failed: %v", err)
	}
	defer transport1.Stop()
	
	if err := transport2.Start(); err != nil {
		t.Fatalf("transport2.Start failed: %v", err)
	}
	defer transport2.Stop()
	
	time.Sleep(500 * time.Millisecond)
	
	endpoint := transport1.GetEndpoint(1)
	if endpoint == nil {
		t.Fatal("endpoint should not be nil")
	}
	
	req := &RequestVoteRequest{
		Term:         1,
		CandidateId:  0,
		LastLogIndex: 10,
		LastLogTerm:  1,
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	resp, err := endpoint.SendRequestVote(ctx, req)
	if err != nil {
		t.Fatalf("SendRequestVote failed: %v", err)
	}
	
	if !resp.VoteGranted {
		t.Error("expected vote granted")
	}
}

func TestGrpcTransport_AppendEntries(t *testing.T) {
	peers := []string{
		"127.0.0.1:50061",
		"127.0.0.1:50062",
	}
	
	transport1 := NewGrpcTransport(&TransportConfig{
		Peers:      peers,
		PeerID:     0,
		ListenAddr: peers[0],
	})
	
	transport2 := NewGrpcTransport(&TransportConfig{
		Peers:      peers,
		PeerID:     1,
		ListenAddr: peers[1],
	})
	
	handler2 := &mockHandler{
		onAppendEntries: func(req *AppendEntriesRequest) *AppendEntriesResponse {
			return &AppendEntriesResponse{Term: 2, Success: true}
		},
	}
	transport2.RegisterHandler(handler2)
	
	if err := transport1.Start(); err != nil {
		t.Fatalf("transport1.Start failed: %v", err)
	}
	defer transport1.Stop()
	
	if err := transport2.Start(); err != nil {
		t.Fatalf("transport2.Start failed: %v", err)
	}
	defer transport2.Stop()
	
	time.Sleep(500 * time.Millisecond)
	
	endpoint := transport1.GetEndpoint(1)
	if endpoint == nil {
		t.Fatal("endpoint should not be nil")
	}
	
	req := &AppendEntriesRequest{
		Term:         2,
		LeaderId:     0,
		PrevLogIndex: 5,
		PrevLogTerm:  1,
		Entries: []LogEntry{
			{Term: 2, CommandValid: true, Command: []byte("test")},
		},
		LeaderCommit: 5,
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	resp, err := endpoint.SendAppendEntries(ctx, req)
	if err != nil {
		t.Fatalf("SendAppendEntries failed: %v", err)
	}
	
	if !resp.Success {
		t.Error("expected success")
	}
}

func TestGrpcTransport_GetPeerCount(t *testing.T) {
	peers := []string{
		"127.0.0.1:50071",
		"127.0.0.1:50072",
		"127.0.0.1:50073",
	}
	
	transport := NewGrpcTransport(&TransportConfig{
		Peers:      peers,
		PeerID:     0,
		ListenAddr: peers[0],
	})
	
	if transport.GetPeerCount() != 3 {
		t.Errorf("expected 3 peers, got %d", transport.GetPeerCount())
	}
	
	if transport.GetPeerID() != 0 {
		t.Errorf("expected peer id 0, got %d", transport.GetPeerID())
	}
}
