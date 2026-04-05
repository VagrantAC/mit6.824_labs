package mr

import (
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
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
	nReduce             int
	workerID            int
	mapTaskIdMap        map[string]TaskStatus
	lastAssignedTimeMap map[string]time.Time
	mu                  sync.Mutex
	kva                 []KeyValue

	reduceTaskIdMap       map[int]TaskStatus
	reduceTaskMap         map[int]ReduceTask
	reduceLastAssignedMap map[int]time.Time
}

func (c *Coordinator) getUnfinishedMapTask() *MapTask {
	for id, status := range c.mapTaskIdMap {
		if status == TaskPending {
			return &MapTask{Id: id, NReduce: c.nReduce}
		}
	}

	for id, status := range c.mapTaskIdMap {
		if status == TaskAssigned && time.Since(c.lastAssignedTimeMap[id]) > TaskTimeout {
			return &MapTask{Id: id, NReduce: c.nReduce}
		}
	}
	return nil
}

func (c *Coordinator) getUnfinishedReduceTask() *ReduceTask {
	for id, status := range c.reduceTaskIdMap {
		if status == TaskPending {
			return &ReduceTask{Id: id, Filenames: c.reduceTaskMap[id].Filenames}
		}
	}

	for id, status := range c.reduceTaskIdMap {
		if status == TaskAssigned && time.Since(c.reduceLastAssignedMap[id]) > TaskTimeout {
			return &ReduceTask{Id: id, Filenames: c.reduceTaskMap[id].Filenames}
		}
	}
	return nil
}

func (c *Coordinator) Register(req *RegisterRequest, resp *RegisterResponse) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	resp.WorkerID = c.workerID + 1
	c.workerID++

	log.WithField("WorkerID", resp.WorkerID).Infof("Register: %v", resp.WorkerID)
	return nil
}

func (c *Coordinator) AssignTask(req *AssignTaskRequest, resp *AssignTaskResponse) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if task := c.getUnfinishedMapTask(); task != nil {
		log.WithField("WorkerID", req.WorkerID).Infof("AssignMapTask: [%v] to [%v]", task.Id, req.WorkerID)
		resp.MapTask = task
		c.mapTaskIdMap[task.Id] = TaskAssigned
		c.lastAssignedTimeMap[task.Id] = time.Now()
		return nil
	}

	if c.allMapTasksCompleted() {
		log.WithField("WorkerID", req.WorkerID).Infof("Finish all map tasks, starting reduce tasks")
	} else {
		return nil
	}

	if task := c.getUnfinishedReduceTask(); task != nil {
		log.WithField("WorkerID", req.WorkerID).Infof("AssignReduceTask: [%v] to [%v], task: %v", task.Id, req.WorkerID, task)
		resp.ReduceTask = task
		c.reduceTaskIdMap[task.Id] = TaskAssigned
		c.reduceLastAssignedMap[task.Id] = time.Now()
		return nil
	}

	if c.allReduceTasksCompleted() {
		log.WithField("WorkerID", req.WorkerID).Infof("Finish all reduce tasks, no more task")
		resp.ReduceTask = nil
	}
	return nil
}

func (c *Coordinator) allMapTasksCompleted() bool {
	for _, status := range c.mapTaskIdMap {
		if status != TaskCompleted {
			return false
		}
	}
	return true
}

func (c *Coordinator) allReduceTasksCompleted() bool {
	for _, status := range c.reduceTaskIdMap {
		if status != TaskCompleted {
			return false
		}
	}
	return true
}

func (c *Coordinator) ReportMapTask(req *ReportMapTaskRequest, resp *ReportMapTaskResponse) error {
	logger := log.WithField("WorkerID", req.WorkerID)
	logger.Infof("ReportMapTask: %v, %v", req.MapTask, req.ReduceTaskMap)

	task := req.MapTask

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.mapTaskIdMap[task.Id] == TaskCompleted {
		logger.Infof("MapTask %v is already completed", task.Id)
		return nil
	}

	c.mapTaskIdMap[task.Id] = TaskCompleted
	logger.Infof("Finish map task: %v", task.Id)

	for id, filename := range req.ReduceTaskMap {
		if _, ok := c.reduceTaskMap[id]; !ok {
			c.reduceTaskMap[id] = ReduceTask{Id: id, Filenames: []string{}}
			c.reduceTaskIdMap[id] = TaskPending
		}
		task := c.reduceTaskMap[id]
		task.Filenames = append(task.Filenames, filename)
		c.reduceTaskMap[id] = task
	}

	return nil
}

func (c *Coordinator) ReportReduceTask(req *ReportReduceTaskRequest, resp *ReportReduceTaskResponse) error {
	logger := log.WithField("WorkerID", req.WorkerID)

	task := req.ReduceTask

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.reduceTaskIdMap[task.Id] == TaskCompleted {
		logger.Infof("ReduceTask %v is already completed", task.Id)
		return nil
	}
	c.reduceTaskIdMap[task.Id] = TaskCompleted
	logger.Infof("Finish reduce task: %v", task.Id)

	return nil
}

func (c *Coordinator) IsDone(req *IsDoneRequest, resp *IsDoneResponse) error {
	resp.Done = c.Done()
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server(sockname string) {
	rpc.Register(c)
	rpc.HandleHTTP()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatalf("listen error %s: %v", sockname, e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, status := range c.mapTaskIdMap {
		if status != TaskCompleted {
			return false
		}
	}

	for _, status := range c.reduceTaskIdMap {
		if status != TaskCompleted {
			return false
		}
	}
	return true
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(sockname string, files []string, nReduce int) *Coordinator {
	c := Coordinator{
		workerID:              0,
		mapTaskIdMap:          make(map[string]TaskStatus),
		lastAssignedTimeMap:   make(map[string]time.Time),
		reduceTaskIdMap:       make(map[int]TaskStatus),
		reduceTaskMap:         make(map[int]ReduceTask),
		reduceLastAssignedMap: make(map[int]time.Time),
		mu:                    sync.Mutex{},
		kva:                   make([]KeyValue, 0),
		nReduce:               nReduce,
	}

	for _, file := range files {
		c.mapTaskIdMap[file] = TaskPending
	}

	c.server(sockname)
	return &c
}
