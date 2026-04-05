package mr

import (
	"sync"
	"time"

	"github.com/samber/lo"
)

type ReduceTaskManager struct {
	tasks        map[int]*ReduceTask
	statusMap    map[int]TaskStatus
	assignedTime map[int]time.Time
	mu           sync.RWMutex
}

func NewReduceTaskManager(nReduce int) *ReduceTaskManager {
	m := &ReduceTaskManager{
		tasks:        make(map[int]*ReduceTask),
		statusMap:    make(map[int]TaskStatus),
		assignedTime: make(map[int]time.Time),
	}

	for i := 0; i < nReduce; i++ {
		m.tasks[i] = &ReduceTask{Id: i, Filenames: []string{}}
		m.statusMap[i] = TaskPending
	}

	return m
}

func (r *ReduceTaskManager) GetPendingTask() *ReduceTask {
	r.mu.Lock()
	defer r.mu.Unlock()

	entries := lo.Entries(r.tasks)
	if entry, found := lo.Find(entries, func(entry lo.Entry[int, *ReduceTask]) bool {
		return r.statusMap[entry.Key] == TaskPending
	}); found {
		return entry.Value
	}

	if entry, found := lo.Find(entries, func(entry lo.Entry[int, *ReduceTask]) bool {
		return r.statusMap[entry.Key] == TaskAssigned && time.Since(r.assignedTime[entry.Key]) > TaskTimeout
	}); found {
		return entry.Value
	}

	return nil
}

func (r *ReduceTaskManager) AssignTask(taskID int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if status, ok := r.statusMap[taskID]; !ok || status == TaskCompleted {
		return false
	}

	r.statusMap[taskID] = TaskAssigned
	r.assignedTime[taskID] = time.Now()
	return true
}

func (r *ReduceTaskManager) CompleteTask(taskID int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.statusMap[taskID]; !ok {
		return false
	}

	r.statusMap[taskID] = TaskCompleted
	return true
}

func (r *ReduceTaskManager) IsTaskCompleted(taskID int) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.statusMap[taskID] == TaskCompleted
}

func (r *ReduceTaskManager) AllCompleted() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return lo.EveryBy(lo.Values(r.statusMap), func(status TaskStatus) bool {
		return status == TaskCompleted
	})
}

func (r *ReduceTaskManager) AddFile(taskID int, filename string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if task, ok := r.tasks[taskID]; ok {
		task.Filenames = append(task.Filenames, filename)
		return true
	}
	return false
}

func (r *ReduceTaskManager) GetTask(taskID int) *ReduceTask {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.tasks[taskID]
}

func (r *ReduceTaskManager) GetTaskCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tasks)
}

func (r *ReduceTaskManager) GetCompletedCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return lo.CountBy(lo.Values(r.statusMap), func(status TaskStatus) bool {
		return status == TaskCompleted
	})
}

func (r *ReduceTaskManager) GetPendingCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return lo.CountBy(lo.Values(r.statusMap), func(status TaskStatus) bool {
		return status == TaskPending
	})
}

func (r *ReduceTaskManager) GetAssignedCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return lo.CountBy(lo.Values(r.statusMap), func(status TaskStatus) bool {
		return status == TaskAssigned
	})
}

func (r *ReduceTaskManager) HasFiles(taskID int) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if task, ok := r.tasks[taskID]; ok {
		return len(task.Filenames) > 0
	}
	return false
}
