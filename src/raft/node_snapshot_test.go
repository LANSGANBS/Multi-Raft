package raft

import (
	"os"
	"testing"
	"time"
)

func TestNode_SnapshotInstall(t *testing.T) {
	network := NewInmemNetwork(1)
	transport := network.CreateTransport(0)
	storage := NewMemoryRaftStorage()
	applyCh := make(chan ApplyMsg, 100)
	
	config := DefaultConfig()
	node := NewNode(config, transport, storage, applyCh)
	defer node.Kill()
	
	node.HandleAppendEntries(&AppendEntriesRequest{
		Term:         2,
		LeaderId:     1,
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries: []LogEntry{
			{Term: 2, CommandValid: true, Command: "cmd1"},
			{Term: 2, CommandValid: true, Command: "cmd2"},
			{Term: 2, CommandValid: true, Command: "cmd3"},
		},
		LeaderCommit: 3,
	})
	
	resp := node.HandleInstallSnapshot(&InstallSnapshotRequest{
		Term:              3,
		LeaderId:          1,
		LastIncludedIndex: 5,
		LastIncludedTerm:  2,
		Snapshot:          []byte("snapshot-data"),
	})
	
	if resp.Term != 3 {
		t.Errorf("expected term 3, got %d", resp.Term)
	}
}

func TestNode_SnapshotRejectOlder(t *testing.T) {
	network := NewInmemNetwork(1)
	transport := network.CreateTransport(0)
	storage := NewMemoryRaftStorage()
	applyCh := make(chan ApplyMsg, 100)
	
	config := DefaultConfig()
	node := NewNode(config, transport, storage, applyCh)
	defer node.Kill()
	
	node.HandleInstallSnapshot(&InstallSnapshotRequest{
		Term:              2,
		LeaderId:          1,
		LastIncludedIndex: 10,
		LastIncludedTerm:  2,
		Snapshot:          []byte("snapshot-data-1"),
	})
	
	resp := node.HandleInstallSnapshot(&InstallSnapshotRequest{
		Term:              2,
		LeaderId:          1,
		LastIncludedIndex: 5,
		LastIncludedTerm:  2,
		Snapshot:          []byte("snapshot-data-2"),
	})
	
	if resp.Term != 2 {
		t.Errorf("expected term 2, got %d", resp.Term)
	}
}

func TestNode_SnapshotPersistence(t *testing.T) {
	dir, err := os.MkdirTemp("", "raft-snapshot-test")
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
	
	node.HandleInstallSnapshot(&InstallSnapshotRequest{
		Term:              2,
		LeaderId:          1,
		LastIncludedIndex: 100,
		LastIncludedTerm:  2,
		Snapshot:          []byte("persistent-snapshot"),
	})
	
	node.Kill()
	storage.Close()
	
	storage2, err := NewBadgerRaftStorage(dir)
	if err != nil {
		t.Fatalf("failed to create storage2: %v", err)
	}
	defer storage2.Close()
	
	snap, err := storage2.LoadSnapshot()
	if err != nil {
		t.Fatalf("failed to load snapshot: %v", err)
	}
	
	if snap.Index != 100 {
		t.Errorf("expected snapshot index 100, got %d", snap.Index)
	}
	
	if snap.Term != 2 {
		t.Errorf("expected snapshot term 2, got %d", snap.Term)
	}
}

func TestNode_SnapshotApply(t *testing.T) {
	network := NewInmemNetwork(1)
	transport := network.CreateTransport(0)
	storage := NewMemoryRaftStorage()
	applyCh := make(chan ApplyMsg, 100)
	
	config := DefaultConfig()
	node := NewNode(config, transport, storage, applyCh)
	defer node.Kill()
	
	go func() {
		for msg := range applyCh {
			_ = msg
		}
	}()
	
	node.HandleInstallSnapshot(&InstallSnapshotRequest{
		Term:              2,
		LeaderId:          1,
		LastIncludedIndex: 50,
		LastIncludedTerm:  2,
		Snapshot:          []byte("snapshot-apply-test"),
	})
	
	time.Sleep(100 * time.Millisecond)
}

func TestNode_LogCompaction(t *testing.T) {
	network := NewInmemNetwork(1)
	transport := network.CreateTransport(0)
	storage := NewMemoryRaftStorage()
	applyCh := make(chan ApplyMsg, 100)
	
	config := DefaultConfig()
	node := NewNode(config, transport, storage, applyCh)
	defer node.Kill()
	
	for i := 0; i < 10; i++ {
		node.HandleAppendEntries(&AppendEntriesRequest{
			Term:         2,
			LeaderId:     1,
			PrevLogIndex: i,
			PrevLogTerm:  2,
			Entries: []LogEntry{
				{Term: 2, CommandValid: true, Command: i},
			},
			LeaderCommit: i + 1,
		})
	}
	
	node.Snapshot(5, []byte("compaction-snapshot"))
}

func TestNode_StateMachineRecovery(t *testing.T) {
	dir, err := os.MkdirTemp("", "raft-recovery-test")
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
	
	node.HandleAppendEntries(&AppendEntriesRequest{
		Term:         5,
		LeaderId:     1,
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries: []LogEntry{
			{Term: 5, CommandValid: true, Command: "recovery-cmd"},
		},
		LeaderCommit: 1,
	})
	
	node.Kill()
	storage.Close()
	
	storage2, err := NewBadgerRaftStorage(dir)
	if err != nil {
		t.Fatalf("failed to create storage2: %v", err)
	}
	
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
	
	storage2.Close()
}
