package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BuildJobSpec defines the desired state of BuildJob
type BuildJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	Repository   *string `json:"repository"`
	Ref          *string `json:"ref"`
	Owner        *string `json:"owner"`
	ArtefactName *string `json:"artefactName"`
	// +optional
	Suspend *bool `json:"suspend"`
}

// BuildJobPhase represents the current phase of the build job
// +kubebuilder:validation:Enum=Pending;Queued;Running;Succeeded;Failed;Cancelled
type BuildJobPhase string

const (
	// BuildJobPhasePending means the build job has been created but not yet validated
	BuildJobPhasePending BuildJobPhase = "Pending"
	// BuildJobPhaseQueued means the build job is queued and waiting for resources
	BuildJobPhaseQueued BuildJobPhase = "Queued"
	// BuildJobPhaseRunning means the build job is actively running
	BuildJobPhaseRunning BuildJobPhase = "Running"
	// BuildJobPhaseSucceeded means the build job completed successfully
	BuildJobPhaseSucceeded BuildJobPhase = "Succeeded"
	// BuildJobPhaseFailed means the build job failed
	BuildJobPhaseFailed BuildJobPhase = "Failed"
	// BuildJobPhaseCancelled means the build job was cancelled
	BuildJobPhaseCancelled BuildJobPhase = "Cancelled"
)

// BuildJobStatus defines the observed state of BuildJob.
type BuildJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Phase represents the current phase of the build job
	// +optional
	Phase BuildJobPhase `json:"phase,omitempty"`

	// BuildStart is when the build started
	// +optional
	BuildStart *metav1.Time `json:"buildStart,omitempty"`

	// BuildEnd is when the build ended
	// +optional
	BuildEnd *metav1.Time `json:"buildEnd,omitempty"`

	// BuildDuration is how long the build took
	// +optional
	BuildDuration *metav1.Duration `json:"duration,omitempty"`

	// BuildID is the unique identifier for this build
	// +optional
	BuildID *string `json:"buildid,omitempty"`

	// The commit sha, used to find the required config map
	// +optional
	CommitSha string `json:"commitsha,omitempty"`

	// RequiredProviders lists the providers needed by the workflow
	// +optional
	RequiredProviders []string `json:"requiredProviders,omitempty"`

	// Message provides additional context about the current phase
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// BuildJob is the Schema for the buildjobs API
type BuildJob struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of BuildJob
	// +required
	Spec BuildJobSpec `json:"spec"`

	// status defines the observed state of BuildJob
	// +optional
	Status BuildJobStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// BuildJobList contains a list of BuildJob
type BuildJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BuildJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BuildJob{}, &BuildJobList{})
}
