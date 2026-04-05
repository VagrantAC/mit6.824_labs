package mr

import (
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	TaskTimeout = 10 * time.Second
)

type TaskStatus int

const (
	TaskPending TaskStatus = iota
	TaskAssigned
	TaskCompleted
)

type Coordinator struct {
	nReduce    int
	workerID   int
	mapManager *MapTaskManager
	reduceMgr  *ReduceTaskManager
	mu         sync.Mutex
	kva        []KeyValue
}

func (c *Coordinator) Register(req *RegisterRequest, resp *RegisterResponse) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	resp.WorkerID = c.workerID + 1
	c.workerID++

	WithFields(logrus.Fields{
		"service":   "mr-coordinator",
		"component": "rpc",
		"worker_id": resp.WorkerID,
	}).Info("Worker registered successfully")
	return nil
}

func (c *Coordinator) AssignTask(req *AssignTaskRequest, resp *AssignTaskResponse) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	logger := WithFields(logrus.Fields{
		"service":   "mr-coordinator",
		"component": "task-scheduler",
		"worker_id": req.WorkerID,
	})

	if task := c.mapManager.GetPendingTask(); task != nil {
		logger.WithField("task_id", task.Id).Info("Assigning MapTask to worker")
		resp.MapTask = task
		c.mapManager.AssignTask(task.Id)
		return nil
	}

	if !c.mapManager.AllCompleted() {
		logger.Debug("Map tasks not completed yet, waiting")
		return nil
	}

	logger.Info("All map tasks completed, starting reduce tasks")

	if task := c.reduceMgr.GetPendingTask(); task != nil {
		logger.WithField("task_id", task.Id).Info("Assigning ReduceTask to worker")
		resp.ReduceTask = task
		c.reduceMgr.AssignTask(task.Id)
		return nil
	}

	if c.reduceMgr.AllCompleted() {
		logger.Info("All reduce tasks completed, no more tasks available")
		resp.ReduceTask = nil
	}

	return nil
}

func (c *Coordinator) ReportMapTask(req *ReportMapTaskRequest, resp *ReportMapTaskResponse) error {
	logger := WithFields(logrus.Fields{
		"service":   "mr-coordinator",
		"component": "task-tracker",
		"worker_id": req.WorkerID,
		"task_id":   req.MapTask.Id,
	})

	logger.WithField("file_count", len(req.ReduceTaskMap)).Info("Map task completed")

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.mapManager.IsTaskCompleted(req.MapTask.Id) {
		logger.Warn("Map task already completed, ignoring duplicate report")
		return nil
	}

	c.mapManager.CompleteTask(req.MapTask.Id)
	logger.Info("Map task marked as completed")

	for id, filename := range req.ReduceTaskMap {
		c.reduceMgr.AddFile(id, filename)
	}

	return nil
}

func (c *Coordinator) ReportReduceTask(req *ReportReduceTaskRequest, resp *ReportReduceTaskResponse) error {
	logger := WithFields(logrus.Fields{
		"service":   "mr-coordinator",
		"component": "task-tracker",
		"worker_id": req.WorkerID,
		"task_id":   req.ReduceTask.Id,
	})

	logger.WithField("output_file", req.Filename).Info("Reduce task completed")

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.reduceMgr.IsTaskCompleted(req.ReduceTask.Id) {
		logger.Warn("Reduce task already completed, ignoring duplicate report")
		return nil
	}

	c.reduceMgr.CompleteTask(req.ReduceTask.Id)
	logger.Info("Reduce task marked as completed")

	return nil
}

func (c *Coordinator) IsDone(req *IsDoneRequest, resp *IsDoneResponse) error {
	resp.Done = c.Done()
	return nil
}

func (c *Coordinator) server(sockname string) {
	rpc.Register(c)
	rpc.HandleHTTP()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		Fatalf("listen error %s: %v", sockname, e)
	}
	go http.Serve(l, nil)
}

func (c *Coordinator) Done() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.mapManager.AllCompleted() {
		return false
	}

	if !c.reduceMgr.AllCompleted() {
		return false
	}

	Info("All tasks completed successfully")
	return true
}

func MakeCoordinator(sockname string, files []string, nReduce int) *Coordinator {
	c := &Coordinator{
		workerID:   0,
		mapManager: NewMapTaskManager(files, nReduce),
		reduceMgr:  NewReduceTaskManager(nReduce),
		mu:         sync.Mutex{},
		kva:        make([]KeyValue, 0),
		nReduce:    nReduce,
	}

	Infof("Coordinator created with %d map tasks and %d reduce tasks",
		c.mapManager.GetTaskCount(), c.reduceMgr.GetTaskCount())

	c.server(sockname)
	return c
}
