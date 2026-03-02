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
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LANSGANBS/Multi-Raft/src/raft"
)

func main() {
	// Create persistent storage
	storage, err := raft.NewBadgerRaftStorage("/tmp/raft-node-0")
	if err != nil {
		panic(err)
	}
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
	if err := transport.Start(); err != nil {
		panic(err)
	}
	defer node.Stop()

	// Handle applied messages in a separate goroutine
	go func() {
		for msg := range applyCh {
			if msg.CommandValid {
				// Apply command to your state machine here
				fmt.Printf("Applied command at index %d: %v\n", msg.CommandIndex, msg.Command)
			} else if msg.SnapshotValid {
				// Apply snapshot to your state machine here
				fmt.Printf("Applied snapshot at index %d\n", msg.SnapshotIndex)
			}
		}
	}()

	// Submit commands (leader only)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if term, isLeader := node.GetState(); isLeader {
				cmd := fmt.Sprintf("command-%d", time.Now().UnixNano())
				index, term, ok := node.Start(cmd)
				if ok {
					fmt.Printf("Submitted command at index %d, term %d\n", index, term)
				}
			} else {
				fmt.Printf("Follower in term %d\n", term)
			}
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	fmt.Println("\nShutting down...")
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
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LANSGANBS/Multi-Raft/src/raft"
)

func main() {
	// 创建持久化存储
	storage, err := raft.NewBadgerRaftStorage("/tmp/raft-node-0")
	if err != nil {
		panic(err)
	}
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
	if err := transport.Start(); err != nil {
		panic(err)
	}
	defer node.Stop()

	// 在单独的 goroutine 中处理已提交的消息
	go func() {
		for msg := range applyCh {
			if msg.CommandValid {
				// 在这里将命令应用到你的状态机
				fmt.Printf("应用命令到索引 %d: %v\n", msg.CommandIndex, msg.Command)
			} else if msg.SnapshotValid {
				// 在这里将快照应用到你的状态机
				fmt.Printf("应用快照到索引 %d\n", msg.SnapshotIndex)
			}
		}
	}()

	// 提交命令（仅领导者有效）
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if term, isLeader := node.GetState(); isLeader {
				cmd := fmt.Sprintf("command-%d", time.Now().UnixNano())
				index, term, ok := node.Start(cmd)
				if ok {
					fmt.Printf("提交命令到索引 %d, 任期 %d\n", index, term)
				}
			} else {
				fmt.Printf("跟随者，任期 %d\n", term)
			}
		}
	}()

	// 等待中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	fmt.Println("\n正在关闭...")
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
