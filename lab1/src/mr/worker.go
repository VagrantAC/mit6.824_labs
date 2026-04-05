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

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

const (
	RPCRetryCount   = 3
	RPCRetryDelay   = 100 * time.Millisecond
	WorkerSleepTime = 100 * time.Millisecond
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

type JobWorker struct {
	workerID int
	mapf     func(string, string) []KeyValue
	reducef  func(string, []string) string
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
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

var coordSockName string // socket for coordinator

// main/mrworker.go calls this function.
func Worker(sockname string, mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	coordSockName = sockname

	worker := JobWorker{
		mapf:    mapf,
		reducef: reducef,
	}
	if ok := worker.Register(); !ok {
		return
	}

	for worker.IsWaiting() {
		worker.AssignTask()
		time.Sleep(time.Millisecond * 100)
	}
}

func (worker *JobWorker) Register() bool {
	logger := log.WithFields(logrus.Fields{"Func": RegisterFunc})
	req := RegisterRequest{}
	resp := RegisterResponse{}

	for i := range RPCRetryCount {
		if call(RegisterFunc, &req, &resp) {
			logger.Infof("resp.WorkerID %v\n", resp.WorkerID)
			worker.workerID = resp.WorkerID
			return true
		}
		if i < RPCRetryCount-1 {
			time.Sleep(RPCRetryDelay)
		}
	}
	logger.Printf("call failed after %d retries!\n", RPCRetryCount)
	return false
}

func (worker *JobWorker) mapTask(task MapTask) error {
	content, err := readFileContent(task.Id)
	if err != nil {
		return err
	}

	kva := worker.mapf(task.Id, content)
	taskMap := map[int][]KeyValue{}
	for _, kv := range kva {
		key := ihash(kv.Key) % task.NReduce
		if _, ok := taskMap[key]; !ok {
			taskMap[key] = []KeyValue{}
		}
		taskMap[key] = append(taskMap[key], kv)
	}

	reduceTaskMap := map[int]string{}

	for key, kvs := range taskMap {
		filename, err := WriteKeyValues(kvs)
		if err != nil {
			return err
		}
		reduceTaskMap[key] = filename
	}

	for i := range RPCRetryCount {
		if call(ReportMapTaskFunc, &ReportMapTaskRequest{
			WorkerID:      worker.workerID,
			MapTask:       task,
			ReduceTaskMap: reduceTaskMap,
		}, &ReportMapTaskResponse{}) {
			return nil
		}
		if i < RPCRetryCount-1 {
			time.Sleep(RPCRetryDelay)
		}
	}
	return fmt.Errorf("failed to report map task after %d retries", RPCRetryCount)
}

func (worker *JobWorker) reduceTask(task ReduceTask) error {
	taskMap := map[string][]string{}
	for _, file := range task.Filenames {
		kvs, err := ReadKeyValues(file)
		if err != nil {
			return err
		}
		for _, kv := range kvs {
			key := kv.Key
			if _, ok := taskMap[key]; !ok {
				taskMap[key] = []string{}
			}
			taskMap[key] = append(taskMap[key], kv.Value)
		}
	}

	var content strings.Builder
	for key, values := range taskMap {
		output := worker.reducef(key, values)
		fmt.Fprintf(&content, "%v %v\n", key, output)
	}

	filename := "mr-out-" + strconv.Itoa(task.Id)

	ofile, err := os.Create(filename)
	if err != nil {
		return err
	}
	if _, err := ofile.WriteString(content.String()); err != nil {
		ofile.Close()
		return err
	}
	ofile.Close()

	for i := range RPCRetryCount {
		if call(ReportReduceTaskFunc, &ReportReduceTaskRequest{
			WorkerID:   worker.workerID,
			ReduceTask: task,
			Filename:   filename,
		}, &ReportReduceTaskResponse{}) {
			return nil
		}
		if i < RPCRetryCount-1 {
			time.Sleep(RPCRetryDelay)
		}
	}
	return fmt.Errorf("failed to report reduce task after %d retries", RPCRetryCount)
}

func (worker *JobWorker) AssignTask() bool {
	logger := log.WithFields(logrus.Fields{
		"Func":     AssignTaskFunc,
		"WorkerID": worker.workerID,
	})
	req := AssignTaskRequest{}
	resp := AssignTaskResponse{}

	for i := range RPCRetryCount {
		if call(AssignTaskFunc, &req, &resp) {
			if resp.MapTask != nil {
				logger.Infof("resp.MapTask %v\n", resp.MapTask)
				if err := worker.mapTask(*resp.MapTask); err != nil {
					logger.Errorf("map task failed: %v", err)
					return false
				}
			}

			if resp.ReduceTask != nil {
				logger.Infof("resp.ReduceTask %v\n", resp.ReduceTask)
				if err := worker.reduceTask(*resp.ReduceTask); err != nil {
					logger.Errorf("reduce task failed: %v", err)
					return false
				}
			}
			return true
		}
		if i < RPCRetryCount-1 {
			time.Sleep(RPCRetryDelay)
		}
	}
	logger.Printf("call failed after %d retries!\n", RPCRetryCount)
	return false
}

func (worker *JobWorker) IsWaiting() bool {
	logger := log.WithFields(logrus.Fields{
		"Func":     IsDoneFunc,
		"WorkerID": worker.workerID,
	})
	req := IsDoneRequest{WorkerID: worker.workerID}
	resp := IsDoneResponse{}

	for i := range RPCRetryCount {
		if call(IsDoneFunc, &req, &resp) {
			logger.Infof("resp %v\n", resp)
			return !resp.Done
		}
		if i < RPCRetryCount-1 {
			time.Sleep(RPCRetryDelay)
		}
	}
	logger.Printf("call failed after %d retries!\n", RPCRetryCount)
	return false
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	c, err := rpc.DialHTTP("unix", coordSockName)
	if err != nil {
		log.Printf("dialing error: %v", err)
		return false
	}
	defer c.Close()

	if err := c.Call(rpcname, args, reply); err == nil {
		return true
	}
	log.Printf("%d: call failed err %v", os.Getpid(), err)
	return false
}
