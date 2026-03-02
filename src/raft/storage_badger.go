package raft

import (
	"bytes"
	"encoding/gob"
	"sync"

	"github.com/dgraph-io/badger/v4"
)

type BadgerStorage struct {
	mu     sync.RWMutex
	db     *badger.DB
	closed bool
}

func NewBadgerStorage(dataDir string) (*BadgerStorage, error) {
	opts := badger.DefaultOptions(dataDir).
		WithLoggingLevel(badger.ERROR).
		WithValueLogFileSize(64 << 20)
	
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	
	return &BadgerStorage{db: db}, nil
}

func (s *BadgerStorage) Get(key []byte) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.closed {
		return nil, ErrClosed
	}
	
	var val []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrNotFound
			}
			return err
		}
		val, err = item.ValueCopy(nil)
		return err
	})
	
	return val, err
}

func (s *BadgerStorage) Set(key []byte, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.closed {
		return ErrClosed
	}
	
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

func (s *BadgerStorage) Delete(key []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.closed {
		return ErrClosed
	}
	
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

func (s *BadgerStorage) Has(key []byte) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.closed {
		return false, ErrClosed
	}
	
	err := s.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		return err
	})
	
	if err == badger.ErrKeyNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *BadgerStorage) BatchWrite(writes []Write) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.closed {
		return ErrClosed
	}
	
	return s.db.Update(func(txn *badger.Txn) error {
		for _, w := range writes {
			if w.Delete {
				if err := txn.Delete(w.Key); err != nil {
					return err
				}
			} else {
				if err := txn.Set(w.Key, w.Value); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (s *BadgerStorage) Sync() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.closed {
		return ErrClosed
	}
	return s.db.Sync()
}

func (s *BadgerStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.closed {
		return nil
	}
	
	s.closed = true
	return s.db.Close()
}

type BadgerRaftStorage struct {
	mu      sync.RWMutex
	storage *BadgerStorage
}

func NewBadgerRaftStorage(dataDir string) (*BadgerRaftStorage, error) {
	storage, err := NewBadgerStorage(dataDir)
	if err != nil {
		return nil, err
	}
	return &BadgerRaftStorage{storage: storage}, nil
}

func (s *BadgerRaftStorage) SaveRaftState(state *RaftState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(state); err != nil {
		return err
	}
	
	return s.storage.Set([]byte(raftStateKey), buf.Bytes())
}

func (s *BadgerRaftStorage) LoadRaftState() (*RaftState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	data, err := s.storage.Get([]byte(raftStateKey))
	if err != nil {
		return nil, err
	}
	
	state := &RaftState{}
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(state); err != nil {
		return nil, ErrCorrupted
	}
	
	return state, nil
}

func (s *BadgerRaftStorage) SaveSnapshot(snapshot *Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(snapshot); err != nil {
		return err
	}
	
	return s.storage.Set([]byte(snapshotKey), buf.Bytes())
}

func (s *BadgerRaftStorage) LoadSnapshot() (*Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	data, err := s.storage.Get([]byte(snapshotKey))
	if err != nil {
		return nil, err
	}
	
	snapshot := &Snapshot{}
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(snapshot); err != nil {
		return nil, ErrCorrupted
	}
	
	return snapshot, nil
}

func (s *BadgerRaftStorage) RaftStateSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	data, err := s.storage.Get([]byte(raftStateKey))
	if err != nil {
		return 0
	}
	return len(data)
}

func (s *BadgerRaftStorage) SnapshotSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	data, err := s.storage.Get([]byte(snapshotKey))
	if err != nil {
		return 0
	}
	return len(data)
}

func (s *BadgerRaftStorage) Storage() Storage {
	return s.storage
}

func (s *BadgerRaftStorage) Close() error {
	return s.storage.Close()
}
