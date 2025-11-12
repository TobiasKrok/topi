package setup

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/tobiaskrok/topi/builder/internal/workflow"
	sharedworkflow "github.com/tobiaskrok/topi/shared/workflow"
)

type GolangStep struct {
	version string
}

func init() {
	workflow.RegisterStep("setup/go", newGolangStep)
}

func newGolangStep(cfg sharedworkflow.StepConfig) (workflow.Step, error) {
	version := cfg.Params["version"]
	if version == "" {
		version = "1.25.1"
	}
	return &GolangStep{
		version: version,
	}, nil
}

func (g *GolangStep) Exec(ctx *workflow.WorkflowContext) *workflow.StepResult {

	// Determine architecture
	arch := runtime.GOARCH
	if arch == "" {
		arch = "amd64"
	}

	installScript := fmt.Sprintf(`
set -e

echo "Downloading Go %s..."
wget -q -O /tmp/go%s.tar.gz https://go.dev/dl/go%s.linux-%s.tar.gz

echo "Installing Go %s..."
sudo tar -C /usr/local -xzf /tmp/go%s.tar.gz

rm /tmp/go%s.tar.gz

`, g.version, g.version, g.version, arch, g.version, g.version, g.version)

	cmd := exec.Command("bash", "-c", installScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "PATH=/usr/local/go/bin:"+os.Getenv("PATH"))

	if err := cmd.Run(); err != nil {
		return &workflow.StepResult{
			Status: workflow.StepFailed,
			Error:  fmt.Errorf("failed to install Go %s: %w", g.version, err),
		}
	}

	os.Setenv("PATH", "/usr/local/go/bin:"+os.Getenv("PATH"))
	os.Setenv("GOROOT", "/usr/local/go")

	return &workflow.StepResult{
		Status: workflow.StepSuccess,
	}
}

func (g *GolangStep) Name() string {
	return "Setup Golang"
}

func (g *GolangStep) Description() string {
	return fmt.Sprintf("Install and set up Golang %s", g.version)
}
