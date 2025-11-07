package providers

import (
	"context"
	"fmt"

	"github.com/tobiaskrok/topi/shared/workflow"
	v1 "github.com/tobiaskrok/topi/topi-operator/api/v1"

	batchv1 "k8s.io/api/batch/v1"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DockerBuildProvider struct {
}

func init() {
	register(&DockerBuildProvider{})
}

func (g *DockerBuildProvider) Name() string {
	return "Docker Build Provider"
}

func (g *DockerBuildProvider) Preflight(ctx context.Context, cl client.Client) error {

	// var we need to check if buildkit is installed and running
	var buildkit appsv1.Deployment
	if err := cl.Get(ctx, types.NamespacedName{
		Name:      "buildkit",
		Namespace: "topi-system",
	}, &buildkit); err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("'buildkit' deployment cannot be found in namespace 'topi-system', has it been deployed?")
		}
		return err
	}
	if buildkit.Status.AvailableReplicas == 0 {
		return fmt.Errorf("no running containers in the 'buildkit' deployment in namespace 'topi-system', please check its logs")
	}
	return nil
}

func (g *DockerBuildProvider) IsRequired(cfg workflow.WorkflowConfig) (bool, error) {
	for _, job := range cfg.Jobs {
		for _, step := range job.Steps {
			if step.Uses == "docker/build" {
				return true, nil
			}
			continue
		}
	}
	return false, nil
}

func (g *DockerBuildProvider) Inject(ctx context.Context, job *batchv1.Job, buildJob *v1.BuildJob) error {
	return nil
}
