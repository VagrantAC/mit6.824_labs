package mr

import (
	"fmt"
	"hash/fnv"
	"io"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
)

const (
	RPCRetryCount   = 3
	RPCRetryDelay   = 100 * time.Millisecond
	WorkerSleepTime = 100 * time.Millisecond
)

type KeyValue struct {
	Key   string
	Value string
}

type JobWorker struct {
	workerID int
	mapf     func(string, string) []KeyValue
	reducef  func(string, []string) string
}

func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

func readFileContent(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("cannot open %v: %w", filename, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("cannot read %v: %w", filename, err)
	}
	return string(content), nil
}

var coordSockName string

func Worker(sockname string, mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	coordSockName = sockname

	worker := JobWorker{
		mapf:    mapf,
		reducef: reducef,
	}

	if !worker.Register() {
		Error("Failed to register worker")
		return
	}

	WithFields(logrus.Fields{
		"service":   "mr-worker",
		"component": "main",
		"worker_id": worker.workerID,
	}).Info("Worker started successfully")

	for worker.IsWaiting() {
		worker.AssignTask()
		time.Sleep(WorkerSleepTime)
	}

	WithFields(logrus.Fields{
		"service":   "mr-worker",
		"component": "main",
		"worker_id": worker.workerID,
	}).Info("Worker finished all tasks")
}

func (worker *JobWorker) Register() bool {
	logger := WithFields(logrus.Fields{
		"service":   "mr-worker",
		"component": "rpc",
	})
	req := RegisterRequest{}
	resp := RegisterResponse{}

	for i := range RPCRetryCount {
		if call(RegisterFunc, &req, &resp) {
			logger.WithField("worker_id", resp.WorkerID).Info("Worker registered successfully")
			worker.workerID = resp.WorkerID
			return true
		}
		if i < RPCRetryCount-1 {
			time.Sleep(RPCRetryDelay)
		}
	}
	logger.Error("Failed to register after retries")
	return false
}

func (worker *JobWorker) mapTask(task MapTask) error {
	logger := WithFields(logrus.Fields{
		"service":   "mr-worker",
		"component": "map-task",
		"worker_id": worker.workerID,
		"task_id":   task.Id,
	})

	logger.Info("Starting map task")

	content, err := readFileContent(task.Id)
	if err != nil {
		logger.WithField("error", err.Error()).Error("Failed to read file")
		return err
	}

	kva := worker.mapf(task.Id, content)
	logger.WithField("kv_count", len(kva)).Info("Map function executed")

	taskMap := lo.GroupBy(kva, func(kv KeyValue) int {
		return ihash(kv.Key) % task.NReduce
	})

	reduceTaskMap := make(map[int]string)
	for key, kvs := range taskMap {
		filename, err := WriteKeyValues(kvs)
		if err != nil {
			logger.WithField("error", err.Error()).Error("Failed to write key-values")
			return err
		}
		reduceTaskMap[key] = filename
	}

	logger.WithField("file_count", len(reduceTaskMap)).Info("Generated intermediate files")

	for i := range RPCRetryCount {
		if call(ReportMapTaskFunc, &ReportMapTaskRequest{
			WorkerID:      worker.workerID,
			MapTask:       task,
			ReduceTaskMap: reduceTaskMap,
		}, &ReportMapTaskResponse{}) {
			logger.Info("Map task reported successfully")
			return nil
		}
		if i < RPCRetryCount-1 {
			time.Sleep(RPCRetryDelay)
		}
	}

	logger.Error("Failed to report map task after retries")
	return fmt.Errorf("failed to report map task after %d retries", RPCRetryCount)
}

