# MIT 6.5840 Distributed Systems Lab

## Project Overview

This is the **MIT 6.5840 (formerly 6.824) Distributed Systems** lab codebase for the 2026 offering. It contains a series of labs implementing core distributed systems components in Go, including:

- **Lab 1** – MapReduce (coordinator/worker model with Go plugins)
- **Lab 2** – Key/Value Service (basic KV server with RPC)
- **Lab 3A–3D** – Raft Consensus Algorithm (leader election, log replication, persistence, snapshots)
- **Lab 4A–4C** – Replicated Key/Value Service (KV built on top of Raft via a Replicated State Machine)
- **Lab 5A–5C** – Sharded Key/Value Service (multi-group KV with a shard controller)

### Key Directories

| Directory | Purpose |
|-----------|---------|
| `src/raft1/` | Raft consensus implementation (core lab 3) |
| `src/kvraft1/` | Raft-backed KV server (lab 4), includes `rsm/` (replicated state machine) |
| `src/kvsrv1/` | Basic KV server (lab 2), includes `lock/` |
| `src/shardkv1/` | Sharded KV service (lab 5), includes `shardcfg/`, `shardctrler/`, `shardgrp/` |
| `src/mr/` | MapReduce coordinator/worker logic (lab 1) |
| `src/mrapps/` | MapReduce plugin apps (wc, indexer, crash tests, timing tests) |
| `src/labrpc/` | Simulated RPC framework for testing (network partitions, drops) |
| `src/labgob/` | Go encoder/decoder wrapper for labrpc |
| `src/tester1/` | Tester infrastructure (persister, group, servers, demux) |
| `src/models1/` | Shared data models (KV types) |
| `src/kvtest1/` | KV test utilities and linearizability checking (porcupine) |
| `src/raftapi/` | Raft API interface definitions |
| `src/main/` | Entry-point binaries for each service |

### Technologies

- **Language:** Go (module: `6.5840`)
- **Go version:** 1.22+
- **Dependencies:** `github.com/anishathalye/porcupine` (linearizability checker)

---

## Building and Running

### Prerequisites

```bash
cd src
```

All commands below should be run from the `src/` directory.

### Build All Labs

```bash
make all
```

### Build and Test Individual Labs

```bash
# Lab 1 – MapReduce
make mr
make mr RUN="-run Wc"   # run only the Wc test

# Lab 2 – KV Service
make kvsrv1

# Lab 3 – Raft
make raft1

# Lab 4 – Raft KV
make kvraft1

# Lab 5 – Sharded KV
make shardkv
```

### Build Binaries Only (no tests)

```bash
make kvsrv1-build
make raft1-build
make kvraft1-build
```

### Race Detection

Tests are run with `-race` by default. On older macOS Go versions (1.17.0–1.17.5), `-race` is automatically disabled due to known crashes.

---

## Submission

Submissions are created via the top-level `Makefile`:

```bash
make lab1   # produces lab1-handin.tar.gz
make lab2   # produces lab2-handin.tar.gz
make lab3a  # produces lab3a-handin.tar.gz
# ... etc for lab3b, lab3c, lab3d, lab4a, lab4b, lab4c, lab5a, lab5b, lab5c
```

This runs a build check to ensure the code compiles with test files reverted to their original state, then creates a `.tar.gz` archive to upload to Gradescope.

---

## Development Conventions

- **Stubs to fill in:** Source files contain `// Your code here` comments and `// Your definitions here` markers indicating where students should write code.
- **Testing:** Each lab has a `*_test.go` file with automated tests. Use `RUN="-run TestName"` to run specific tests.
- **Race safety:** The `-race` flag is enabled by default; ensure proper synchronization (mutexes, channels, etc.).
- **Do not modify test files:** The submission checker verifies that test-related files are unmodified from the originals.
- **Lab stubs are intentionally minimal:** Server constructors, RPC handlers, and core logic return placeholder values or `nil` — students implement the full logic.
- **Timeout:** Lab 5 (shardkv) uses a 15-minute test timeout; other labs use the default.

---

## Architecture Notes

### Raft (Lab 3)
- Defined in `src/raft1/raft.go` with the `Raft` struct.
- Uses `labrpc` for simulated RPC communication between peers.
- Persister (`tester1.Persister`) handles durable state storage.
- The `raftapi` package defines the interface Raft exposes to upper layers.

### Replicated State Machine (Lab 4)
- `src/kvraft1/rsm/` provides the RSM layer that bridges Raft to the KV server.
- `KVServer` in `src/kvraft1/server.go` submits operations via `kv.rsm.Submit()`.
- Supports `Get`, `Put`, and `Append` operations with linearizable semantics.

### MapReduce (Lab 1)
- Coordinator in `src/mr/coordinator.go` schedules map/reduce tasks.
- Workers in `src/mr/worker.go` execute tasks and register with the coordinator.
- Plugins in `src/mrapps/` are built as Go shared libraries (`-buildmode=plugin`).
