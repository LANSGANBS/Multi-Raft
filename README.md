# Multi-Raft

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/LANSGANBS/Multi-Raft)](https://goreportcard.com/report/github.com/LANSGANBS/Multi-Raft)

**生产级 Raft 分布式一致性协议 Go 语言实现**

[功能特性](#功能特性) • [快速开始](#快速开始) • [安装指南](#安装指南) • [API文档](#api文档) • [贡献指南](#贡献指南)

</div>

---

## 目录

- [项目概述](#项目概述)
- [功能特性](#功能特性)
- [架构设计](#架构设计)
- [项目结构](#项目结构)
- [安装指南](#安装指南)
- [快速开始](#快速开始)
- [配置说明](#配置说明)
- [API文档](#api文档)
- [测试说明](#测试说明)
- [性能优化](#性能优化)
- [常见问题](#常见问题)
- [贡献指南](#贡献指南)
- [许可证](#许可证)
- [致谢](#致谢)

---

## 项目概述

Multi-Raft 是一个生产级的 Raft 分布式一致性协议 Go 语言实现。本项目源自 MIT 6.824 分布式系统课程，经过全面重构和优化，提供了真实网络通信和持久化存储能力，可直接用于生产环境。

### 设计目标

- **生产可用**：提供真实网络通信和持久化存储
- **易于扩展**：模块化设计，支持自定义传输层和存储层
- **高性能**：优化的并发控制和内存管理
- **完整测试**：全面的单元测试和竞态检测

---

## 功能特性

| 特性 | 描述 | 状态 |
|------|------|------|
| 领导者选举 | 随机选举超时，投票授权机制 | ✅ |
| 日志复制 | 冲突快速回溯，批量提交优化 | ✅ |
| 快照机制 | 日志压缩，InstallSnapshot RPC | ✅ |
| 持久化存储 | Badger KV 存储，状态持久化 | ✅ |
| 网络通信 | gRPC 传输层，连接池管理 | ✅ |
| 并发安全 | 竞态检测通过，无数据竞争 | ✅ |

---

## 架构设计

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

## 项目结构

```
Multi-Raft/
├── README.md                 # 项目文档
├── LICENSE                   # 许可证
└── src/
    ├── go.mod                # Go 模块定义
    ├── go.sum                # 依赖锁定
    ├── raft/                 # Raft 核心实现
    │   ├── node.go           # Raft 节点
    │   ├── raft_log.go       # 日志管理
    │   ├── transport.go      # 传输层接口
    │   ├── transport_grpc.go # gRPC 传输实现
    │   ├── transport_inmem.go# 内存传输（测试用）
    │   ├── storage.go        # 存储层接口
    │   ├── storage_badger.go # Badger 存储实现
    │   ├── storage_memory.go # 内存存储（测试用）
    │   ├── config.go         # 配置管理
    │   └── pb/               # Protocol Buffers
    │       ├── raft.proto    # 协议定义
    │       ├── raft.pb.go    # 生成的代码
    │       └── raft_grpc.pb.go
    ├── kvraft/               # 线性一致性 KV 服务
    ├── shardctrler/          # 分片控制器
    ├── shardkv/              # 分片 KV 服务
    ├── labgob/               # 自定义编码
    ├── labrpc/               # RPC 框架
    ├── models/               # 数据模型
    └── porcupine/            # 线性一致性检查
```

---

## 安装指南

### 环境要求

- Go 1.22 或更高版本
- Protocol Buffers 编译器（可选，用于重新生成 proto）

### 安装步骤

```bash
# 克隆仓库
git clone https://github.com/LANSGANBS/Multi-Raft.git
cd Multi-Raft/src

# 安装依赖
go mod tidy

# 验证安装
go build ./...
```

### 依赖项

| 依赖 | 版本 | 用途 |
|------|------|------|
| github.com/dgraph-io/badger/v4 | v4.5.1 | 持久化存储 |
| google.golang.org/grpc | v1.70.0 | 网络通信 |
| google.golang.org/protobuf | v1.36.4 | 序列化 |

---

## 快速开始

### 基础示例

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
    // 1. 创建持久化存储
    storage, err := raft.NewBadgerRaftStorage("/tmp/raft-node-0")
    if err != nil {
        panic(err)
    }
    defer storage.Close()

    // 2. 创建 gRPC 传输层
    transport := raft.NewGrpcTransport(&raft.TransportConfig{
        Peers:      []string{"localhost:50051", "localhost:50052", "localhost:50053"},
        PeerID:     0,
        ListenAddr: "localhost:50051",
    })

    // 3. 创建应用通道
    applyCh := make(chan raft.ApplyMsg, 100)

    // 4. 创建 Raft 节点
    config := raft.DefaultConfig()
    node := raft.NewNode(config, transport, storage, applyCh)

    // 5. 启动传输层
    if err := transport.Start(); err != nil {
        panic(err)
    }
    defer node.Stop()

    // 6. 处理已提交的消息
    go func() {
        for msg := range applyCh {
            if msg.CommandValid {
                fmt.Printf("[Apply] Index: %d, Command: %v\n", msg.CommandIndex, msg.Command)
            } else if msg.SnapshotValid {
                fmt.Printf("[Snapshot] Index: %d\n", msg.SnapshotIndex)
            }
        }
    }()

    // 7. 提交命令（仅领导者有效）
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

    // 8. 等待退出信号
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh
    fmt.Println("\n[Shutdown] Stopping...")
}
```

### 启动集群

```bash
# 终端 1 - 节点 0
go run example/main.go 0

# 终端 2 - 节点 1
go run example/main.go 1

# 终端 3 - 节点 2
go run example/main.go 2
```

---

## 配置说明

### Config 结构体

```go
type Config struct {
    ElectionTimeoutMin time.Duration  // 最小选举超时（默认 250ms）
    ElectionTimeoutMax time.Duration  // 最大选举超时（默认 400ms）
    ReplicateInterval  time.Duration  // 日志复制间隔（默认 30ms）
}
```

### TransportConfig 结构体

```go
type TransportConfig struct {
    Peers      []string  // 所有节点地址
    PeerID     int       // 当前节点 ID
    ListenAddr string    // 监听地址
}
```

### 使用自定义配置

```go
config := &raft.Config{
    ElectionTimeoutMin: 300 * time.Millisecond,
    ElectionTimeoutMax: 500 * time.Millisecond,
    ReplicateInterval:  50 * time.Millisecond,
}
node := raft.NewNode(config, transport, storage, applyCh)
```

---

## API文档

### Node 接口

#### 创建节点

```go
func NewNode(config *Config, transport Transport, storage RaftStorage, applyCh chan ApplyMsg) *Node
```

#### 状态查询

```go
func (n *Node) GetState() (term int, isLeader bool)
```

#### 提交命令

```go
func (n *Node) Start(command interface{}) (index int, term int, isLeader bool)
```

#### 快照操作

```go
func (n *Node) Snapshot(index int, snapshot []byte)
```

#### 生命周期

```go
func (n *Node) Kill()   // 标记节点为停止状态
func (n *Node) Stop()   // 优雅关闭节点
```

### Transport 接口

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

### Storage 接口

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

### RaftStorage 接口

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

### ApplyMsg 结构体

```go
type ApplyMsg struct {
    CommandValid   bool          // 是否为命令
    Command        interface{}   // 命令内容
    CommandIndex   int           // 命令索引

    SnapshotValid  bool          // 是否为快照
    Snapshot       []byte        // 快照数据
    SnapshotTerm   int           // 快照任期
    SnapshotIndex  int           // 快照索引
}
```

---

## 测试说明

### 运行测试

```bash
# 运行所有测试
go test -v ./raft/...

# 启用竞态检测
go test -race -v ./raft/...

# 运行特定测试
go test -v ./raft -run TestNode_ElectionWithMultipleNodes

# 并行测试
go test -race -parallel 8 -v ./raft/...
```

### 测试覆盖

| 测试类别 | 测试数量 | 覆盖内容 |
|---------|---------|---------|
| 节点测试 | 23 | 选举、日志复制、快照、持久化 |
| 存储测试 | 15 | Badger/Memory 存储功能 |
| 传输测试 | 8 | gRPC/Inmem 传输功能 |

---

## 性能优化

### 已实施的优化

1. **goroutine 生命周期管理**
   - 使用 `sync.WaitGroup` 确保正确退出
   - 避免资源泄漏

2. **Channel 缓冲**
   - 所有 channel 设置合理缓冲大小
   - 避免阻塞

3. **内存分配**
   - 切片预分配容量
   - 减少内存拷贝

4. **并发控制**
   - 细粒度锁
   - 避免死锁

### 性能指标

| 指标 | 数值 |
|------|------|
| 选举延迟 | 250-400ms |
| 日志复制延迟 | ~30ms |
| 竞态检测 | 通过 |

---

## 常见问题

### Q: 如何处理网络分区？

A: Raft 协议自动处理网络分区。分区后，少数派节点无法获得多数票，将保持 Follower 状态。分区恢复后，节点将自动同步日志。

### Q: 如何实现状态机？

A: 监听 `applyCh` 通道，将已提交的命令应用到状态机：

```go
for msg := range applyCh {
    if msg.CommandValid {
        stateMachine.Apply(msg.Command)
    }
}
```

### Q: 如何触发快照？

A: 当日志过大时，调用 `Snapshot` 方法：

```go
snapshot := stateMachine.Snapshot()
node.Snapshot(lastAppliedIndex, snapshot)
```

---

## 贡献指南

我们欢迎所有形式的贡献！

### 贡献流程

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

### 代码规范

- 遵循 Go 官方代码规范
- 所有新功能需添加测试
- 确保通过竞态检测 (`go test -race`)

### 提交信息格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

类型：
- `feat`: 新功能
- `fix`: 修复 Bug
- `docs`: 文档更新
- `test`: 测试相关
- `refactor`: 代码重构

---

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

---

## 致谢

- [MIT 6.824](https://pdos.csail.mit.edu/6.824/) - 分布式系统课程
- [Raft Paper](https://raft.github.io/raft.pdf) - Raft 协议论文
- [Badger](https://github.com/dgraph-io/badger) - 高性能 KV 存储

---

<div align="center">

**⭐ 如果这个项目对你有帮助，请给一个 Star！⭐**

</div>