func (worker *JobWorker) reduceTask(task ReduceTask) error {
	logger := WithFields(logrus.Fields{
		"service":   "mr-worker",
		"component": "reduce-task",
		"worker_id": worker.workerID,
		"task_id":   task.Id,
	})

	logger.WithField("file_count", len(task.Filenames)).Info("Starting reduce task")

	allKVs := lo.FlatMap(task.Filenames, func(item string, index int) []KeyValue {
		kvs, err := ReadKeyValues(item)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error": err.Error(),
				"file":  item,
			}).Error("Failed to read key-values")
			return []KeyValue{}
		}
		return kvs
	})

	taskMap := lo.GroupBy(allKVs, func(kv KeyValue) string {
		return kv.Key
	})

	logger.WithField("key_count", len(taskMap)).Info("Reduce task processing")

	var content strings.Builder
	for key, values := range taskMap {
		output := worker.reducef(key, lo.Map(values, func(kv KeyValue, _ int) string {
			return kv.Value
		}))
		fmt.Fprintf(&content, "%v %v\n", key, output)
	}

	filename := "mr-out-" + strconv.Itoa(task.Id)

	ofile, err := os.Create(filename)
	if err != nil {
		logger.WithField("error", err.Error()).Error("Failed to create output file")
		return err
	}
	if _, err := ofile.WriteString(content.String()); err != nil {
		ofile.Close()
		logger.WithField("error", err.Error()).Error("Failed to write output file")
		return err
	}
	ofile.Close()

	logger.WithField("output_file", filename).Info("Reduce task completed")

	for i := range RPCRetryCount {
		if call(ReportReduceTaskFunc, &ReportReduceTaskRequest{
			WorkerID:   worker.workerID,
			ReduceTask: task,
			Filename:   filename,
		}, &ReportReduceTaskResponse{}) {
			logger.Info("Reduce task reported successfully")
			return nil
		}
		if i < RPCRetryCount-1 {
			time.Sleep(RPCRetryDelay)
		}
	}

	logger.Error("Failed to report reduce task after retries")
	return fmt.Errorf("failed to report reduce task after %d retries", RPCRetryCount)
}

func (worker *JobWorker) AssignTask() bool {
	logger := WithFields(logrus.Fields{
		"service":   "mr-worker",
		"component": "task-client",
		"worker_id": worker.workerID,
	})
	req := AssignTaskRequest{}
	resp := AssignTaskResponse{}

	for i := range RPCRetryCount {
		if call(AssignTaskFunc, &req, &resp) {
			if resp.MapTask != nil {
				logger.WithField("task_id", resp.MapTask.Id).Info("Received MapTask")
				if err := worker.mapTask(*resp.MapTask); err != nil {
					logger.WithField("error", err.Error()).Error("Map task failed")
					return false
				}
			}

			if resp.ReduceTask != nil {
				logger.WithField("task_id", resp.ReduceTask.Id).Info("Received ReduceTask")
				if err := worker.reduceTask(*resp.ReduceTask); err != nil {
					logger.WithField("error", err.Error()).Error("Reduce task failed")
					return false
				}
			}
			return true
		}
		if i < RPCRetryCount-1 {
			time.Sleep(RPCRetryDelay)
		}
	}
	logger.Warn("No task available after retries")
	return false
}

func (worker *JobWorker) IsWaiting() bool {
	logger := WithFields(logrus.Fields{
		"service":   "mr-worker",
		"component": "task-client",
		"worker_id": worker.workerID,
	})
	req := IsDoneRequest{WorkerID: worker.workerID}
	resp := IsDoneResponse{}

	for i := range RPCRetryCount {
		if call(IsDoneFunc, &req, &resp) {
			if resp.Done {
				logger.Info("All tasks completed, worker can exit")
			}
			return !resp.Done
		}
		if i < RPCRetryCount-1 {
			time.Sleep(RPCRetryDelay)
		}
	}
	logger.Error("Failed to check job status after retries")
	return false
}

func call(rpcname string, args any, reply any) bool {
	c, err := rpc.DialHTTP("unix", coordSockName)
	if err != nil {
		Errorf("RPC dial error: %v", err)
		return false
	}
	defer c.Close()

	if err := c.Call(rpcname, args, reply); err == nil {
		return true
	}
	Errorf("RPC call failed for %s: %v", rpcname, err)
	return false
}
