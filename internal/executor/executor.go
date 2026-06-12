package executor

import (
	"context"
	"taskbridge/internal/model"
)

// Result is returned after executing a job.
type Result struct {
	Status model.JobStatus `json:"status"`
	Logs   []string        `json:"logs"`
	Result map[string]any  `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// Executor executes a single job type.
type Executor interface {
	Type() model.JobType
	Execute(ctx context.Context, job model.Job) Result
}

// Registry maps job types to executors.
type Registry struct {
	executors map[model.JobType]Executor
}

func NewRegistry() *Registry {
	return &Registry{executors: map[model.JobType]Executor{}}
}

func (r *Registry) Register(ex Executor) {
	r.executors[ex.Type()] = ex
}

func (r *Registry) Get(t model.JobType) (Executor, bool) {
	ex, ok := r.executors[t]
	return ex, ok
}

// TODO: Candidate should implement safe executors:
//   http_check
//   tcp_check
//   file_exists
//   checksum
//   write_file
//   wait
