package workflow

import "context"

type StepStatus int

type JobStatus int

const (
	StepFailed StepStatus = iota
	StepSuccess
	StepSkipped
)

const (
	JobFailed StepStatus = iota
	JobPartial
	JobSkipped
	JobSuccess
)

var stepStatusName = map[StepStatus]string{
	StepSuccess: "success",
	StepFailed:  "failed",
	StepSkipped: "skipped",
}

func (ss StepStatus) String() string {
	return stepStatusName[ss]
}

// Workflow -> Job -> Steps

type Workflow struct {
	Name               string
	Ctx                *WorkflowContext
	Jobs               []Job
	Results            map[string]map[string]*StepResult // jpb name -> step -> result
	EnvironmentManager *EnvironmentManager
}

// TODO secrets
type WorkflowContext struct {
	Params             map[string]string
	Workspace          string
	BuildID            string
	Source             string // git repo
	SystemDir          string
	EnvironmentManager *EnvironmentManager
	Context            context.Context
}

type Job struct {
	Name        string
	Needs       []string // job dependencies
	Steps       []Step
	ErrorPolicy string
}

type Step interface {
	Name() string
	Description() string
	Exec(ctx *WorkflowContext) (*StepResult, error)
}

type StepResult struct {
	// StdOut string
	// Outputs map[string]string
	Status StepStatus
}
