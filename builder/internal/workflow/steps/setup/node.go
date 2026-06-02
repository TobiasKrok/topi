package setup

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/tobiaskrok/topi/builder/internal/workflow"
	sharedworkflow "github.com/tobiaskrok/topi/shared/workflow"
)

type NodeStep struct {
	version string
}

func init() {
	workflow.RegisterStep("setup/node", newNodeStep)
}

func newNodeStep(cfg sharedworkflow.StepConfig) (workflow.Step, error) {
	version := cfg.Params["version"]
	if version == "" {
		version = "20"
	}
	return &NodeStep{
		version: version,
	}, nil
}

func (n *NodeStep) Exec(ctx *workflow.WorkflowContext) *workflow.StepResult {
	//TODO: shell escape,this is dangerous!!!
	installScript := fmt.Sprintf(`
set -e

curl -fsSL https://deb.nodesource.com/setup_%s.x | sudo -E bash -

echo "Installing Node.js %s..."
sudo apt-get install -y nodejs

`, n.version, n.version)

	cmd := exec.Command("bash", "-c", installScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return &workflow.StepResult{
			Status: workflow.StepFailed,
			Error:  fmt.Errorf("failed to install Node.js %s: %w", n.version, err),
		}
	}

	os.Setenv("PATH", "/usr/bin:"+os.Getenv("PATH"))

	return &workflow.StepResult{
		Status: workflow.StepSuccess,
	}
}

func (n *NodeStep) Name() string {
	return "Setup Node.js"
}

func (n *NodeStep) Description() string {
	return fmt.Sprintf("Install and set up Node.js %s with npm", n.version)
}
