package providers

import (
	"context"
	"fmt"
	"log"

	v1 "github.com/tobiaskrok/topi/scheduler/api/v1"
	"github.com/tobiaskrok/topi/shared/workflow"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GitProvider struct {
}

func init() {
	register(&GitProvider{})
}

func (g *GitProvider) Name() string {
	return "Git Provider"
}

func (g *GitProvider) Preflight(ctx context.Context, cl client.Client) error {
	var secret corev1.Secret
	if err := cl.Get(ctx, types.NamespacedName{
		Name:      "git-token",
		Namespace: "topi-providers",
	}, &secret); err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("could not find secret 'git-token' in namespace 'topi-providers'")
		}
		return err
	}

	token := string(secret.Data["token"])
	if token == "" {
		return fmt.Errorf("'token' is missing from 'git-token' secret")
	}

	return nil
}

func (g *GitProvider) IsRequired(cfg workflow.WorkflowConfig) (bool, error) {

	for _, job := range cfg.Jobs {
		for _, step := range job.Steps {
			log.Default().Printf("USES: %s", step.Uses)
			if step.Uses == "git/checkout" {
				return true, nil
			}
			continue
		}
	}
	return false, nil
}

func (g *GitProvider) Inject(ctx context.Context, job *batchv1.Job, buildJob *v1.BuildJob) error {
	// TODO: Inject GIT_TOKEN environment variable from secret
	// Example:
	// job.Spec.Template.Spec.Containers[0].Env = append(
	//     job.Spec.Template.Spec.Containers[0].Env,
	//     corev1.EnvVar{
	//         Name: "GIT_TOKEN",
	//         ValueFrom: &corev1.EnvVarSource{
	//             SecretKeyRef: &corev1.SecretKeySelector{
	//                 LocalObjectReference: corev1.LocalObjectReference{
	//                     Name: "git-token",
	//                 },
	//                 Key: "token",
	//             },
	//         },
	//     },
	// )

	return nil
}
