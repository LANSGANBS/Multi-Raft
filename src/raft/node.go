package raft

import (
	"bytes"
	"context"
	"encoding/gob"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultElectionTimeoutMin = 250 * time.Millisecond
	DefaultElectionTimeoutMax = 400 * time.Millisecond
	DefaultReplicateInterval  = 30 * time.Millisecond
)

type Config struct {
	ElectionTimeoutMin time.Duration
	ElectionTimeoutMax time.Duration
	ReplicateInterval  time.Duration
}

func DefaultConfig() *Config {
	return &Config{
		ElectionTimeoutMin: DefaultElectionTimeoutMin,
		ElectionTimeoutMax: DefaultElectionTimeoutMax,
		ReplicateInterval:  DefaultReplicateInterval,
	}
}

func (c *Config) Validate() {
	if c.ElectionTimeoutMin <= 0 {
		c.ElectionTimeoutMin = DefaultElectionTimeoutMin
	}
	if c.ElectionTimeoutMax <= c.ElectionTimeoutMin {
		c.ElectionTimeoutMax = DefaultElectionTimeoutMax
	}
	if c.ReplicateInterval <= 0 {
		c.ReplicateInterval = DefaultReplicateInterval
	}
}

type Node struct {
	mu        sync.Mutex
	transport Transport
	storage   RaftStorage
	config    *Config
	me        int
	dead      int32
	wg        sync.WaitGroup

	role        Role
	currentTerm int
	votedFor    int

	log *RaftLog

	nextIndex  []int
	matchIndex []int

	commitIndex int
	lastApplied int
	applyCh     chan ApplyMsg
	snapPending bool
	applyCond   *sync.Cond

	electionStart   time.Time
	electionTimeout time.Duration
}

func NewNode(config *Config, transport Transport, storage RaftStorage, applyCh chan ApplyMsg) *Node {
	if config == nil {
		config = DefaultConfig()
	}
	config.Validate()

	n := &Node{
		config:      config,
		transport:   transport,
		storage:     storage,
		me:          transport.GetPeerID(),
		role:        Follower,
		currentTerm: 1,
		votedFor:    -1,
		log:         NewLog(InvalidIndex, InvalidTerm, nil, nil),
		applyCh:     applyCh,
		commitIndex: 0,
		lastApplied: 0,
	}

	n.applyCond = sync.NewCond(&n.mu)

	peerCount := transport.GetPeerCount()
	n.nextIndex = make([]int, peerCount)
	n.matchIndex = make([]int, peerCount)

	n.loadFromStorage()

	n.resetElectionTimerLocked()

	transport.RegisterHandler(n)

	n.wg.Add(2)
	go n.electionTicker()
	go n.applicationTicker()

	return n
}

func (n *Node) loadFromStorage() {
	state, err := n.storage.LoadRaftState()
	if err == nil && state != nil {
		n.currentTerm = state.CurrentTerm
		n.votedFor = state.VotedFor
		if len(state.Log) > 0 {
			n.deserializeLog(state.Log)
		}
	}

	snapshot, err := n.storage.LoadSnapshot()
	if err == nil && snapshot != nil {
		n.log = NewLog(snapshot.Index, snapshot.Term, snapshot.Data, nil)
		if snapshot.Index > n.commitIndex {
			n.commitIndex = snapshot.Index
			n.lastApplied = snapshot.Index
		}
	}
}

func (n *Node) GetState() (int, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.currentTerm, n.role == Leader
}

func (n *Node) Start(command interface{}) (int, int, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.role != Leader {
		return 0, 0, false
	}

	n.log.append(LogEntry{
		CommandValid: true,
		Command:      command,
		Term:         n.currentTerm,
	})

	n.persistLocked()

	return n.log.size() - 1, n.currentTerm, true
}

func (n *Node) Kill() {
	atomic.StoreInt32(&n.dead, 1)
}

func (n *Node) killed() bool {
	return atomic.LoadInt32(&n.dead) == 1
}

func (n *Node) Stop() {
	n.Kill()
	n.applyCond.Signal()
	n.wg.Wait()
	n.transport.Stop()
}

func (n *Node) Snapshot(index int, snapshot []byte) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if index > n.commitIndex {
		return
	}
	if index <= n.log.snapLastIdx {
		return
	}

	n.log.doSnapshot(index, snapshot)
	n.persistLocked()
}

