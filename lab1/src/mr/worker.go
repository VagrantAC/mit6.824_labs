package mr

import (
	"fmt"
	"hash/fnv"
	"io"
	"net/rpc"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
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

// utils

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
		log.Fatalf("cannot open %v", filename)
		return "", err
	}
	content, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", filename)
		return "", err
	}
	file.Close()
	return string(content), nil
}

var coordSockName string // socket for coordinator

// main/mrworker.go calls this function.
func Worker(sockname string, mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	coordSockName = sockname

	worker := JobWorker{}
	if ok := worker.Register(); !ok {
		return
	}

	for {

	}
}

func (worker *JobWorker) Register() bool {
	logger := log.WithFields(logrus.Fields{"Func": RegisterFunc})
	req := RegisterRequest{}
	resp := RegisterResponse{}
	ok := call(RegisterFunc, &req, &resp)
	if ok {
		logger.Infof("resp.WorkerID %v\n", resp.WorkerID)
		worker.workerID = resp.WorkerID
	} else {
		logger.Printf("call failed!\n")
	}
	return ok
}

func (worker *JobWorker) mapTask(taskFile string) ([]string, error) {
	content, err := readFileContent(taskFile)
	if err != nil {
		return nil, err
	}

	kva := worker.mapf(taskFile, content)
	taskMap := map[int][]KeyValue{}
	for _, kv := range kva {
		key := ihash(kv.Key)
		if _, ok := taskMap[key]; !ok {
			taskMap[key] = []KeyValue{}
		}
		taskMap[key] = append(taskMap[key], kv)
	}

	reduceIds := []string{}

	for key, kvs := range taskMap {
		filename := GenReduceTaskFilename(taskFile, worker.workerID, key)
		if err := WriteKeyValues(filename, kvs); err != nil {
			return nil, err
		}
		reduceIds = append(reduceIds, filename)
	}
	return reduceIds, nil
}

func (worker *JobWorker) reduceTask(taskFiles []string) (string, error) {
	taskMap := map[string][]string{}
	for _, file := range taskFiles {
		kvs, err := ReadKeyValues(file)
		if err != nil {
			return "", err
		}
		for _, kv := range kvs {
			key := kv.Key
			if _, ok := taskMap[key]; !ok {
				taskMap[key] = []string{}
			}
			taskMap[key] = append(taskMap[key], kv.Value)
		}
	}

	oname := "mr-out-" + strconv.Itoa(worker.workerID)
	ofile, err := os.Create(oname)
	if err != nil {
		return "", err
	}
	defer ofile.Close()

	for key, values := range taskMap {
		taskMap[key] = values
		output := worker.reducef(key, values)
		if _, err := fmt.Fprintf(ofile, "%v %v\n", key, output); err != nil {
			return "", err
		}
	}

	return oname, nil
}

func (worker *JobWorker) AssignTask() bool {
	logger := log.WithFields(logrus.Fields{
		"Func":     AssignTaskFunc,
		"WorkerID": worker.workerID,
	})
	req := AssignTaskRequest{}
	resp := AssignTaskResponse{}
	ok := call(AssignTaskFunc, &req, &resp)
	if ok {
		logger.Infof("resp.TaskFile %v\n", resp.TaskFile)
	} else {
		logger.Printf("call failed!\n")
	}
	return ok
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	c, err := rpc.DialHTTP("unix", coordSockName)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	if err := c.Call(rpcname, args, reply); err == nil {
		return true
	}
	log.Printf("%d: call failed err %v", os.Getpid(), err)
	return false
}
