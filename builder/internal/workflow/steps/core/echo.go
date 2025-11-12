package core

import (
	"fmt"

	"github.com/tobiaskrok/topi/builder/internal/workflow"
	sharedworkflow "github.com/tobiaskrok/topi/shared/workflow"
)

// TODO: print env variables
func init() {
	workflow.RegisterStep("echo", newEchoStep)
}

type EchoStep struct {
	name    string
	message string
}

func newEchoStep(cfg sharedworkflow.StepConfig) (workflow.Step, error) {
	message, ok := cfg.Params["message"]
	if !ok {
		message = ""
	}

	return &EchoStep{
		name:    cfg.Name,
		message: message,
	}, nil
}

func (s *EchoStep) Exec(ctx *workflow.WorkflowContext) *workflow.StepResult {
	fmt.Println(ctx.EnvironmentManager.ExpandString(s.message))
	return &workflow.StepResult{
		Status: workflow.StepSuccess,
	}
}

func (s *EchoStep) Name() string {
	return s.name
}

func (s *EchoStep) Description() string {
	return "Prints a message to stdout"
}
