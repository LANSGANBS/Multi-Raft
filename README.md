# Multi-Raft

[English](#english) | [简体中文](#简体中文)

---

<a name="english"></a>

## English

A production-ready Raft distributed consensus protocol implementation in Go.

### Features

- **Real Network Communication**: gRPC-based transport layer
- **Persistent Storage**: Badger KV store for durable state persistence
- **Pluggable Architecture**: Abstract interfaces for Transport and Storage
- **Full Raft Features**: Leader election, log replication, snapshot

### Project Structure

```
src/
├── raft/           # Core Raft implementation
│   ├── node.go     # Raft node
│   ├── transport/  # gRPC transport
│   ├── storage/    # Badger storage
│   └── pb/         # Protocol buffers
├── kvraft/         # Key-value service
├── shardctrler/    # Shard controller
├── shardkv/        # Sharded key-value service
├── labgob/         # Custom encoding
├── labrpc/         # Lab RPC framework
├── models/         # Data models
└── porcupine/      # Linearizability checker
```

### Quick Start

```go
package main

import (
    "github.com/LANSGANBS/Multi-Raft/src/raft"
)

func main() {
    storage, _ := raft.NewBadgerRaftStorage("/data/raft-node-0")
    defer storage.Close()

    transport := raft.NewGrpcTransport(&raft.TransportConfig{
        Peers:      []string{"localhost:50051", "localhost:50052", "localhost:50053"},
        PeerID:     0,
        ListenAddr: "localhost:50051",
    })

    applyCh := make(chan raft.ApplyMsg, 100)
    config := raft.DefaultConfig()
    node := raft.NewNode(config, transport, storage, applyCh)

    transport.Start()
    defer node.Stop()

    for msg := range applyCh {
        if msg.CommandValid {
            applyToStateMachine(msg.Command)
        }
    }
}
```

### Installation

```bash
git clone https://github.com/LANSGANBS/Multi-Raft.git
cd Multi-Raft/src
go mod tidy
```

### Testing

```bash
go test -race -v ./raft/...
```

---

<a name="简体中文"></a>

## 简体中文

生产级的 Raft 分布式一致性协议 Go 语言实现。

### 特性

- **真实网络通信**：基于 gRPC 的传输层
- **持久化存储**：使用 Badger KV 存储实现状态持久化
- **可插拔架构**：抽象的 Transport 和 Storage 接口
- **完整 Raft 功能**：领导者选举、日志复制、快照

### 项目结构

```
src/
├── raft/           # Raft 核心实现
│   ├── node.go     # Raft 节点
│   ├── transport/  # gRPC 传输层
│   ├── storage/    # Badger 存储
│   └── pb/         # Protocol buffers
├── kvraft/         # 键值服务
├── shardctrler/    # 分片控制器
├── shardkv/        # 分片键值服务
├── labgob/         # 自定义编码
├── labrpc/         # 实验室 RPC 框架
├── models/         # 数据模型
└── porcupine/      # 线性一致性检查器
```

### 快速开始

```go
package main

import (
    "github.com/LANSGANBS/Multi-Raft/src/raft"
)

func main() {
    storage, _ := raft.NewBadgerRaftStorage("/data/raft-node-0")
    defer storage.Close()

    transport := raft.NewGrpcTransport(&raft.TransportConfig{
        Peers:      []string{"localhost:50051", "localhost:50052", "localhost:50053"},
        PeerID:     0,
        ListenAddr: "localhost:50051",
    })

    applyCh := make(chan raft.ApplyMsg, 100)
    config := raft.DefaultConfig()
    node := raft.NewNode(config, transport, storage, applyCh)

    transport.Start()
    defer node.Stop()

    for msg := range applyCh {
        if msg.CommandValid {
            applyToStateMachine(msg.Command)
        }
    }
}
```

### 安装

```bash
git clone https://github.com/LANSGANBS/Multi-Raft.git
cd Multi-Raft/src
go mod tidy
```

### 运行测试

```bash
go test -race -v ./raft/...
```

### License

MIT License
