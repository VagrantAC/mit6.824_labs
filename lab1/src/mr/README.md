# 🗺️ MapReduce 系统架构文档

## 📑 目录
- [🏗️ 系统概览](#系统概览)
- [🎯 核心组件](#核心组件)
- [🔄 完整链路图](#完整链路图)
- [📊 数据流转过程](#数据流转过程)
- [⚙️ 关键特性](#关键特性)
- [🔧 故障处理](#故障处理)

---

## 🏗️ 系统概览

MapReduce 系统由以下核心组件组成：

```mermaid
graph TB
    subgraph "MapReduce System"
        C[🎛️ Coordinator]
        W[👷 Worker]
        
        subgraph "Coordinator Components"
            MTM[📋 MapTaskManager]
            RTM[📋 ReduceTaskManager]
        end
        
        subgraph "Worker Components"
            MT[🗺️ Map Task]
            RT[📉 Reduce Task]
        end
    end
    
    C <-->|RPC| W
    C --> MTM
    C --> RTM
    W --> MT
    W --> RT
    
    style C fill:#e1f5ff
    style W fill:#fff4e1
    style MTM fill:#e8f5e9
    style RTM fill:#e8f5e9
    style MT fill:#f3e5f5
    style RT fill:#f3e5f5
```

### 📋 核心组件职责

| 🎛️ 组件 | 📝 职责 | 🔧 主要方法 |
|----------|--------|------------|
| **Coordinator** | 任务调度、状态管理、RPC 服务 | `AssignTask`, `ReportMapTask`, `ReportReduceTask`, `Done` |
| **Worker** | 执行 Map/Reduce 任务、与 Coordinator 通信 | `Worker`, `mapTask`, `reduceTask` |
| **MapTaskManager** | 管理 Map 任务的生命周期 | `GetPendingTask`, `AssignTask`, `CompleteTask` |
| **ReduceTaskManager** | 管理 Reduce 任务的生命周期 | `GetPendingTask`, `AssignTask`, `CompleteTask`, `AddFile` |

---

## 🎯 核心组件

### 🎛️ Coordinator

**📌 核心职责**
- 🎯 任务调度和分配
- 📊 任务状态跟踪
- 👷 Worker 注册和管理
- 🌐 RPC 服务端

**🔄 主要工作流程**

1. **👷 Worker 注册**
```mermaid
sequenceDiagram
    participant W as 👷 Worker
    participant C as 🎛️ Coordinator
    
    W->>C: Register()
    C->>C: 分配 WorkerID
    C-->>W: 返回 WorkerID
```

2. **📋 任务分配**
```mermaid
sequenceDiagram
    participant W as 👷 Worker
    participant C as 🎛️ Coordinator
    participant MTM as 📋 MapTaskManager
    participant RTM as 📋 ReduceTaskManager
    
    W->>C: AssignTask()
    C->>MTM: GetPendingTask()
    alt 有 Map 任务
        MTM-->>C: MapTask
        C->>MTM: AssignTask()
        C-->>W: 返回 MapTask
    else Map 任务完成
        C->>RTM: GetPendingTask()
        alt 有 Reduce 任务
            RTM-->>C: ReduceTask
            C->>RTM: AssignTask()
            C-->>W: 返回 ReduceTask
        else 所有任务完成
            C-->>W: 返回空任务
        end
    end
```

3. **✅ 任务报告**
```mermaid
sequenceDiagram
    participant W as 👷 Worker
    participant C as 🎛️ Coordinator
    participant MTM as 📋 MapTaskManager
    participant RTM as 📋 ReduceTaskManager
    
    W->>C: ReportMapTask(MapTask, ReduceTaskMap)
    C->>MTM: IsTaskCompleted()
    alt 任务未完成
        C->>MTM: CompleteTask()
        loop 每个 Reduce 任务
            C->>RTM: AddFile(taskID, filename)
        end
    end
    C-->>W: 返回成功
```

---

### 👷 Worker

**📌 核心职责**
- 🗺️ 执行 Map 任务
- 📉 执行 Reduce 任务
- 🌐 与 Coordinator 通信
- 🔄 处理任务失败和重试

**🔄 主要工作流程**

1. **🗺️ Map 任务执行**
```mermaid
flowchart TD
    A[📥 接收 MapTask] --> B[📖 读取输入文件]
    B --> C[⚙️ 调用 Map 函数]
    C --> D[📊 生成键值对]
    D --> E[🔄 按 Reduce ID 分组]
    E --> F[💾 写入中间文件]
    F --> G[📤 报告任务完成]
    G --> H{✅ 成功?}
    H -->|是| I[✅ 返回成功]
    H -->|否| J[❌ 返回失败]
    
    style A fill:#e1f5ff
    style B fill:#fff4e1
    style C fill:#e8f5e9
    style D fill:#f3e5f5
    style E fill:#ffe8e8
    style F fill:#e8ffe8
    style G fill:#fff4e1
    style I fill:#e8ffe8
    style J fill:#ffe8e8
```

2. **📉 Reduce 任务执行**
```mermaid
flowchart TD
    A[📥 接收 ReduceTask] --> B[📖 读取中间文件]
    B --> C[🔄 合并相同 Key]
    C --> D[⚙️ 调用 Reduce 函数]
    D --> E[📝 生成输出]
    E --> F[💾 写入输出文件]
    F --> G[📤 报告任务完成]
    G --> H{✅ 成功?}
    H -->|是| I[✅ 返回成功]
    H -->|否| J[❌ 返回失败]
    
    style A fill:#e1f5ff
    style B fill:#fff4e1
    style C fill:#e8f5e9
    style D fill:#f3e5f5
    style E fill:#ffe8e8
    style F fill:#e8ffe8
    style G fill:#fff4e1
    style I fill:#e8ffe8
    style J fill:#ffe8e8
```

---

### 📋 MapTaskManager

**📌 核心职责**
- 🗺️ 管理 Map 任务的生命周期
- 📊 跟踪任务状态 (Pending/Assigned/Completed)
- ⏰ 处理任务超时和重新分配
- 📈 提供任务统计信息

**🔄 任务状态流转**
```
Pending → Assigned → Completed
    ↑         ↓
    └──── 超时重新分配 ────┘
```

**⏰ 超时机制**
- 任务分配后，如果 10 秒内未完成，视为超时
- 超时任务可以被重新分配给其他 Worker
- 防止 Worker 崩溃导致任务永久卡住

---

### 📋 ReduceTaskManager

**📌 核心职责**
- 📉 管理 Reduce 任务的生命周期
- 📊 跟踪任务状态
- 📁 管理中间文件
- ⏰ 处理任务超时和重新分配
- 📈 提供任务统计信息

**🔄 任务状态流转**
```
Pending → Assigned → Completed
    ↑         ↓
    └──── 超时重新分配 ────┘
```

---

## 🔄 完整链路图

### 🚀 系统启动流程

```mermaid
sequenceDiagram
    participant C as 🎛️ Coordinator
    participant W as 👷 Worker
    
    Note over C: 🚀 Coordinator 启动
    C->>C: MakeCoordinator(sockname, files, nReduce)
    C->>C: 创建 MapTaskManager
    C->>C: 创建 ReduceTaskManager
    C->>C: 启动 RPC 服务器
    
    Note over W: 🚀 Worker 启动
    W->>W: Worker(sockname, mapf, reducef)
    W->>C: Register()
    C-->>W: 返回 WorkerID
    W->>W: 进入主循环
    
    loop Worker 主循环
        W->>C: AssignTask()
        C-->>W: 返回任务
        W->>W: 执行任务
        W->>C: ReportTask()
        C-->>W: 返回成功
        W->>C: IsDone()
        C-->>W: 返回完成状态
        alt 任务完成
            W->>W: 退出循环
        end
    end
```

### 🗺️ Map 任务执行流程

```mermaid
sequenceDiagram
    participant W as 👷 Worker
    participant C as 🎛️ Coordinator
    participant MTM as 📋 MapTaskManager
    participant RTM as 📋 ReduceTaskManager
    
    W->>C: AssignTask()
    C->>MTM: GetPendingTask()
    MTM-->>C: MapTask
    C->>MTM: AssignTask()
    C-->>W: 返回 MapTask
    
    Note over W: 执行 mapTask()
    W->>W: 读取输入文件
    W->>W: 调用 mapf()
    W->>W: 生成键值对
    W->>W: 分组到 Reduce 任务
    W->>W: 写入中间文件
    
    W->>C: ReportMapTask(ReduceTaskMap)
    C->>MTM: IsTaskCompleted()
    alt 任务未完成
        C->>MTM: CompleteTask()
        loop 每个 Reduce 任务
            C->>RTM: AddFile(taskID, filename)
        end
    end
    C-->>W: 返回成功
```

### 📉 Reduce 任务执行流程

```mermaid
sequenceDiagram
    participant W as 👷 Worker
    participant C as 🎛️ Coordinator
    participant RTM as 📋 ReduceTaskManager
    
    W->>C: AssignTask()
    C->>RTM: GetPendingTask()
    RTM-->>C: ReduceTask
    C->>RTM: AssignTask()
    C-->>W: 返回 ReduceTask
    
    Note over W: 执行 reduceTask()
    W->>W: 读取所有中间文件
    W->>W: 合并相同 Key 的值
    W->>W: 调用 reducef()
    W->>W: 写入输出文件
    
    W->>C: ReportReduceTask(Filename)
    C->>RTM: IsTaskCompleted()
    alt 任务未完成
        C->>RTM: CompleteTask()
    end
    C-->>W: 返回成功
```

### ⏰ 任务超时和重新分配流程

```mermaid
sequenceDiagram
    participant WA as 👷 Worker A
    participant WB as 👷 Worker B
    participant C as 🎛️ Coordinator
    participant MTM as 📋 MapTaskManager
    
    WA->>C: AssignTask()
    C->>MTM: GetPendingTask()
    MTM-->>C: MapTask
    C->>MTM: AssignTask()
    Note over MTM: 状态: Pending → Assigned<br/>时间: T0
    C-->>WA: 返回 MapTask
    
    Note over WA: 💥 Worker A 崩溃
    
    Note over MTM: ⏰ 时间: T0 + 10秒
    
    WB->>C: AssignTask()
    C->>MTM: GetPendingTask()
    Note over MTM: 检查 Pending 任务<br/>无 Pending 任务<br/>检查超时任务<br/>发现超时任务
    MTM-->>C: 超时 MapTask
    C->>MTM: AssignTask()
    Note over MTM: 状态: Assigned (保持)<br/>时间: T0 + 10秒 (更新)
    C-->>WB: 返回超时任务
    
    Note over WB: ✅ Worker B 完成任务
    WB->>C: ReportMapTask()
    C->>MTM: CompleteTask()
    Note over MTM: 状态: Assigned → Completed
    C-->>WB: 返回成功
```

---

## 📊 数据流转过程

### 🔄 完整数据流示例

假设有 2 个输入文件和 3 个 Reduce 任务：

```mermaid
graph TB
    subgraph "📁 输入文件"
        F1[📄 pg-huckleberry_finn.txt]
        F2[📄 pg-moby_dick.txt]
    end
    
    subgraph "🗺️ Map 阶段"
        W1[👷 Worker 1]
        W2[👷 Worker 2]
    end
    
    subgraph "📁 中间文件"
        subgraph "Worker 1 生成"
            M10[mr-1-0]
            M11[mr-1-1]
            M12[mr-1-2]
        end
        subgraph "Worker 2 生成"
            M20[mr-2-0]
            M21[mr-2-1]
            M22[mr-2-2]
        end
    end
    
    subgraph "📉 Reduce 阶段"
        R0[Reduce 0]
        R1[Reduce 1]
        R2[Reduce 2]
    end
    
    subgraph "📤 输出文件"
        O0[mr-out-0]
        O1[mr-out-1]
        O2[mr-out-2]
    end
    
    F1 --> W1
    F2 --> W2
    W1 --> M10
    W1 --> M11
    W1 --> M12
    W2 --> M20
    W2 --> M21
    W2 --> M22
    M10 --> R0
    M20 --> R0
    M11 --> R1
    M21 --> R1
    M12 --> R2
    M22 --> R2
    R0 --> O0
    R1 --> O1
    R2 --> O2
    
    style F1 fill:#e1f5ff
    style F2 fill:#e1f5ff
    style W1 fill:#fff4e1
    style W2 fill:#fff4e1
    style M10 fill:#e8f5e9
    style M11 fill:#e8f5e9
    style M12 fill:#e8f5e9
    style M20 fill:#e8f5e9
    style M21 fill:#e8f5e9
    style M22 fill:#e8f5e9
    style R0 fill:#f3e5f5
    style R1 fill:#f3e5f5
    style R2 fill:#f3e5f5
    style O0 fill:#e8ffe8
    style O1 fill:#e8ffe8
    style O2 fill:#e8ffe8
```

### 📊 数据处理流程

```mermaid
graph LR
    subgraph "📁 输入文件"
        F1[📄 pg-huckleberry_finn.txt]
        F2[📄 pg-moby_dick.txt]
    end
    
    subgraph "📁 中间文件"
        M1[📋 mr-1-0, mr-1-1, mr-1-2]
        M2[📋 mr-2-0, mr-2-1, mr-2-2]
    end
    
    subgraph "📤 输出文件"
        O1[📄 mr-out-0]
        O2[📄 mr-out-1]
        O3[📄 mr-out-2]
    end
    
    F1 --> M1
    F2 --> M2
    M1 --> O1
    M1 --> O2
    M1 --> O3
    M2 --> O1
    M2 --> O2
    M2 --> O3
    
    style F1 fill:#e1f5ff
    style F2 fill:#e1f5ff
    style M1 fill:#e8f5e9
    style M2 fill:#e8f5e9
    style O1 fill:#e8ffe8
    style O2 fill:#e8ffe8
    style O3 fill:#e8ffe8
```

### ⏱️ 并发处理示例

```mermaid
gantt
    title MapReduce 并发处理时间线
    dateFormat  HH:mm:ss
    axisFormat  %H:%M:%S
    
    section 🎛️ Coordinator
    启动           :a1, 20:34:00, 1m
    
    section 👷 Worker 1
    启动           :b1, 20:34:01, 30s
    🗺️ Map 任务    :b2, after b1, 2m
    📉 Reduce 任务  :b3, after b2, 1m
    
    section 👷 Worker 2
    启动           :c1, 20:34:02, 30s
    🗺️ Map 任务    :c2, after c1, 2m
    📉 Reduce 任务  :c3, after c2, 1m
    
    section 👷 Worker 3
    启动           :d1, 20:34:03, 30s
    📉 Reduce 任务  :d2, after d1, 1m
```

---

## ⚙️ 关键特性

### ⏰ 任务超时机制
- 任务分配后，如果 10 秒内未完成，自动重新分配
- 防止 Worker 崩溃导致任务永久卡住
- 确保系统的高可用性

### 🔄 任务去重
- 通过任务状态跟踪，避免重复执行
- 已完成的任务会被标记，不会再次分配

### 🌐 RPC 重试机制
- Worker 与 Coordinator 的 RPC 调用支持重试
- 最多重试 3 次，每次间隔 100ms
- 提高系统的容错能力

### 🔒 并发安全
- 使用 `sync.RWMutex` 保护共享数据
- Coordinator 和 Task Manager 都有适当的锁机制
- 确保多 Worker 并发访问时的数据一致性

### 📝 结构化日志
- 使用 `logrus` 提供结构化日志
- 支持日志级别：DEBUG, INFO, WARN, ERROR
- 包含丰富的上下文信息（service, component, worker_id 等）

### 🎨 函数式编程
- 使用 `github.com/samber/lo` 库简化代码
- 提供丰富的函数式工具（Find, Filter, Map, Reduce 等）
- 提高代码的可读性和可维护性

---

## 🔧 故障处理

### 💥 Worker 崩溃处理
- 任务超时机制会自动检测崩溃的 Worker
- 超时任务会被重新分配给其他 Worker
- 系统继续正常运行，无需人工干预

### 🎛️ Coordinator 崩溃处理
- 当前实现中，Coordinator 是单点故障
- 崩溃后需要重启整个系统
- 未来可以考虑实现 Coordinator 的高可用

### 🌐 网络故障处理
- RPC 调用失败会自动重试
- 最多重试 3 次，每次间隔 100ms
- 如果重试失败，Worker 会退出

### 💾 磁盘故障处理
- 中间文件和输出文件存储在本地磁盘
- 磁盘故障会导致任务失败
- 建议使用可靠的存储系统

---

## 📝 总结

MapReduce 系统是一个分布式计算框架，通过以下核心功能实现大规模数据处理：

- 🎯 **任务调度**: Coordinator 负责任务的分配和调度
- 🗺️ **Map 阶段**: Worker 并行处理输入文件，生成中间结果
- 📉 **Reduce 阶段**: Worker 合并中间结果，生成最终输出
- ⏰ **容错机制**: 任务超时、RPC 重试确保系统高可用
- 🔒 **并发安全**: 使用锁机制保护共享数据

系统采用现代化的编程实践，包括结构化日志、函数式编程等，确保代码的可读性和可维护性。