package raft

import (
	"os"
	"testing"
	"time"
)

func TestNode_GetState(t *testing.T) {
	network := NewInmemNetwork(1)
	transport := network.CreateTransport(0)
	storage := NewMemoryRaftStorage()
	applyCh := make(chan ApplyMsg, 100)
	
	config := DefaultConfig()
	node := NewNode(config, transport, storage, applyCh)
	defer node.Kill()
	
	term, isLeader := node.GetState()
	
	if term < 1 {
		t.Errorf("expected term >= 1, got %d", term)
	}
	
	if isLeader {
		t.Error("newly created node should not be leader")
	}
}

func TestNode_Start_NotLeader(t *testing.T) {
	network := NewInmemNetwork(1)
	transport := network.CreateTransport(0)
	storage := NewMemoryRaftStorage()
	applyCh := make(chan ApplyMsg, 100)
	
	config := DefaultConfig()
	node := NewNode(config, transport, storage, applyCh)
	defer node.Kill()
	
	index, term, isLeader := node.Start("test-command")
	
	if isLeader {
		t.Error("should not be leader")
	}
	if index != 0 {
		t.Errorf("expected index 0, got %d", index)
	}
	if term != 0 {
		t.Errorf("expected term 0, got %d", term)
	}
}

func TestNode_HandleRequestVote(t *testing.T) {
	network := NewInmemNetwork(1)
	transport := network.CreateTransport(0)
	storage := NewMemoryRaftStorage()
	applyCh := make(chan ApplyMsg, 100)
	
	config := DefaultConfig()
	node := NewNode(config, transport, storage, applyCh)
	defer node.Kill()
	
	req := &RequestVoteRequest{
		Term:         2,
		CandidateId:  1,
		LastLogIndex: 0,
		LastLogTerm:  0,
	}
	
	resp := node.HandleRequestVote(req)
	
	if !resp.VoteGranted {
		t.Error("expected vote to be granted")
	}
	
	if resp.Term != 2 {
		t.Errorf("expected term 2, got %d", resp.Term)
	}
}

func TestNode_HandleRequestVote_RejectLowerTerm(t *testing.T) {
	network := NewInmemNetwork(1)
	transport := network.CreateTransport(0)
	storage := NewMemoryRaftStorage()
	applyCh := make(chan ApplyMsg, 100)
	
	config := DefaultConfig()
	node := NewNode(config, transport, storage, applyCh)
	defer node.Kill()
	
	node.HandleRequestVote(&RequestVoteRequest{
		Term:         5,
		CandidateId:  1,
		LastLogIndex: 0,
		LastLogTerm:  0,
	})
	
	resp := node.HandleRequestVote(&RequestVoteRequest{
		Term:         3,
		CandidateId:  2,
		LastLogIndex: 0,
		LastLogTerm:  0,
	})
	
	if resp.VoteGranted {
		t.Error("should not grant vote for lower term")
	}
}

func TestNode_HandleAppendEntries(t *testing.T) {
	network := NewInmemNetwork(1)
	transport := network.CreateTransport(0)
	storage := NewMemoryRaftStorage()
	applyCh := make(chan ApplyMsg, 100)
	
	config := DefaultConfig()
	node := NewNode(config, transport, storage, applyCh)
	defer node.Kill()
	
	resp := node.HandleAppendEntries(&AppendEntriesRequest{
		Term:         2,
		LeaderId:     1,
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries: []LogEntry{
			{Term: 2, CommandValid: true, Command: []byte("cmd1")},
		},
		LeaderCommit: 0,
	})
	
	if !resp.Success {
		t.Error("expected success")
	}
	
	term, _ := node.GetState()
	if term != 2 {
		t.Errorf("expected term 2, got %d", term)
	}
}

func TestNode_HandleAppendEntries_RejectLowerTerm(t *testing.T) {
	network := NewInmemNetwork(1)
	transport := network.CreateTransport(0)
	storage := NewMemoryRaftStorage()
	applyCh := make(chan ApplyMsg, 100)
	
	config := DefaultConfig()
	node := NewNode(config, transport, storage, applyCh)
	defer node.Kill()
	
	node.HandleAppendEntries(&AppendEntriesRequest{
		Term:         5,
		LeaderId:     1,
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries:      nil,
		LeaderCommit: 0,
	})
	
	resp := node.HandleAppendEntries(&AppendEntriesRequest{
		Term:         3,
		LeaderId:     1,
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries:      nil,
		LeaderCommit: 0,
	})
	
	if resp.Success {
		t.Error("should not accept append entries from lower term")
	}
}

func TestNode_Snapshot(t *testing.T) {
	network := NewInmemNetwork(1)
	transport := network.CreateTransport(0)
	storage := NewMemoryRaftStorage()
	applyCh := make(chan ApplyMsg, 100)
	
	config := DefaultConfig()
	node := NewNode(config, transport, storage, applyCh)
	defer node.Kill()
	
	node.Snapshot(10, []byte("snapshot-data"))
}

func TestNode_Persistence(t *testing.T) {
	dir, err := os.MkdirTemp("", "raft-node-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)
	
	network := NewInmemNetwork(1)
	transport := network.CreateTransport(0)
	storage, err := NewBadgerRaftStorage(dir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	applyCh := make(chan ApplyMsg, 100)
	
	config := DefaultConfig()
	node := NewNode(config, transport, storage, applyCh)
	
	node.HandleRequestVote(&RequestVoteRequest{
		Term:         5,
		CandidateId:  1,
		LastLogIndex: 0,
		LastLogTerm:  0,
	})
	
	node.Kill()
	storage.Close()
	
	storage2, err := NewBadgerRaftStorage(dir)
	if err != nil {
		t.Fatalf("failed to create storage2: %v", err)
	}
	defer storage2.Close()
	
	state, err := storage2.LoadRaftState()
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}
	
	if state.CurrentTerm != 5 {
		t.Errorf("expected term 5, got %d", state.CurrentTerm)
	}
	
	if state.VotedFor != 1 {
		t.Errorf("expected votedFor 1, got %d", state.VotedFor)
	}
}

func TestNode_ElectionWithMultipleNodes(t *testing.T) {
	network := NewInmemNetwork(3)
	
	nodes := make([]*Node, 3)
	
	for i := 0; i < 3; i++ {
		transport := network.CreateTransport(i)
		storage := NewMemoryRaftStorage()
		applyCh := make(chan ApplyMsg, 100)
		
		config := &Config{
			ElectionTimeoutMin: 150 * time.Millisecond,
			ElectionTimeoutMax: 300 * time.Millisecond,
			ReplicateInterval:  50 * time.Millisecond,
		}
		
		nodes[i] = NewNode(config, transport, storage, applyCh)
	}
	
	time.Sleep(1 * time.Second)
	
	leaderCount := 0
	for _, node := range nodes {
		_, isLeader := node.GetState()
		if isLeader {
			leaderCount++
		}
	}
	
	if leaderCount != 1 {
		t.Errorf("expected exactly 1 leader, got %d", leaderCount)
	}
	
	for _, node := range nodes {
		node.Kill()
	}
}