func (n *Node) HandleRequestVote(req *RequestVoteRequest) *RequestVoteResponse {
	n.mu.Lock()
	defer n.mu.Unlock()

	resp := &RequestVoteResponse{
		Term:        n.currentTerm,
		VoteGranted: false,
	}

	if req.Term < n.currentTerm {
		return resp
	}

	if req.Term > n.currentTerm {
		n.becomeFollowerLocked(req.Term)
		resp.Term = n.currentTerm
	}

	if n.votedFor != -1 && n.votedFor != req.CandidateId {
		return resp
	}

	if n.isMoreUpToDateLocked(req.LastLogIndex, req.LastLogTerm) {
		return resp
	}

	resp.VoteGranted = true
	n.votedFor = req.CandidateId
	n.persistLocked()
	n.resetElectionTimerLocked()

	return resp
}

func (n *Node) HandleAppendEntries(req *AppendEntriesRequest) *AppendEntriesResponse {
	n.mu.Lock()
	defer n.mu.Unlock()

	resp := &AppendEntriesResponse{
		Term:    n.currentTerm,
		Success: false,
	}

	if req.Term < n.currentTerm {
		return resp
	}

	if req.Term >= n.currentTerm {
		n.becomeFollowerLocked(req.Term)
	}

	defer n.resetElectionTimerLocked()

	if req.PrevLogIndex >= n.log.size() {
		resp.ConflictIdx = n.log.size()
		resp.ConflictTerm = InvalidTerm
		return resp
	}

	if req.PrevLogIndex < n.log.snapLastIdx {
		resp.ConflictIdx = n.log.snapLastIdx
		resp.ConflictTerm = n.log.snapLastTerm
		return resp
	}

	if n.log.at(req.PrevLogIndex).Term != req.PrevLogTerm {
		resp.ConflictTerm = n.log.at(req.PrevLogIndex).Term
		resp.ConflictIdx = n.log.firstFor(resp.ConflictTerm)
		return resp
	}

	entries := make([]LogEntry, len(req.Entries))
	for i, e := range req.Entries {
		entries[i] = LogEntry{
			Term:         e.Term,
			CommandValid: e.CommandValid,
			Command:      e.Command,
		}
	}
	n.log.appendFrom(req.PrevLogIndex, entries)
	n.persistLocked()
	resp.Success = true

	if req.LeaderCommit > n.commitIndex {
		lastIdx, _ := n.log.last()
		if req.LeaderCommit > lastIdx {
			n.commitIndex = lastIdx
		} else {
			n.commitIndex = req.LeaderCommit
		}
		n.applyCond.Signal()
	}

	return resp
}

func (n *Node) HandleInstallSnapshot(req *InstallSnapshotRequest) *InstallSnapshotResponse {
	n.mu.Lock()
	defer n.mu.Unlock()

	if req.Term < n.currentTerm {
		return &InstallSnapshotResponse{Term: n.currentTerm}
	}

	if req.Term >= n.currentTerm {
		n.becomeFollowerLocked(req.Term)
	}

	defer n.resetElectionTimerLocked()

	if n.log.snapLastIdx >= req.LastIncludedIndex {
		return &InstallSnapshotResponse{Term: n.currentTerm}
	}

	n.log.installSnapshot(req.LastIncludedIndex, req.LastIncludedTerm, req.Snapshot)
	n.persistLocked()
	n.snapPending = true

	if req.LastIncludedIndex > n.commitIndex {
		n.commitIndex = req.LastIncludedIndex
	}

	n.applyCond.Signal()

	return &InstallSnapshotResponse{Term: n.currentTerm}
}

func (n *Node) becomeFollowerLocked(term int) {
	if term < n.currentTerm {
		return
	}

	n.role = Follower
	shouldPersist := n.currentTerm != term
	if term > n.currentTerm {
		n.votedFor = -1
	}
	n.currentTerm = term
	if shouldPersist {
		n.persistLocked()
	}
}

func (n *Node) becomeCandidateLocked() {
	if n.role == Leader {
		return
	}

	n.currentTerm++
	n.role = Candidate
	n.votedFor = n.me
	n.resetElectionTimerLocked()
	n.persistLocked()
}

func (n *Node) becomeLeaderLocked() {
	if n.role != Candidate {
		return
	}

	n.role = Leader
	for peer := 0; peer < n.transport.GetPeerCount(); peer++ {
		n.nextIndex[peer] = n.log.size()
		n.matchIndex[peer] = 0
	}
}

func (n *Node) isMoreUpToDateLocked(candidateIndex, candidateTerm int) bool {
	lastIndex, lastTerm := n.log.last()

	if lastTerm != candidateTerm {
		return lastTerm > candidateTerm
	}
	return lastIndex > candidateIndex
}

func (n *Node) resetElectionTimerLocked() {
	n.electionStart = time.Now()
	timeout := n.config.ElectionTimeoutMax - n.config.ElectionTimeoutMin
	n.electionTimeout = n.config.ElectionTimeoutMin + time.Duration(rand.Int63n(int64(timeout)))
}

