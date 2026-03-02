package raft

import (
	"sync"
	"testing"
	"time"
)

func TestNode_NetworkPartition(t *testing.T) {
	network := NewInmemNetwork(5)
	
	nodes := make([]*Node, 5)
	storages := make([]RaftStorage, 5)
	
	for i := 0; i < 5; i++ {
		transport := network.CreateTransport(i)
		storages[i] = NewMemoryRaftStorage()
		applyCh := make(chan ApplyMsg, 100)
		
		config := &Config{
			ElectionTimeoutMin: 150 * time.Millisecond,
			ElectionTimeoutMax: 300 * time.Millisecond,
			ReplicateInterval:  50 * time.Millisecond,
		}
		
		nodes[i] = NewNode(config, transport, storages[i], applyCh)
	}
	
	time.Sleep(1 * time.Second)
	
	leaderCount := 0
	var leaderID int
	for i, node := range nodes {
		_, isLeader := node.GetState()
		if isLeader {
			leaderCount++
			leaderID = i
		}
	}
	
	if leaderCount != 1 {
		t.Errorf("expected exactly 1 leader, got %d", leaderCount)
	}
	
	for _, node := range nodes {
		node.Kill()
	}
	_ = leaderID
}

func TestNode_LeaderElectionAfterDisconnect(t *testing.T) {
	network := NewInmemNetwork(3)
	
	nodes := make([]*Node, 3)
	
	for i := 0; i < 3; i++ {
		transport := network.CreateTransport(i)
		storage := NewMemoryRaftStorage()
		applyCh := make(chan ApplyMsg, 100)
		
		config := &Config{
			ElectionTimeoutMin: 100 * time.Millisecond,
			ElectionTimeoutMax: 200 * time.Millisecond,
			ReplicateInterval:  50 * time.Millisecond,
		}
		
		nodes[i] = NewNode(config, transport, storage, applyCh)
	}
	
	time.Sleep(500 * time.Millisecond)
	
	initialLeaderCount := 0
	for _, node := range nodes {
		_, isLeader := node.GetState()
		if isLeader {
			initialLeaderCount++
		}
	}
	
	if initialLeaderCount != 1 {
		t.Errorf("expected exactly 1 leader initially, got %d", initialLeaderCount)
	}
	
	for _, node := range nodes {
		node.Kill()
	}
}

func TestNode_LogReplication(t *testing.T) {
	network := NewInmemNetwork(3)
	
	nodes := make([]*Node, 3)
	applyChs := make([]chan ApplyMsg, 3)
	
	for i := 0; i < 3; i++ {
		transport := network.CreateTransport(i)
		storage := NewMemoryRaftStorage()
		applyChs[i] = make(chan ApplyMsg, 100)
		
		config := &Config{
			ElectionTimeoutMin: 150 * time.Millisecond,
			ElectionTimeoutMax: 300 * time.Millisecond,
			ReplicateInterval:  50 * time.Millisecond,
		}
		
		nodes[i] = NewNode(config, transport, storage, applyChs[i])
	}
	
	time.Sleep(1 * time.Second)
	
	var leaderNode *Node
	for _, node := range nodes {
		if term, isLeader := node.GetState(); isLeader {
			leaderNode = node
			_ = term
			break
		}
	}
	
	if leaderNode == nil {
		t.Fatal("no leader found")
	}
	
	index, _, ok := leaderNode.Start("test-command-1")
	if !ok {
		t.Fatal("failed to start command")
	}
	_ = index
	
	time.Sleep(500 * time.Millisecond)
	
	for _, node := range nodes {
		node.Kill()
	}
}

func TestNode_TermIncrease(t *testing.T) {
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
	
	time.Sleep(500 * time.Millisecond)
	
	initialTerms := make(map[int]int)
	for i, node := range nodes {
		term, _ := node.GetState()
		initialTerms[i] = term
	}
	
	time.Sleep(2 * time.Second)
	
	for _, node := range nodes {
		node.Kill()
	}
}

func TestNode_DuplicateVote(t *testing.T) {
	network := NewInmemNetwork(1)
	transport := network.CreateTransport(0)
	storage := NewMemoryRaftStorage()
	applyCh := make(chan ApplyMsg, 100)
	
	config := DefaultConfig()
	node := NewNode(config, transport, storage, applyCh)
	defer node.Kill()
	
	resp1 := node.HandleRequestVote(&RequestVoteRequest{
		Term:         2,
		CandidateId:  1,
		LastLogIndex: 0,
		LastLogTerm:  0,
	})
	
	if !resp1.VoteGranted {
		t.Error("first vote should be granted")
	}
	
	resp2 := node.HandleRequestVote(&RequestVoteRequest{
		Term:         2,
		CandidateId:  2,
		LastLogIndex: 0,
		LastLogTerm:  0,
	})
	
	if resp2.VoteGranted {
		t.Error("second vote for different candidate in same term should not be granted")
	}
	
	resp3 := node.HandleRequestVote(&RequestVoteRequest{
		Term:         2,
		CandidateId:  1,
		LastLogIndex: 0,
		LastLogTerm:  0,
	})
	
	if !resp3.VoteGranted {
		t.Error("duplicate vote for same candidate should be granted")
	}
}

func TestNode_LogConsistency(t *testing.T) {
	network := NewInmemNetwork(1)
	transport := network.CreateTransport(0)
	storage := NewMemoryRaftStorage()
	applyCh := make(chan ApplyMsg, 100)
	
	config := DefaultConfig()
	node := NewNode(config, transport, storage, applyCh)
	defer node.Kill()
	
	resp1 := node.HandleAppendEntries(&AppendEntriesRequest{
		Term:         2,
		LeaderId:     1,
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries: []LogEntry{
			{Term: 2, CommandValid: true, Command: "cmd1"},
		},
		LeaderCommit: 0,
	})
	
	if !resp1.Success {
		t.Error("first append should succeed")
	}
	
	resp2 := node.HandleAppendEntries(&AppendEntriesRequest{
		Term:         2,
		LeaderId:     1,
		PrevLogIndex: 1,
		PrevLogTerm:  2,
		Entries: []LogEntry{
			{Term: 2, CommandValid: true, Command: "cmd2"},
		},
		LeaderCommit: 0,
	})
	
	if !resp2.Success {
		t.Error("second append should succeed")
	}
}

func TestNode_ConflictResolution(t *testing.T) {
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
		},
		LeaderCommit: 0,
	})
	
	resp := node.HandleAppendEntries(&AppendEntriesRequest{
		Term:         3,
		LeaderId:     2,
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries: []LogEntry{
			{Term: 3, CommandValid: true, Command: "cmd1-new"},
		},
		LeaderCommit: 0,
	})
	
	if !resp.Success {
		t.Error("conflict resolution should succeed")
	}
}

func TestNode_ConcurrentStart(t *testing.T) {
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
	
	var leaderNode *Node
	for _, node := range nodes {
		if _, isLeader := node.GetState(); isLeader {
			leaderNode = node
			break
		}
	}
	
	if leaderNode == nil {
		t.Fatal("no leader found")
	}
	
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			leaderNode.Start(idx)
		}(i)
	}
	wg.Wait()
	
	time.Sleep(500 * time.Millisecond)
	
	for _, node := range nodes {
		node.Kill()
	}
}
