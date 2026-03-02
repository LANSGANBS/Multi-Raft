package raft

import (
	"bytes"
	"encoding/gob"
	"sync"
)

type MemoryStorage struct {
	mu     sync.RWMutex
	data   map[string][]byte
	closed bool
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string][]byte),
	}
}

func (s *MemoryStorage) Get(key []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.closed {
		return nil, ErrClosed
	}
	
	val, ok := s.data[string(key)]
	if !ok {
		return nil, ErrNotFound
	}
	
	result := make([]byte, len(val))
	copy(result, val)
	return result, nil
}

func (s *MemoryStorage) Set(key []byte, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.closed {
		return ErrClosed
	}
	
	s.data[string(key)] = value
	return nil
}

func (s *MemoryStorage) Delete(key []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.closed {
		return ErrClosed
	}
	
	delete(s.data, string(key))
	return nil
}

func (s *MemoryStorage) Has(key []byte) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.closed {
		return false, ErrClosed
	}
	
	_, ok := s.data[string(key)]
	return ok, nil
}

func (s *MemoryStorage) BatchWrite(writes []Write) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.closed {
		return ErrClosed
	}
	
	for _, w := range writes {
		if w.Delete {
			delete(s.data, string(w.Key))
		} else {
			s.data[string(w.Key)] = w.Value
		}
	}
	return nil
}

func (s *MemoryStorage) Sync() error {
	return nil
}

func (s *MemoryStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.closed = true
	s.data = nil
	return nil
}

type MemoryRaftStorage struct {
	mu         sync.RWMutex
	storage    *MemoryStorage
	raftState  []byte
	snapshot   []byte
}

func NewMemoryRaftStorage() *MemoryRaftStorage {
	return &MemoryRaftStorage{
		storage: NewMemoryStorage(),
	}
}

func (s *MemoryRaftStorage) SaveRaftState(state *RaftState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(state); err != nil {
		return err
	}
	
	s.raftState = buf.Bytes()
	return nil
}

func (s *MemoryRaftStorage) LoadRaftState() (*RaftState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if len(s.raftState) == 0 {
		return nil, ErrNotFound
	}
	
	state := &RaftState{}
	buf := bytes.NewBuffer(s.raftState)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(state); err != nil {
		return nil, ErrCorrupted
	}
	
	return state, nil
}

func (s *MemoryRaftStorage) SaveSnapshot(snapshot *Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(snapshot); err != nil {
		return err
	}
	
	s.snapshot = buf.Bytes()
	return nil
}

func (s *MemoryRaftStorage) LoadSnapshot() (*Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if len(s.snapshot) == 0 {
		return nil, ErrNotFound
	}
	
	snapshot := &Snapshot{}
	buf := bytes.NewBuffer(s.snapshot)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(snapshot); err != nil {
		return nil, ErrCorrupted
	}
	
	return snapshot, nil
}

func (s *MemoryRaftStorage) RaftStateSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.raftState)
}

func (s *MemoryRaftStorage) SnapshotSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.snapshot)
}

func (s *MemoryRaftStorage) Storage() Storage {
	return s.storage
}

func (s *MemoryRaftStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.storage.Close()
}
