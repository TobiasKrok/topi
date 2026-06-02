package core

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/tobiaskrok/topi/builder/internal/workflow"
	sharedworkflow "github.com/tobiaskrok/topi/shared/workflow"
)

// TODO: print env variables
func init() {
	workflow.RegisterStep("shell", newShellStep)
}

type ShellStep struct {
	cmd  string
	name string
}

func newShellStep(cfg sharedworkflow.StepConfig) (workflow.Step, error) {
	var cmd string
	if cfg.Run != "" {
		cmd = cfg.Run
	} else if cfg.Uses != "" {
		cmd = cfg.Params["cmd"]
	} else {
		return nil, fmt.Errorf("step '%s' is missing 'run' property ", cfg.Name)
	}

	return &ShellStep{
		name: cfg.Name,
		cmd:  cmd,
	}, nil
}

func (s *ShellStep) Exec(ctx *workflow.WorkflowContext) *workflow.StepResult {
	// shell escpae
	expanded := ctx.EnvironmentManager.ExpandString(s.cmd)

	cmd := exec.Command("bash", "-c", expanded) // Example command
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Print(err)
		return &workflow.StepResult{
			Status: workflow.StepFailed,
			Error:  err,
		}
	}

	return &workflow.StepResult{
		Status: workflow.StepSuccess,
	}
}

func (s *ShellStep) Name() string {
	return s.name
}

func (s *ShellStep) Description() string {
	return "Prints a message to stdout"
}
