package providers

import (
	"context"
	"fmt"

	"github.com/tobiaskrok/topi/shared/workflow"
	v1 "github.com/tobiaskrok/topi/topi-operator/api/v1"

	batchv1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	// Providers
)

var (
	registry = make(map[string]ResourceProvider)
)

func register(p ResourceProvider) {
	registry[p.Name()] = p
}

type ResourceProvider interface {
	Name() string
	Preflight(ctx context.Context, cl client.Client) error
	IsRequired(wf workflow.WorkflowConfig) (bool, error)
	Inject(ctx context.Context, job *batchv1.Job, buildJob *v1.BuildJob) error
}

func GetProvidersFromWorkflow(ctx context.Context, wf workflow.WorkflowConfig) ([]string, error) {

	log := logf.FromContext(ctx)
	var providers []string
	for _, provider := range registry {

		required, err := provider.IsRequired(wf)
		if err != nil {
			return nil, fmt.Errorf("provider %s failed requirement check: %w", provider.Name(), err)
		}
		if !required {
			log.V(1).Info("Provider not required, skipping", "name", provider.Name())
			continue
		}

		log.V(1).Info("Added provider", "name", provider.Name())
		providers = append(providers, provider.Name())
	}
	return providers, nil
}

func Apply(ctx context.Context, cl client.Client, job *batchv1.Job, buildJob *v1.BuildJob, providers []string) error {
	log := logf.FromContext(ctx)

	for _, p := range providers {
		provider := registry[p]
		log.Info("Running pre-flight checks for ", "name", provider.Name())
		if err := provider.Preflight(ctx, cl); err != nil {
			return fmt.Errorf("provider %s preflight failed: %w", provider.Name(), err)
		}

		log.Info("Injecting resources from provider", "name", provider.Name())

		if err := provider.Inject(ctx, job, buildJob); err != nil {
			return fmt.Errorf("provider %s injection failed: %w", provider.Name(), err)
		}

		log.Info("Provider completed successfully", "name", provider.Name())
	}

	return nil
}
