package raft

import "context"

type RequestVoteRequest struct {
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteResponse struct {
	Term        int
	VoteGranted bool
}

type AppendEntriesRequest struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

type AppendEntriesResponse struct {
	Term         int
	Success      bool
	ConflictIdx  int
	ConflictTerm int
}

type InstallSnapshotRequest struct {
	Term              int
	LeaderId          int
	LastIncludedIndex int
	LastIncludedTerm  int
	Snapshot          []byte
}

type InstallSnapshotResponse struct {
	Term int
}

type RPCEndpoint interface {
	SendRequestVote(ctx context.Context, req *RequestVoteRequest) (*RequestVoteResponse, error)
	SendAppendEntries(ctx context.Context, req *AppendEntriesRequest) (*AppendEntriesResponse, error)
	SendInstallSnapshot(ctx context.Context, req *InstallSnapshotRequest) (*InstallSnapshotResponse, error)
}

type RPCHandler interface {
	HandleRequestVote(req *RequestVoteRequest) *RequestVoteResponse
	HandleAppendEntries(req *AppendEntriesRequest) *AppendEntriesResponse
	HandleInstallSnapshot(req *InstallSnapshotRequest) *InstallSnapshotResponse
}

type Transport interface {
	GetPeerCount() int
	GetPeerID() int
	GetEndpoint(peerID int) RPCEndpoint
	RegisterHandler(handler RPCHandler)
	Start() error
	Stop()
}

type TransportConfig struct {
	Peers      []string
	PeerID     int
	ListenAddr string
}
