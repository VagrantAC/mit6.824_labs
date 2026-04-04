# MIT 6.5840 分布式系统实验

## 项目概述

这是 **MIT 6.5840（原 6.824）分布式系统** 2026 年的实验代码。包含一系列用 Go 语言实现的核心分布式系统组件实验：

- **Lab 1** – MapReduce（基于 Go 插件的 Coordinator/Worker 模型）
- **Lab 2** – 键值存储服务（基于 RPC 的基础 KV 服务器）
- **Lab 3A–3D** – Raft 共识算法（领导者选举、日志复制、持久化、快照）
- **Lab 4A–4C** – 复制键值服务（基于 Raft + 复制状态机的 KV 服务）
- **Lab 5A–5C** – 分片键值服务（多组 KV + 分片控制器）

### 核心目录

| 目录 | 用途 |
|------|------|
| `src/raft1/` | Raft 共识算法实现（Lab 3 核心） |
| `src/kvraft1/` | 基于 Raft 的 KV 服务器（Lab 4），包含 `rsm/`（复制状态机） |
| `src/kvsrv1/` | 基础 KV 服务器（Lab 2），包含 `lock/` |
| `src/shardkv1/` | 分片 KV 服务（Lab 5），包含 `shardcfg/`、`shardctrler/`、`shardgrp/` |
| `src/mr/` | MapReduce 协调器/Worker 逻辑（Lab 1） |
| `src/mrapps/` | MapReduce 插件应用（wc、indexer、崩溃测试、性能测试） |
| `src/labrpc/` | 模拟 RPC 框架（支持网络分区、丢包等测试场景） |
| `src/labgob/` | Go 编/解码器封装（用于 labrpc） |
| `src/tester1/` | 测试基础设施（持久化、组管理、服务器、多路复用） |
| `src/models1/` | 共享数据模型（KV 类型定义） |
| `src/kvtest1/` | KV 测试工具和线性一致性检查（porcupine） |
| `src/raftapi/` | Raft API 接口定义 |
| `src/main/` | 各服务的入口程序 |

### 技术栈

- **语言：** Go（模块名：`6.5840`）
- **Go 版本：** 1.22+
- **依赖：** `github.com/anishathalye/porcupine`（线性一致性检查器）

---

## 构建与运行

### 前置条件

```bash
cd src
```

以下命令均需在 `src/` 目录下执行。

### 构建所有实验

```bash
make all
```

### 构建并测试单个实验

```bash
# Lab 1 – MapReduce
make mr
make mr RUN="-run Wc"   # 仅运行 Wc 测试

# Lab 2 – KV 服务
make kvsrv1

# Lab 3 – Raft
make raft1

# Lab 4 – Raft KV
make kvraft1

# Lab 5 – 分片 KV
make shardkv
```

### 仅构建二进制文件（不运行测试）

```bash
make kvsrv1-build
make raft1-build
make kvraft1-build
```

### 数据竞争检测

测试默认启用 `-race` 参数。在旧版 macOS Go 版本（1.17.0–1.17.5）中，由于已知崩溃问题，`-race` 会被自动禁用。

---

## 提交作业

通过顶层 `Makefile` 创建提交文件：

```bash
make lab1   # 生成 lab1-handin.tar.gz
make lab2   # 生成 lab2-handin.tar.gz
make lab3a  # 生成 lab3a-handin.tar.gz
# ... 同理 lab3b, lab3c, lab3d, lab4a, lab4b, lab4c, lab5a, lab5b, lab5c
```

该命令会先执行构建检查（确保代码在测试文件还原为原始版本时仍能编译），然后生成 `.tar.gz` 压缩包，需手动上传至 Gradescope。

---

## 开发规范

- **待填充代码：** 源文件中包含 `// Your code here` 和 `// Your definitions here` 标记，指示学生需要编写代码的位置。
- **测试：** 每个实验都有 `*_test.go` 自动化测试文件。使用 `RUN="-run 测试名"` 可运行特定测试。
- **数据竞争安全：** 默认启用 `-race` 检测；确保正确使用同步机制（互斥锁、channel 等）。
- **不要修改测试文件：** 提交检查器会验证测试相关文件是否与原始版本一致。
- **实验框架代码 intentionally 极简：** 服务器构造函数、RPC 处理器和核心逻辑返回占位值或 `nil` —— 学生需实现完整逻辑。
- **超时设置：** Lab 5（shardkv）测试超时为 15 分钟；其他实验使用默认超时。

---

## 架构说明

### Raft（Lab 3）
- 定义在 `src/raft1/raft.go` 中的 `Raft` 结构体。
- 使用 `labrpc` 模拟节点间的 RPC 通信。
- 持久化器（`tester1.Persister`）处理状态持久存储。
- `raftapi` 包定义了 Raft 向上层暴露的接口。

### 复制状态机（Lab 4）
- `src/kvraft1/rsm/` 提供连接 Raft 与 KV 服务器的 RSM 层。
- `KVServer`（位于 `src/kvraft1/server.go`）通过 `kv.rsm.Submit()` 提交操作。
- 支持 `Get`、`Put`、`Append` 操作，保证线性一致性语义。

### MapReduce（Lab 1）
- 协调器在 `src/mr/coordinator.go` 中调度 map/reduce 任务。
- Worker 在 `src/mr/worker.go` 中执行任务并向协调器注册。
- `src/mrapps/` 中的插件编译为 Go 共享库（`-buildmode=plugin`）。
