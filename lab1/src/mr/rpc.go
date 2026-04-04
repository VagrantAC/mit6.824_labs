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

type AssignTaskRequest struct {
	WorkerID int
}

type AssignTaskResponse struct {
	MapTasks    []string
	ReduceTasks []string
}

type ReportTaskRequest struct {
	WorkerID int
	TaskFile string
	Filename string
}

type ReportTaskResponse struct {
}

// Add your RPC definitions here.
const (
	RegisterFunc   = "Coordinator.Register"
	AssignTaskFunc = "Coordinator.AssignTask"
	ReportTaskFunc = "Coordinator.ReportTask"
)
