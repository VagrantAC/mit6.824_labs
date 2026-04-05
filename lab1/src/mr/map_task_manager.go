package mr

import (
	"sync"
	"time"

	"github.com/samber/lo"
)

type MapTaskManager struct {
	tasks        map[string]*MapTask
	statusMap    map[string]TaskStatus
	assignedTime map[string]time.Time
	nReduce      int
	mu           sync.RWMutex
}

func NewMapTaskManager(files []string, nReduce int) *MapTaskManager {
	m := &MapTaskManager{
		tasks:        make(map[string]*MapTask),
		statusMap:    make(map[string]TaskStatus),
		assignedTime: make(map[string]time.Time),
		nReduce:      nReduce,
	}

	for _, file := range files {
		m.tasks[file] = &MapTask{Id: file, NReduce: nReduce}
		m.statusMap[file] = TaskPending
	}

	return m
}

func (m *MapTaskManager) GetPendingTask() *MapTask {
	m.mu.Lock()
	defer m.mu.Unlock()

	entries := lo.Entries(m.tasks)
	if entry, found := lo.Find(entries, func(entry lo.Entry[string, *MapTask]) bool {
		return m.statusMap[entry.Key] == TaskPending
	}); found {
		return entry.Value
	}

	if entry, found := lo.Find(entries, func(entry lo.Entry[string, *MapTask]) bool {
		return m.statusMap[entry.Key] == TaskAssigned && time.Since(m.assignedTime[entry.Key]) > TaskTimeout
	}); found {
		return entry.Value
	}

	return nil
}

func (m *MapTaskManager) AssignTask(taskID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if status, ok := m.statusMap[taskID]; !ok || status == TaskCompleted {
		return false
	}

	m.statusMap[taskID] = TaskAssigned
	m.assignedTime[taskID] = time.Now()
	return true
}

func (m *MapTaskManager) CompleteTask(taskID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.statusMap[taskID]; !ok {
		return false
	}

	m.statusMap[taskID] = TaskCompleted
	return true
}

func (m *MapTaskManager) IsTaskCompleted(taskID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.statusMap[taskID] == TaskCompleted
}

func (m *MapTaskManager) AllCompleted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return lo.EveryBy(lo.Values(m.statusMap), func(status TaskStatus) bool {
		return status == TaskCompleted
	})
}

func (m *MapTaskManager) GetTaskCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.tasks)
}

func (m *MapTaskManager) GetCompletedCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return lo.CountBy(lo.Values(m.statusMap), func(status TaskStatus) bool {
		return status == TaskCompleted
	})
}

func (m *MapTaskManager) GetPendingCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return lo.CountBy(lo.Values(m.statusMap), func(status TaskStatus) bool {
		return status == TaskPending
	})
}

func (m *MapTaskManager) GetAssignedCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return lo.CountBy(lo.Values(m.statusMap), func(status TaskStatus) bool {
		return status == TaskAssigned
	})
}
