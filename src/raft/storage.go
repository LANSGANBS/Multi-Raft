package raft

import "io"

type StorageError string

func (e StorageError) Error() string { return string(e) }

const (
	ErrNotFound   StorageError = "key not found"
	ErrCorrupted  StorageError = "data corrupted"
	ErrClosed     StorageError = "storage closed"
)

type Storage interface {
	io.Closer
	
	Get(key []byte) ([]byte, error)
	Set(key []byte, value []byte) error
	Delete(key []byte) error
	
	Has(key []byte) (bool, error)
	
	BatchWrite(writes []Write) error
	
	Sync() error
}

type Write struct {
	Key   []byte
	Value []byte
	Delete bool
}

type RaftState struct {
	CurrentTerm int
	VotedFor    int
	Log         []byte
}

type Snapshot struct {
	Index int
	Term  int
	Data  []byte
}

const (
	raftStateKey = "raft_state"
	snapshotKey  = "snapshot"
)

type RaftStorage interface {
	SaveRaftState(state *RaftState) error
	LoadRaftState() (*RaftState, error)
	
	SaveSnapshot(snapshot *Snapshot) error
	LoadSnapshot() (*Snapshot, error)
	
	RaftStateSize() int
	SnapshotSize() int
	
	Storage() Storage
}

type StorageConfig struct {
	DataDir string
}