func (n *Node) isElectionTimeoutLocked() bool {
	return time.Since(n.electionStart) > n.electionTimeout
}

func (n *Node) contextLostLocked(role Role, term int) bool {
	return !(n.currentTerm == term && n.role == role)
}

func (n *Node) persistLocked() {
	logData := n.serializeLog()
	n.storage.SaveRaftState(&RaftState{
		CurrentTerm: n.currentTerm,
		VotedFor:    n.votedFor,
		Log:         logData,
	})
	n.storage.SaveSnapshot(&Snapshot{
		Index: n.log.snapLastIdx,
		Term:  n.log.snapLastTerm,
		Data:  n.log.snapshot,
	})
}

func (n *Node) serializeLog() []byte {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	enc.Encode(n.log.snapLastIdx)
	enc.Encode(n.log.snapLastTerm)
	enc.Encode(n.log.tailLog)
	return buf.Bytes()
}

func (n *Node) deserializeLog(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)

	var snapLastIdx int
	if err := dec.Decode(&snapLastIdx); err != nil {
		return err
	}

	var snapLastTerm int
	if err := dec.Decode(&snapLastTerm); err != nil {
		return err
	}

	var tailLog []LogEntry
	if err := dec.Decode(&tailLog); err != nil {
		return err
	}

	n.log = &RaftLog{
		snapLastIdx:  snapLastIdx,
		snapLastTerm: snapLastTerm,
		tailLog:      tailLog,
	}
	return nil
}

func (n *Node) electionTicker() {
	defer n.wg.Done()
	for !n.killed() {
		n.mu.Lock()
		if n.role != Leader && n.isElectionTimeoutLocked() {
			n.becomeCandidateLocked()
			go n.startElection(n.currentTerm)
		}
		n.mu.Unlock()

		time.Sleep(50 * time.Millisecond)
	}
}

func (n *Node) startElection(term int) {
	var votesMu sync.Mutex
	votes := 1

	n.mu.Lock()
	if n.contextLostLocked(Candidate, term) {
		n.mu.Unlock()
		return
	}

	lastIdx, lastTerm := n.log.last()
	n.resetElectionTimerLocked()
	n.mu.Unlock()

	for peer := 0; peer < n.transport.GetPeerCount(); peer++ {
		if peer == n.me {
			continue
		}

		go func(peerID int) {
			endpoint := n.transport.GetEndpoint(peerID)
			if endpoint == nil {
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			resp, err := endpoint.SendRequestVote(ctx, &RequestVoteRequest{
				Term:         term,
				CandidateId:  n.me,
				LastLogIndex: lastIdx,
				LastLogTerm:  lastTerm,
			})

			if err != nil {
				return
			}

			n.mu.Lock()

			if resp.Term > n.currentTerm {
				n.becomeFollowerLocked(resp.Term)
				n.mu.Unlock()
				return
			}

			if n.contextLostLocked(Candidate, term) {
				n.mu.Unlock()
				return
			}

			n.mu.Unlock()

			if resp.VoteGranted {
				votesMu.Lock()
				votes++
				won := votes > n.transport.GetPeerCount()/2
				votesMu.Unlock()

				if won {
					n.mu.Lock()
					if n.role == Candidate && n.currentTerm == term {
						n.becomeLeaderLocked()
						go n.replicationTicker(term)
					}
					n.mu.Unlock()
				}
			}
		}(peer)
	}
}

func (n *Node) replicationTicker(term int) {
	for !n.killed() {
		n.mu.Lock()
		if n.contextLostLocked(Leader, term) {
			n.mu.Unlock()
			return
		}
		n.mu.Unlock()

		n.startReplication(term)
		time.Sleep(n.config.ReplicateInterval)
	}
}

func (n *Node) startReplication(term int) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.contextLostLocked(Leader, term) {
		return
	}

	for peer := 0; peer < n.transport.GetPeerCount(); peer++ {
		if peer == n.me {
			n.matchIndex[peer] = n.log.size() - 1
			n.nextIndex[peer] = n.log.size()
			continue
		}

		prevIdx := n.nextIndex[peer] - 1

		if prevIdx < n.log.snapLastIdx {
			snapIdx := n.log.snapLastIdx
			snapTerm := n.log.snapLastTerm
			snapData := make([]byte, len(n.log.snapshot))
			copy(snapData, n.log.snapshot)
			go n.sendInstallSnapshot(peer, term, snapIdx, snapTerm, snapData)
			continue
		}

		prevTerm := n.log.at(prevIdx).Term
		entries := n.log.tail(prevIdx + 1)
		entriesCopy := make([]LogEntry, len(entries))
		copy(entriesCopy, entries)
		commitIdx := n.commitIndex

		go n.sendAppendEntries(peer, term, prevIdx, prevTerm, entriesCopy, commitIdx)
	}
}

