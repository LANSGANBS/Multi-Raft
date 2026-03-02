package raft

import (
	"testing"
)

func TestMemoryStorage_Basic(t *testing.T) {
	s := NewMemoryStorage()
	defer s.Close()
	
	key := []byte("test-key")
	value := []byte("test-value")
	
	has, err := s.Has(key)
	if err != nil {
		t.Fatalf("Has failed: %v", err)
	}
	if has {
		t.Error("key should not exist")
	}
	
	if err := s.Set(key, value); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	
	has, err = s.Has(key)
	if err != nil {
		t.Fatalf("Has failed: %v", err)
	}
	if !has {
		t.Error("key should exist")
	}
	
	got, err := s.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(got) != string(value) {
		t.Errorf("expected %s, got %s", value, got)
	}
	
	if err := s.Delete(key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	
	has, err = s.Has(key)
	if err != nil {
		t.Fatalf("Has failed: %v", err)
	}
	if has {
		t.Error("key should not exist after delete")
	}
}

func TestMemoryStorage_NotFound(t *testing.T) {
	s := NewMemoryStorage()
	defer s.Close()
	
	_, err := s.Get([]byte("nonexistent"))
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStorage_BatchWrite(t *testing.T) {
	s := NewMemoryStorage()
	defer s.Close()
	
	writes := []Write{
		{Key: []byte("key1"), Value: []byte("value1")},
		{Key: []byte("key2"), Value: []byte("value2")},
		{Key: []byte("key3"), Value: []byte("value3")},
	}
	
	if err := s.BatchWrite(writes); err != nil {
		t.Fatalf("BatchWrite failed: %v", err)
	}
	
	for _, w := range writes {
		got, err := s.Get(w.Key)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if string(got) != string(w.Value) {
			t.Errorf("expected %s, got %s", w.Value, got)
		}
	}
}

func TestMemoryStorage_Closed(t *testing.T) {
	s := NewMemoryStorage()
	
	if err := s.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	
	key := []byte("test")
	
	if err := s.Set(key, []byte("value")); err != ErrClosed {
		t.Errorf("expected ErrClosed, got %v", err)
	}
	
	if _, err := s.Get(key); err != ErrClosed {
		t.Errorf("expected ErrClosed, got %v", err)
	}
}

func TestMemoryRaftStorage_State(t *testing.T) {
	s := NewMemoryRaftStorage()
	
	state := &RaftState{
		CurrentTerm: 5,
		VotedFor:    3,
		Log:         []byte("log-data"),
	}
	
	if err := s.SaveRaftState(state); err != nil {
		t.Fatalf("SaveRaftState failed: %v", err)
	}
	
	if s.RaftStateSize() == 0 {
		t.Error("RaftStateSize should be > 0")
	}
	
	loaded, err := s.LoadRaftState()
	if err != nil {
		t.Fatalf("LoadRaftState failed: %v", err)
	}
	
	if loaded.CurrentTerm != state.CurrentTerm {
		t.Errorf("expected term %d, got %d", state.CurrentTerm, loaded.CurrentTerm)
	}
	if loaded.VotedFor != state.VotedFor {
		t.Errorf("expected votedFor %d, got %d", state.VotedFor, loaded.VotedFor)
	}
}

func TestMemoryRaftStorage_Snapshot(t *testing.T) {
	s := NewMemoryRaftStorage()
	
	snapshot := &Snapshot{
		Index: 100,
		Term:  5,
		Data:  []byte("snapshot-data"),
	}
	
	if err := s.SaveSnapshot(snapshot); err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}
	
	if s.SnapshotSize() == 0 {
		t.Error("SnapshotSize should be > 0")
	}
	
	loaded, err := s.LoadSnapshot()
	if err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}
	
	if loaded.Index != snapshot.Index {
		t.Errorf("expected index %d, got %d", snapshot.Index, loaded.Index)
	}
	if loaded.Term != snapshot.Term {
		t.Errorf("expected term %d, got %d", snapshot.Term, loaded.Term)
	}
}

func TestMemoryRaftStorage_NotFound(t *testing.T) {
	s := NewMemoryRaftStorage()
	
	_, err := s.LoadRaftState()
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	
	_, err = s.LoadSnapshot()
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
