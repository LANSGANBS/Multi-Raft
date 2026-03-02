package raft

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBadgerStorage_Basic(t *testing.T) {
	dir, err := os.MkdirTemp("", "badger-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)
	
	s, err := NewBadgerStorage(dir)
	if err != nil {
		t.Fatalf("NewBadgerStorage failed: %v", err)
	}
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

func TestBadgerStorage_NotFound(t *testing.T) {
	dir, err := os.MkdirTemp("", "badger-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)
	
	s, err := NewBadgerStorage(dir)
	if err != nil {
		t.Fatalf("NewBadgerStorage failed: %v", err)
	}
	defer s.Close()
	
	_, err = s.Get([]byte("nonexistent"))
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestBadgerStorage_BatchWrite(t *testing.T) {
	dir, err := os.MkdirTemp("", "badger-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)
	
	s, err := NewBadgerStorage(dir)
	if err != nil {
		t.Fatalf("NewBadgerStorage failed: %v", err)
	}
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

func TestBadgerRaftStorage_State(t *testing.T) {
	dir, err := os.MkdirTemp("", "badger-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)
	
	s, err := NewBadgerRaftStorage(dir)
	if err != nil {
		t.Fatalf("NewBadgerRaftStorage failed: %v", err)
	}
	defer s.Close()
	
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

func TestBadgerRaftStorage_Snapshot(t *testing.T) {
	dir, err := os.MkdirTemp("", "badger-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)
	
	s, err := NewBadgerRaftStorage(dir)
	if err != nil {
		t.Fatalf("NewBadgerRaftStorage failed: %v", err)
	}
	defer s.Close()
	
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

func TestBadgerRaftStorage_Persistence(t *testing.T) {
	dir, err := os.MkdirTemp("", "badger-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)
	
	s1, err := NewBadgerRaftStorage(dir)
	if err != nil {
		t.Fatalf("NewBadgerRaftStorage failed: %v", err)
	}
	
	state := &RaftState{
		CurrentTerm: 10,
		VotedFor:    2,
		Log:         []byte("log-data"),
	}
	
	if err := s1.SaveRaftState(state); err != nil {
		t.Fatalf("SaveRaftState failed: %v", err)
	}
	s1.Close()
	
	s2, err := NewBadgerRaftStorage(dir)
	if err != nil {
		t.Fatalf("NewBadgerRaftStorage failed on reopen: %v", err)
	}
	defer s2.Close()
	
	loaded, err := s2.LoadRaftState()
	if err != nil {
		t.Fatalf("LoadRaftState failed: %v", err)
	}
	
	if loaded.CurrentTerm != state.CurrentTerm {
		t.Errorf("expected term %d, got %d", state.CurrentTerm, loaded.CurrentTerm)
	}
}

func TestBadgerStorage_Sync(t *testing.T) {
	dir, err := os.MkdirTemp("", "badger-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)
	
	s, err := NewBadgerStorage(dir)
	if err != nil {
		t.Fatalf("NewBadgerStorage failed: %v", err)
	}
	defer s.Close()
	
	if err := s.Set([]byte("key"), []byte("value")); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	
	if err := s.Sync(); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
}

func TestBadgerStorage_DataDir(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "badger-test-datadir")
	defer os.RemoveAll(dir)
	
	s, err := NewBadgerStorage(dir)
	if err != nil {
		t.Fatalf("NewBadgerStorage failed: %v", err)
	}
	defer s.Close()
	
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("data directory should exist")
	}
}
