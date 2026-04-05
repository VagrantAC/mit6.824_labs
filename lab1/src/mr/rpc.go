package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

// Add your RPC definitions here.
type RegisterRequest struct {
}

type RegisterResponse struct {
	WorkerID int
}

type MapTask struct {
	Id      string // map task filename
	NReduce int
}

type ReduceTask struct {
	Id        int // ihash value
	Filenames []string
}

type AssignTaskRequest struct {
	WorkerID int
}

type AssignTaskResponse struct {
	MapTask    *MapTask    // map task filename
	ReduceTask *ReduceTask // reduce task filename
}

type ReportMapTaskRequest struct {
	WorkerID      int
	MapTask       MapTask
	ReduceTaskMap map[int]string
}

type ReportMapTaskResponse struct {
}

type ReportReduceTaskRequest struct {
	WorkerID   int
	ReduceTask ReduceTask
	Filename   string
}

type ReportReduceTaskResponse struct {
}

type IsDoneRequest struct {
	WorkerID int
}

type IsDoneResponse struct {
	Done bool
}

// Add your RPC definitions here.
const (
	RegisterFunc         = "Coordinator.Register"
	AssignTaskFunc       = "Coordinator.AssignTask"
	ReportReduceTaskFunc = "Coordinator.ReportReduceTask"
	ReportMapTaskFunc    = "Coordinator.ReportMapTask"
	IsDoneFunc           = "Coordinator.IsDone"
)
