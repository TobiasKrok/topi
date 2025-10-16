package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	v1 "github.com/tobiaskrok/topi/scheduler/api/v1"
	"github.com/tobiaskrok/topi/scheduler/internal/controller/providers"
	"github.com/tobiaskrok/topi/shared/workflow"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

/*
Topi Scheduelr

The scehduler is responsible for controlling BuildJobs, scheduling them, list actice builds, cancel them



*/

const builderNamespace = "topi-builds"

// BuildJobReconciler reconciles a BuildJob object
type BuildJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=build.topi.tobias.no,resources=buildjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=build.topi.tobias.no,resources=buildjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=build.topi.tobias.no,resources=buildjobs/finalizers,verbs=update
// +kubebuilder:rbac:groups=build.topi.tobias.no,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=build.topi.tobias.no,resources=configmap,verbs=get;list;watch;
// +kubebuilder:rbac:groups=build.topi.tobias.no,resources=secrets,verbs=get;list;watch;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BuildJob object against the actual cluster state, and then:
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *BuildJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1. Fetch the BuildJob
	var buildJob v1.BuildJob
	if err := r.Get(ctx, req.NamespacedName, &buildJob); err != nil {
		// BuildJob was deleted, nothing to do
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Check if BuildJob is suspended (cancellation)
	if buildJob.Spec.Suspend != nil && *buildJob.Spec.Suspend {
		return r.handleSuspension(ctx, &buildJob)
	}

	// 3. Initialize phase if empty
	if buildJob.Status.Phase == "" {
		buildJob.Status.Phase = v1.BuildJobPhasePending
		if err := r.Status().Update(ctx, &buildJob); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// 4. State machine: handle current phase
	switch buildJob.Status.Phase {
	case v1.BuildJobPhasePending:
		return r.reconcilePending(ctx, &buildJob)

	case v1.BuildJobPhaseQueued:
		return r.reconcileQueued(ctx, &buildJob)

	case v1.BuildJobPhaseRunning:
		return r.reconcileRunning(ctx, &buildJob)

	case v1.BuildJobPhaseSucceeded, v1.BuildJobPhaseFailed, v1.BuildJobPhaseCancelled:
		// Terminal states - nothing to do
		log.V(1).Info("BuildJob in terminal state", "phase", buildJob.Status.Phase)
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// handleSuspension handles the suspension (cancellation) of a BuildJob
func (r *BuildJobReconciler) handleSuspension(ctx context.Context, buildJob *v1.BuildJob) (ctrl.Result, error) {
	// Only cancel if not already in a terminal state
	if buildJob.Status.Phase != v1.BuildJobPhaseCancelled &&
		buildJob.Status.Phase != v1.BuildJobPhaseSucceeded &&
		buildJob.Status.Phase != v1.BuildJobPhaseFailed {

		// TODO: Delete the Kubernetes Job if it exists
		// Check if Job exists and delete it

		buildJob.Status.Phase = v1.BuildJobPhaseCancelled
		if buildJob.Status.BuildEnd == nil {
			now := metav1.Now()
			buildJob.Status.BuildEnd = &now
		}
		if err := r.Status().Update(ctx, buildJob); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// reconcilePending handles the Pending phase
func (r *BuildJobReconciler) reconcilePending(ctx context.Context, buildJob *v1.BuildJob) (ctrl.Result, error) {
	// TODO: Fetch and parse workflow from ConfigMap
	// Check if ConfigMap exists
	// Validate workflow structure
	// Analyze workflow requirements
	if buildJob.Annotations["builder.topi.io/commit-sha"] == "" {
		return ctrl.Result{}, fmt.Errorf("build job is missing annotation 'builder.topi.io/commit-sha'")
	}
	var cfg corev1.ConfigMap
	if err := r.Get(ctx, types.NamespacedName{
		Name:      buildJob.Annotations["builder.topi.io/commit-sha"],
		Namespace: builderNamespace,
	}, &cfg); err != nil {
		if errors.IsNotFound(err) {
			// might need some more time
			return ctrl.Result{RequeueAfter: 6 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	wfYaml, ok := cfg.Data["workflow.yaml"]
	if !ok {
		return ctrl.Result{}, fmt.Errorf("workflow.yaml not found in ConfigMap")
	}

	wf, err := workflow.ParseWorkflow([]byte(wfYaml))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to parse workflow: %w", err)
	}
	buildID := r.generateBuildID()
	buildJob.Status.BuildID = &buildID
	buildJob.Status.Phase = v1.BuildJobPhaseQueued
	p, err := providers.GetProvidersFromWorkflow(ctx, wf)
	if err != nil {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, fmt.Errorf("failed to get the required providers: %w", err)
	}
	buildJob.Status.RequiredProviders = p
	if err := r.Status().Update(ctx, buildJob); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: true}, nil
}

func (r *BuildJobReconciler) reconcileQueued(ctx context.Context, buildJob *v1.BuildJob) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	jobName := r.getJobName(buildJob)
	var existingJob batchv1.Job
	err := r.Get(ctx, types.NamespacedName{
		Name:      jobName,
		Namespace: builderNamespace,
	}, &existingJob)

	if err == nil {
		log.Info("Job already exists, transitioning to Running", "jobName", jobName)
		buildJob.Status.Phase = v1.BuildJobPhaseRunning
		if err := r.Status().Update(ctx, buildJob); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if !errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	job := r.createJob(*buildJob)
	if err := providers.Apply(ctx, r.Client, job, buildJob, buildJob.Status.RequiredProviders); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to apply providers: %w", err)
	}

	if err := r.Create(ctx, job); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create Job: %w", err)
	}

	log.Info("Created Kubernetes Job", "jobName", jobName)

	now := metav1.Now()
	buildJob.Status.BuildStart = &now
	buildJob.Status.Phase = v1.BuildJobPhaseRunning

	if err := r.Status().Update(ctx, buildJob); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: true}, nil
}

// reconcileRunning handles the Running phase
func (r *BuildJobReconciler) reconcileRunning(ctx context.Context, buildJob *v1.BuildJob) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	jobName := r.getJobName(buildJob)
	var job batchv1.Job
	err := r.Get(ctx, types.NamespacedName{
		Name:      jobName,
		Namespace: builderNamespace,
	}, &job)

	if err != nil {
		if errors.IsNotFound(err) {
			buildJob.Status.Phase = v1.BuildJobPhaseFailed
			buildJob.Status.Message = "Kubernetes Job was deleted"
			now := metav1.Now()
			buildJob.Status.BuildEnd = &now
			if err := r.Status().Update(ctx, buildJob); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Check Job status
	if job.Status.Succeeded > 0 {
		// Build succeeded
		log.Info("Build succeeded", "jobName", jobName)
		buildJob.Status.Phase = v1.BuildJobPhaseSucceeded
		now := metav1.Now()
		buildJob.Status.BuildEnd = &now
		if buildJob.Status.BuildStart != nil {
			duration := now.Time.Sub(buildJob.Status.BuildStart.Time)
			buildJob.Status.BuildDuration = &metav1.Duration{Duration: duration}
		}
		if err := r.Status().Update(ctx, buildJob); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if job.Status.Failed > 0 {
		// Build failed
		log.Info("Build failed", "jobName", jobName)
		buildJob.Status.Phase = v1.BuildJobPhaseFailed
		buildJob.Status.Message = fmt.Sprintf("Job failed after %d attempts", job.Status.Failed)
		now := metav1.Now()
		buildJob.Status.BuildEnd = &now
		if buildJob.Status.BuildStart != nil {
			duration := now.Time.Sub(buildJob.Status.BuildStart.Time)
			buildJob.Status.BuildDuration = &metav1.Duration{Duration: duration}
		}
		if err := r.Status().Update(ctx, buildJob); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Still running
	log.V(1).Info("Build still running", "jobName", jobName, "active", job.Status.Active)
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BuildJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.BuildJob{}).
		Owns(&batchv1.Job{}). // Watch Jobs owned by BuildJob
		Named("buildjob").
		Complete(r)
}

func (r *BuildJobReconciler) generateBuildID() string {
	id := uuid.New()
	return id.String()[:8]
}

func (r *BuildJobReconciler) getJobName(buildJob *v1.BuildJob) string {
	if buildJob.Status.BuildID == nil {
		return ""
	}
	return buildJob.Name + "-" + *buildJob.Status.BuildID
}

func (r *BuildJobReconciler) createJob(buildJob v1.BuildJob) *batchv1.Job {

	var backoffLimit int32 = 4

	ref := "main" //TODO: probably needs to be customized
	if buildJob.Spec.Ref != nil && *buildJob.Spec.Ref != "" {
		ref = *buildJob.Spec.Ref
	}

	jobSpec := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildJob.Name + "-" + *buildJob.Status.BuildID,
			Namespace: builderNamespace,
			Labels: map[string]string{
				// "builder.topi.io/repo":     *buildJob.Spec.Repository,
				"builder.topi.io/artefact": *buildJob.Spec.ArtefactName,
			},
			Annotations: map[string]string{
				"builder.topi.io/build-id": *buildJob.Status.BuildID,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes: []corev1.Volume{
						{
							Name: "workflow-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: buildJob.Annotations["builder.topi.io/commit-sha"],
									},
								},
							},
						},
						{
							Name: "workspace",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "system",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "topi-builder",
							Image: "localhost:15001/topi-builder:latest", //TODO: make this configurable via env/config
							Env: []corev1.EnvVar{
								{
									Name:  "SOURCE_REPO",
									Value: *buildJob.Spec.Repository,
								},
								{
									Name:  "SOURCE_WORKFLOW",
									Value: "/opt/topi/workflow/workflow.yaml",
								},
								{
									Name:  "TOPI_WORKSPACE",
									Value: "/opt/topi/workspace",
								},
								{
									Name:  "TOPI_SYSTEM_DIR",
									Value: "/opt/topi/system",
								},
								{
									Name:  "BUILD_ID",
									Value: *buildJob.Status.BuildID,
								},
								{
									Name:  "SOURCE_REF",
									Value: ref,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "workflow-config",
									MountPath: "/opt/topi/workflow",
									ReadOnly:  true,
								},
								{
									Name:      "workspace",
									MountPath: "/opt/topi/workspace",
								},
								{
									Name:      "system",
									MountPath: "/opt/topi/system",
								},
							},
						},
					},
				},
			},
		},
	}

	return jobSpec
}
