package providers

import (
	"context"
	"fmt"

	"github.com/tobiaskrok/topi/shared/workflow"
	v1 "github.com/tobiaskrok/topi/topi-operator/api/v1"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MinioProvider struct {
}

func init() {
	register(&MinioProvider{})
}

func (m *MinioProvider) Name() string {
	return "MinIO Provider"
}

// TODO: maybe protect from deletion after checking it exists?
// TODO: validate the configuration?
func (m *MinioProvider) Preflight(ctx context.Context, cl client.Client) error {
	var secret corev1.Secret
	if err := cl.Get(ctx, types.NamespacedName{
		Name:      "provider-minio",
		Namespace: "topi-builds",
	}, &secret); err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("could not find secret 'provider-minio' in namespace 'topi-builds'")
		}
		return err
	}

	var config corev1.ConfigMap
	if err := cl.Get(ctx, types.NamespacedName{
		Name:      "provider-minio",
		Namespace: "topi-builds",
	}, &config); err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("could not find config 'provider-minio' in namespace 'topi-builds'")
		}
		return err
	}

	accessSecret := string(secret.Data["accessSecret"])
	accessKey := string(secret.Data["accessKey"])

	endpoint := config.Data["endpoint"]
	bucket := config.Data["bucket"]

	if accessSecret == "" {
		return fmt.Errorf("'accessSecret' is missing from 'provider-minio' secret")
	}

	if accessKey == "" {
		return fmt.Errorf("'accessKey' is missing from 'provider-minio' secret")
	}
	if endpoint == "" {
		return fmt.Errorf("'endpoint' is missing from 'provider-minio' configmap")
	}
	if bucket == "" {

		return fmt.Errorf("'bucket' is missing from 'provider-minio' configmap")
	}

	return nil
}

func (m *MinioProvider) IsRequired(cfg workflow.WorkflowConfig) (bool, error) {
	for _, job := range cfg.Jobs {
		for _, step := range job.Steps {
			if step.Uses == "artefact/minio" {
				return true, nil
			}
		}
	}
	return false, nil
}

func (m *MinioProvider) Inject(ctx context.Context, job *batchv1.Job, buildJob *v1.BuildJob) error {
	job.Spec.Template.Spec.Containers[0].Env = append(
		job.Spec.Template.Spec.Containers[0].Env,
		corev1.EnvVar{
			Name: "MINIO_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{

					LocalObjectReference: corev1.LocalObjectReference{
						Name: "provider-minio",
					},
					Key: "accessSecret",
				},
			},
		},
		corev1.EnvVar{
			Name: "MINIO_ID",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{

					LocalObjectReference: corev1.LocalObjectReference{
						Name: "provider-minio",
					},
					Key: "accessKey",
				},
			},
		},
		corev1.EnvVar{
			Name: "MINIO_ENDPOINT",
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "provider-minio",
					},
					Key: "endpoint",
				},
			},
		},
		corev1.EnvVar{
			Name: "MINIO_ARTEFACT_BUCKET",
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "provider-minio",
					},
					Key: "bucket",
				},
			},
		},
		corev1.EnvVar{
			Name: "MINIO_USE_SSL",
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "provider-minio",
					},
					Key: "use-ssl",
				},
			},
		},
	)

	return nil
}
