# Multi-Raft

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**A Production-Ready Raft Distributed Consensus Protocol Implementation in Go**

English | [简体中文](README.md)

</div>

---

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [API Documentation](#api-documentation)
- [Testing](#testing)
- [Performance](#performance)
- [FAQ](#faq)
- [Contributing](#contributing)
- [License](#license)
- [Acknowledgments](#acknowledgments)

---

## Overview

Multi-Raft is a production-ready Raft distributed consensus protocol implementation in Go. This project originated from the MIT 6.824 Distributed Systems course and has been fully refactored and optimized with real network communication and persistent storage capabilities, ready for production use.

### Design Goals

- **Production Ready**: Real network communication and persistent storage
- **Extensible**: Modular design with pluggable transport and storage layers
- **High Performance**: Optimized concurrency control and memory management
- **Well Tested**: Comprehensive unit tests with race detection

---

## Features

| Feature | Description | Status |
|---------|-------------|--------|
| Leader Election | Randomized election timeout, vote granting mechanism | ✅ |
| Log Replication | Fast conflict rollback, batch commit optimization | ✅ |
| Snapshot | Log compaction, InstallSnapshot RPC | ✅ |
| Persistent Storage | Badger KV store, state persistence | ✅ |
| Network Communication | gRPC transport layer, connection pool management | ✅ |
| Concurrency Safe | Race detection passed, no data races | ✅ |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                          Application                             │
│                    (State Machine Interface)                     │
└─────────────────────────────────────────────────────────────────┘
                              ▲
                              │ ApplyMsg
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                           Raft Node                              │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐       │
│  │    Election   │  │  Replication  │  │   Snapshot    │       │
│  │    Ticker     │  │    Ticker     │  │    Handler    │       │
│  └───────────────┘  └───────────────┘  └───────────────┘       │
│  ┌───────────────────────────────────────────────────────┐     │
│  │                      Raft Log                          │     │
│  │              (In-Memory with Snapshot)                 │     │
│  └───────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────┘
         │                                      │
         ▼                                      ▼
┌─────────────────────┐              ┌─────────────────────┐
│      Transport      │              │       Storage       │
│   ┌─────────────┐   │              │   ┌─────────────┐   │
│   │    gRPC     │   │              │   │   Badger    │   │
│   │  Transport  │   │              │   │   Storage   │   │
│   └─────────────┘   │              │   └─────────────┘   │
│   ┌─────────────┐   │              │   ┌─────────────┐   │
│   │   Inmem     │   │              │   │   Memory    │   │
│   │  Transport  │   │              │   │   Storage   │   │
│   └─────────────┘   │              │   └─────────────┘   │
└─────────────────────┘              └─────────────────────┘
```

---

## Project Structure

```
Multi-Raft/
├── README.md                 # Documentation (Chinese)
├── README_EN.md              # Documentation (English)
├── LICENSE                   # License
└── src/
    ├── go.mod                # Go module definition
    ├── go.sum                # Dependency lock
    ├── raft/                 # Raft core implementation
    │   ├── node.go           # Raft node
    │   ├── raft_log.go       # Log management
    │   ├── transport.go      # Transport interface
    │   ├── transport_grpc.go # gRPC transport implementation
    │   ├── transport_inmem.go# In-memory transport (for testing)
    │   ├── storage.go        # Storage interface
    │   ├── storage_badger.go # Badger storage implementation
    │   ├── storage_memory.go # Memory storage (for testing)
    │   ├── config.go         # Configuration management
    │   └── pb/               # Protocol Buffers
    │       ├── raft.proto    # Protocol definition
    │       ├── raft.pb.go    # Generated code
    │       └── raft_grpc.pb.go
    ├── kvraft/               # Linearizable KV service
    ├── shardctrler/          # Shard controller
    ├── shardkv/              # Sharded KV service
    ├── labgob/               # Custom encoding
    ├── labrpc/               # RPC framework
    ├── models/               # Data models
    └── porcupine/            # Linearizability checker
```

---

## Installation

### Requirements

- Go 1.22 or higher
- Protocol Buffers compiler (optional, for regenerating proto)

### Steps

```bash
# Clone the repository
git clone https://github.com/LANSGANBS/Multi-Raft.git
cd Multi-Raft/src

# Install dependencies
go mod tidy

# Verify installation
go build ./...
```

### Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| github.com/dgraph-io/badger/v4 | v4.5.1 | Persistent storage |
| google.golang.org/grpc | v1.70.0 | Network communication |
| google.golang.org/protobuf | v1.36.4 | Serialization |

---

## Quick Start

### Basic Example

```go
package main

import (
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/LANSGANBS/Multi-Raft/src/raft"
)

func main() {
    // 1. Create persistent storage
    storage, err := raft.NewBadgerRaftStorage("/tmp/raft-node-0")
    if err != nil {
        panic(err)
    }
    defer storage.Close()

    // 2. Create gRPC transport
    transport := raft.NewGrpcTransport(&raft.TransportConfig{
        Peers:      []string{"localhost:50051", "localhost:50052", "localhost:50053"},
        PeerID:     0,
        ListenAddr: "localhost:50051",
    })

    // 3. Create apply channel
    applyCh := make(chan raft.ApplyMsg, 100)

    // 4. Create Raft node
    config := raft.DefaultConfig()
    node := raft.NewNode(config, transport, storage, applyCh)

    // 5. Start transport
    if err := transport.Start(); err != nil {
        panic(err)
    }
    defer node.Stop()

    // 6. Handle committed messages
    go func() {
        for msg := range applyCh {
            if msg.CommandValid {
                fmt.Printf("[Apply] Index: %d, Command: %v\n", msg.CommandIndex, msg.Command)
            } else if msg.SnapshotValid {
                fmt.Printf("[Snapshot] Index: %d\n", msg.SnapshotIndex)
            }
        }
    }()

    // 7. Submit commands (leader only)
    go func() {
        ticker := time.NewTicker(2 * time.Second)
        defer ticker.Stop()
        for range ticker.C {
            if _, isLeader := node.GetState(); isLeader {
                cmd := fmt.Sprintf("command-%d", time.Now().UnixNano())
                if index, term, ok := node.Start(cmd); ok {
                    fmt.Printf("[Submit] Index: %d, Term: %d\n", index, term)
                }
            }
        }
    }()

    // 8. Wait for shutdown signal
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh
    fmt.Println("\n[Shutdown] Stopping...")
}
```

### Start a Cluster

```bash
# Terminal 1 - Node 0
go run example/main.go 0

# Terminal 2 - Node 1
go run example/main.go 1

# Terminal 3 - Node 2
go run example/main.go 2
```

---

## Configuration

### Config Struct

```go
type Config struct {
    ElectionTimeoutMin time.Duration  // Min election timeout (default 250ms)
    ElectionTimeoutMax time.Duration  // Max election timeout (default 400ms)
    ReplicateInterval  time.Duration  // Log replication interval (default 30ms)
}
```

### TransportConfig Struct

```go
type TransportConfig struct {
    Peers      []string  // All node addresses
    PeerID     int       // Current node ID
    ListenAddr string    // Listen address
}
```

### Custom Configuration

```go
config := &raft.Config{
    ElectionTimeoutMin: 300 * time.Millisecond,
    ElectionTimeoutMax: 500 * time.Millisecond,
    ReplicateInterval:  50 * time.Millisecond,
}
node := raft.NewNode(config, transport, storage, applyCh)
```

---

## API Documentation

### Node Interface

#### Create Node

```go
func NewNode(config *Config, transport Transport, storage RaftStorage, applyCh chan ApplyMsg) *Node
```

#### Get State

```go
func (n *Node) GetState() (term int, isLeader bool)
```

#### Submit Command

```go
func (n *Node) Start(command interface{}) (index int, term int, isLeader bool)
```

#### Snapshot

```go
func (n *Node) Snapshot(index int, snapshot []byte)
```

#### Lifecycle

```go
func (n *Node) Kill()   // Mark node as stopped
func (n *Node) Stop()   // Graceful shutdown
```

### Transport Interface

```go
type Transport interface {
    GetPeerCount() int
    GetPeerID() int
    GetEndpoint(peerID int) RPCEndpoint
    RegisterHandler(handler RPCHandler)
    Start() error
    Stop()
}
```

### Storage Interface

```go
type Storage interface {
    Get(key []byte) ([]byte, error)
    Set(key []byte, value []byte) error
    Delete(key []byte) error
    Has(key []byte) (bool, error)
    BatchWrite(writes []Write) error
    Sync() error
    Close() error
}
```

### RaftStorage Interface

```go
type RaftStorage interface {
    SaveRaftState(state *RaftState) error
    LoadRaftState() (*RaftState, error)
    SaveSnapshot(snapshot *Snapshot) error
    LoadSnapshot() (*Snapshot, error)
    RaftStateSize() int
    SnapshotSize() int
    Storage() Storage
}
```

### ApplyMsg Struct

```go
type ApplyMsg struct {
    CommandValid   bool          // Is command
    Command        interface{}   // Command content
    CommandIndex   int           // Command index

    SnapshotValid  bool          // Is snapshot
    Snapshot       []byte        // Snapshot data
    SnapshotTerm   int           // Snapshot term
    SnapshotIndex  int           // Snapshot index
}
```

---

## Testing

### Run Tests

```bash
# Run all tests
go test -v ./raft/...

# Enable race detection
go test -race -v ./raft/...

# Run specific test
go test -v ./raft -run TestNode_ElectionWithMultipleNodes

# Parallel tests
go test -race -parallel 8 -v ./raft/...
```

### Test Coverage

| Category | Count | Coverage |
|----------|-------|----------|
| Node Tests | 23 | Election, log replication, snapshot, persistence |
| Storage Tests | 15 | Badger/Memory storage functionality |
| Transport Tests | 8 | gRPC/Inmem transport functionality |

---

## Performance

### Optimizations

1. **goroutine Lifecycle Management**
   - Use `sync.WaitGroup` for proper exit
   - Avoid resource leaks

2. **Channel Buffering**
   - All channels have reasonable buffer sizes
   - Avoid blocking

3. **Memory Allocation**
   - Pre-allocate slice capacity
   - Reduce memory copies

4. **Concurrency Control**
   - Fine-grained locking
   - Avoid deadlocks

### Metrics

| Metric | Value |
|--------|-------|
| Election Latency | 250-400ms |
| Log Replication Latency | ~30ms |
| Race Detection | Passed |

---

## FAQ

### Q: How to handle network partitions?

A: The Raft protocol automatically handles network partitions. After a partition, minority nodes cannot obtain majority votes and will remain as Followers. When the partition heals, nodes will automatically synchronize logs.

### Q: How to implement a state machine?

A: Listen to the `applyCh` channel and apply committed commands to your state machine:

```go
for msg := range applyCh {
    if msg.CommandValid {
        stateMachine.Apply(msg.Command)
    }
}
```

### Q: How to trigger a snapshot?

A: When the log becomes too large, call the `Snapshot` method:

```go
snapshot := stateMachine.Snapshot()
node.Snapshot(lastAppliedIndex, snapshot)
```

---

## Contributing

We welcome all forms of contributions!

### Process

1. Fork this repository
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Create a Pull Request

### Code Standards

- Follow Go official code standards
- Add tests for all new features
- Ensure race detection passes (`go test -race`)

### Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation update
- `test`: Test related
- `refactor`: Code refactoring

---

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---

## Acknowledgments

- [MIT 6.824](https://pdos.csail.mit.edu/6.824/) - Distributed Systems course
- [Raft Paper](https://raft.github.io/raft.pdf) - Raft protocol paper
- [Badger](https://github.com/dgraph-io/badger) - High-performance KV store

---

<div align="center">

**⭐ If this project helps you, please give it a Star! ⭐**

</div>