func (n *Node) sendAppendEntries(peer, term, prevIdx, prevTerm int, entries []LogEntry, commitIdx int) {
	endpoint := n.transport.GetEndpoint(peer)
	if endpoint == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := endpoint.SendAppendEntries(ctx, &AppendEntriesRequest{
		Term:         term,
		LeaderId:     n.me,
		PrevLogIndex: prevIdx,
		PrevLogTerm:  prevTerm,
		Entries:      entries,
		LeaderCommit: commitIdx,
	})

	if err != nil {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if resp.Term > n.currentTerm {
		n.becomeFollowerLocked(resp.Term)
		return
	}

	if n.contextLostLocked(Leader, term) {
		return
	}

	if !resp.Success {
		if resp.ConflictTerm == InvalidTerm {
			n.nextIndex[peer] = resp.ConflictIdx
		} else {
			firstIdx := n.log.firstFor(resp.ConflictTerm)
			if firstIdx != InvalidIndex {
				n.nextIndex[peer] = firstIdx
			} else {
				n.nextIndex[peer] = resp.ConflictIdx
			}
		}
		if n.nextIndex[peer] > prevIdx+1 {
			n.nextIndex[peer] = prevIdx + 1
		}
		minIdx := n.log.snapLastIdx + 1
		if n.nextIndex[peer] < minIdx {
			n.nextIndex[peer] = minIdx
		}
		return
	}

	n.matchIndex[peer] = prevIdx + len(entries)
	n.nextIndex[peer] = n.matchIndex[peer] + 1

	n.updateCommitIndexLocked()
}

func (n *Node) sendInstallSnapshot(peer, term, snapIdx, snapTerm int, snapData []byte) {
	endpoint := n.transport.GetEndpoint(peer)
	if endpoint == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := endpoint.SendInstallSnapshot(ctx, &InstallSnapshotRequest{
		Term:              term,
		LeaderId:          n.me,
		LastIncludedIndex: snapIdx,
		LastIncludedTerm:  snapTerm,
		Snapshot:          snapData,
	})

	if err != nil {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if resp.Term > n.currentTerm {
		n.becomeFollowerLocked(resp.Term)
		return
	}

	if n.contextLostLocked(Leader, term) {
		return
	}

	n.matchIndex[peer] = n.log.snapLastIdx
	n.nextIndex[peer] = n.log.snapLastIdx + 1
}

func (n *Node) updateCommitIndexLocked() {
	peerCount := n.transport.GetPeerCount()
	tmpIndexes := make([]int, peerCount)
	copy(tmpIndexes, n.matchIndex)

	for i := 0; i < peerCount-1; i++ {
		for j := i + 1; j < peerCount; j++ {
			if tmpIndexes[i] > tmpIndexes[j] {
				tmpIndexes[i], tmpIndexes[j] = tmpIndexes[j], tmpIndexes[i]
			}
		}
	}

	majorityIdx := tmpIndexes[(peerCount-1)/2]

	if majorityIdx > n.commitIndex {
		if majorityIdx >= n.log.snapLastIdx && n.log.at(majorityIdx).Term == n.currentTerm {
			n.commitIndex = majorityIdx
			n.applyCond.Signal()
		}
	}
}

func (n *Node) applicationTicker() {
	defer n.wg.Done()
	for !n.killed() {
		n.mu.Lock()

		for !n.killed() && n.commitIndex <= n.lastApplied && !n.snapPending {
			n.applyCond.Wait()
		}

		if n.killed() {
			n.mu.Unlock()
			return
		}

		var msgs []ApplyMsg

		if n.snapPending {
			n.snapPending = false
			msgs = append(msgs, ApplyMsg{
				SnapshotValid: true,
				Snapshot:      n.log.snapshot,
				SnapshotTerm:  n.log.snapLastTerm,
				SnapshotIndex: n.log.snapLastIdx,
			})
			n.lastApplied = n.log.snapLastIdx
		}

		for n.lastApplied < n.commitIndex {
			n.lastApplied++
			entry := n.log.at(n.lastApplied)
			msgs = append(msgs, ApplyMsg{
				CommandValid: true,
				Command:      entry.Command,
				CommandIndex: n.lastApplied,
			})
		}

		n.mu.Unlock()

		for _, msg := range msgs {
			select {
			case n.applyCh <- msg:
			default:
			}
		}
	}
}
