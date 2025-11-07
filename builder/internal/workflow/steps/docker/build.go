package docker

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/tobiaskrok/topi/builder/internal/workflow"
	sharedworkflow "github.com/tobiaskrok/topi/shared/workflow"
)

// TODO: print env variables
func init() {
	workflow.RegisterStep("docker/build", newDockerBuildStep)
}

type DockerBuildStep struct {
	name     string
	registry string
	image    string
	tag      string
}

func newDockerBuildStep(cfg sharedworkflow.StepConfig) (workflow.Step, error) {

	image, ok := cfg.Params["image"]
	if !ok {
		return nil, fmt.Errorf("[docker-build] propery 'image' cannot be empty")
	}
	tag, ok := cfg.Params["tag"]
	if !ok {
		tag = "latest"
	}
	registry, ok := cfg.Params["registry"]
	if !ok {
		return nil, fmt.Errorf("[docker-build] property 'registry' cannot be empty")
	}
	return &DockerBuildStep{
		name:     cfg.Name,
		image:    image,
		tag:      tag,
		registry: registry,
	}, nil
}

func (s *DockerBuildStep) Exec(ctx *workflow.WorkflowContext) (*workflow.StepResult, error) {

	//TODO: TLS, insecure false!!
	buildx := fmt.Sprintf("buildctl --addr tcp://buildkit.topi-system.svc.cluster.local:1234 build --frontend=dockerfile.v0 --local context=. --local dockerfile=. --output type=image,name=%s/%s:%s,push=true,insecure=true", s.registry, s.image, s.tag)
	cmd := exec.Command("bash", "-c", buildx)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Print(err)
		return &workflow.StepResult{
			Status: workflow.StepFailed,
		}, err
	}

	return &workflow.StepResult{
		Status: workflow.StepSuccess,
	}, nil
}

func (s *DockerBuildStep) Name() string {
	return s.name
}

func (s *DockerBuildStep) Description() string {
	return "Builds docker image"
}
