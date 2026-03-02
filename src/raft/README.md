# Raft Distributed Consensus Implementation

[English](#english) | [简体中文](#简体中文)

---

<a name="english"></a>

## English

### Overview

This is a production-ready Raft distributed consensus protocol implementation in Go, featuring:

- **Real Network Communication**: gRPC-based transport layer replacing simulated networks
- **Persistent Storage**: Badger KV store for durable state persistence
- **Pluggable Architecture**: Abstract interfaces for Transport and Storage layers
- **Full Raft Features**: Leader election, log replication, snapshot, and membership changes

### Features

| Feature | Description |
|---------|-------------|
| Leader Election | Randomized election timeout with vote granting |
| Log Replication | Conflict resolution with fast rollback |
| Snapshot | Log compaction and InstallSnapshot RPC |
| Persistence | State and snapshot persistence with Badger |
| Transport | gRPC-based network communication |

### Installation

```bash
cd src
go mod tidy
```

### Quick Start

```go
package main

import (
    "github.com/LANSGANBS/Multi-Raft/src/raft"
)

func main() {
    // Create persistent storage
    storage, _ := raft.NewBadgerRaftStorage("/data/raft-node-0")
    defer storage.Close()

    // Create gRPC transport
    transport := raft.NewGrpcTransport(&raft.TransportConfig{
        Peers:      []string{"localhost:50051", "localhost:50052", "localhost:50053"},
        PeerID:     0,
        ListenAddr: "localhost:50051",
    })

    // Create apply channel
    applyCh := make(chan raft.ApplyMsg, 100)

    // Create Raft node
    config := raft.DefaultConfig()
    node := raft.NewNode(config, transport, storage, applyCh)

    // Start transport
    transport.Start()

    // Handle applied messages
    go func() {
        for msg := range applyCh {
            if msg.CommandValid {
                // Apply command to state machine
                applyToStateMachine(msg.Command)
            }
        }
    }()

    // Submit commands (leader only)
    if index, term, ok := node.Start("my-command"); ok {
        fmt.Printf("Command submitted at index %d, term %d\n", index, term)
    }

    // Graceful shutdown
    node.Stop()
}
```

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                        Node                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │  Election   │  │ Replication │  │  Snapshot   │     │
│  │   Ticker    │  │   Ticker    │  │   Handler   │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
└─────────────────────────────────────────────────────────┘
           │                  │                  │
           ▼                  ▼                  ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│    Transport    │  │     Storage     │  │    RaftLog      │
│    (gRPC)       │  │    (Badger)     │  │   (In-Memory)   │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

### Interface Design

#### Transport Interface

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

#### Storage Interface

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

### Configuration

```go
config := &raft.Config{
    ElectionTimeoutMin: 250 * time.Millisecond,
    ElectionTimeoutMax: 400 * time.Millisecond,
    ReplicateInterval:  30 * time.Millisecond,
}
```

### Testing

```bash
# Run all tests
go test -v ./raft/...

# Run with race detection
go test -race -v ./raft/...

# Run specific tests
go test -v ./raft -run TestNode_ElectionWithMultipleNodes
```

### Project Structure

```
src/raft/
├── node.go                 # Core Raft node implementation
├── raft_log.go             # Log management
├── transport.go            # Transport interface
├── transport_grpc.go       # gRPC transport implementation
├── transport_inmem.go      # In-memory transport for testing
├── storage.go              # Storage interface
├── storage_badger.go       # Badger storage implementation
├── storage_memory.go       # Memory storage for testing
├── pb/
│   ├── raft.proto          # gRPC protocol definition
│   ├── raft.pb.go          # Generated protobuf code
│   └── raft_grpc.pb.go     # Generated gRPC code
└── example/
    └── main.go             # Usage example
```

---

<a name="简体中文"></a>

## 简体中文

### 概述

这是一个生产级的 Raft 分布式一致性协议 Go 语言实现，具有以下特点：

- **真实网络通信**：基于 gRPC 的传输层，替代模拟网络
- **持久化存储**：使用 Badger KV 存储实现状态持久化
- **可插拔架构**：抽象的 Transport 和 Storage 接口
- **完整 Raft 功能**：领导者选举、日志复制、快照、成员变更

### 特性

| 特性 | 描述 |
|------|------|
| 领导者选举 | 随机选举超时 + 投票授权 |
| 日志复制 | 冲突快速回溯解决 |
| 快照机制 | 日志压缩 + InstallSnapshot RPC |
| 持久化 | 基于 Badger 的状态和快照持久化 |
| 网络传输 | 基于 gRPC 的网络通信 |

### 安装

```bash
cd src
go mod tidy
```

### 快速开始

```go
package main

import (
    "github.com/LANSGANBS/Multi-Raft/src/raft"
)

func main() {
    // 创建持久化存储
    storage, _ := raft.NewBadgerRaftStorage("/data/raft-node-0")
    defer storage.Close()

    // 创建 gRPC 传输层
    transport := raft.NewGrpcTransport(&raft.TransportConfig{
        Peers:      []string{"localhost:50051", "localhost:50052", "localhost:50053"},
        PeerID:     0,
        ListenAddr: "localhost:50051",
    })

    // 创建应用通道
    applyCh := make(chan raft.ApplyMsg, 100)

    // 创建 Raft 节点
    config := raft.DefaultConfig()
    node := raft.NewNode(config, transport, storage, applyCh)

    // 启动传输层
    transport.Start()

    // 处理已提交的消息
    go func() {
        for msg := range applyCh {
            if msg.CommandValid {
                // 将命令应用到状态机
                applyToStateMachine(msg.Command)
            }
        }
    }()

    // 提交命令（仅领导者有效）
    if index, term, ok := node.Start("my-command"); ok {
        fmt.Printf("命令已提交，索引: %d, 任期: %d\n", index, term)
    }

    // 优雅关闭
    node.Stop()
}
```

### 架构设计

```
┌─────────────────────────────────────────────────────────┐
│                        Node                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │  选举定时器  │  │ 复制定时器  │  │  快照处理器  │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
└─────────────────────────────────────────────────────────┘
           │                  │                  │
           ▼                  ▼                  ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│    Transport    │  │     Storage     │  │    RaftLog      │
│    (gRPC)       │  │    (Badger)     │  │    (内存)       │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

### 接口设计

#### Transport 接口

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

#### Storage 接口

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

### 配置说明

```go
config := &raft.Config{
    ElectionTimeoutMin: 250 * time.Millisecond,  // 最小选举超时
    ElectionTimeoutMax: 400 * time.Millisecond,  // 最大选举超时
    ReplicateInterval:  30 * time.Millisecond,   // 日志复制间隔
}
```

### 运行测试

```bash
# 运行所有测试
go test -v ./raft/...

# 启用竞态检测
go test -race -v ./raft/...

# 运行特定测试
go test -v ./raft -run TestNode_ElectionWithMultipleNodes
```

### 项目结构

```
src/raft/
├── node.go                 # Raft 节点核心实现
├── raft_log.go             # 日志管理
├── transport.go            # 传输层接口
├── transport_grpc.go       # gRPC 传输实现
├── transport_inmem.go      # 内存传输（测试用）
├── storage.go              # 存储层接口
├── storage_badger.go       # Badger 存储实现
├── storage_memory.go       # 内存存储（测试用）
├── pb/
│   ├── raft.proto          # gRPC 协议定义
│   ├── raft.pb.go          # 生成的 protobuf 代码
│   └── raft_grpc.pb.go     # 生成的 gRPC 代码
└── example/
    └── main.go             # 使用示例
```

### 核心实现

#### 领导者选举

- 随机选举超时（250-400ms）
- 投票授权检查（日志新旧比较）
- 任期管理和状态转换

#### 日志复制

- PrevLogIndex/Term 一致性检查
- 冲突快速回溯优化
- 批量提交提高效率

#### 快照机制

- 日志压缩减少内存占用
- InstallSnapshot RPC 同步快照
- 边界检查保证数据完整性

### 最佳实践

1. **goroutine 生命周期管理**：使用 `sync.WaitGroup` 确保 goroutine 正确退出
2. **channel 缓冲**：所有 channel 都设置了合理的缓冲大小
3. **并发安全**：通过 `-race` 检测，无数据竞争
4. **资源释放**：`Stop()` 方法确保所有资源正确释放

### License

MIT License

### Contributing

欢迎提交 Issue 和 Pull Request！
