package mr

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
)

type TaskStatus int

const (
	TaskPending TaskStatus = iota
	TaskAssigned
	TaskCompleted
)

type Coordinator struct {
	workerID    int
	taskFileMap map[string]TaskStatus
	mu          sync.Mutex
	kva         []KeyValue
}

func (c *Coordinator) getUnfinishedTask() *string {
	for file, status := range c.taskFileMap {
		if status == TaskPending {
			return &file
		}
	}

	for file, status := range c.taskFileMap {
		if status == TaskAssigned {
			return &file
		}
	}
	return nil
}

func (c *Coordinator) Register(req *RegisterRequest, resp *RegisterResponse) error {
	resp.WorkerID = c.workerID + 1
	c.workerID++

	log.WithField("WorkerID", resp.WorkerID).Infof("Register: %v", resp.WorkerID)
	return nil
}

func (c *Coordinator) AssignTask(req *AssignTaskRequest, resp *AssignTaskResponse) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if taskFile := c.getUnfinishedTask(); taskFile != nil {
		log.WithField("WorkerID", req.WorkerID).Infof("AssignTask: %v to %v", *taskFile, req.WorkerID)
		resp.TaskFile = *taskFile
		c.taskFileMap[*taskFile] = TaskAssigned
	} else {
		log.WithField("WorkerID", req.WorkerID).Infof("Finish all tasks, no more task")
		resp.TaskFile = ""
	}
	return nil
}

func (c *Coordinator) ReportTask(req *ReportTaskRequest, resp *ReportTaskResponse) error {
	logger := log.WithField("WorkerID", req.WorkerID)
	logger.Infof("ReportTask: %v", req)
	if req.TaskFile == "" || req.Filename == "" {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.taskFileMap[req.TaskFile] == TaskCompleted {
		logger.Infof("Task %v is already completed", req.TaskFile)
		return nil
	}

	logger.Infof("Finish task: %v", req.TaskFile)
	file, err := os.Open(req.Filename)
	if err != nil {
		logger.Errorf("Open file %v error: %v", req.Filename, err)
		return err
	}
	defer file.Close()

	dec := json.NewDecoder(file)

	var kv KeyValue
	for {
		if err := dec.Decode(&kv); err == io.EOF {
			break
		} else if err != nil {
			logger.Errorf("Decode file %v error: %v", req.Filename, err)
			return err
		}
		c.kva = append(c.kva, kv)
	}

	c.taskFileMap[req.TaskFile] = TaskCompleted
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

	for _, status := range c.taskFileMap {
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
		workerID:    0,
		taskFileMap: make(map[string]TaskStatus),
		mu:          sync.Mutex{},
		kva:         make([]KeyValue, 0),
	}

	for _, file := range files {
		c.taskFileMap[file] = TaskPending
	}

	c.server(sockname)
	return &c
}
