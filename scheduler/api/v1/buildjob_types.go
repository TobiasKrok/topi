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
}

// BuildJobStatus defines the observed state of BuildJob.
type BuildJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status        *string          `json:"artefactName"`
	BuildStart    *metav1.Time     `json:"buildStart"`
	BuildEnd      *metav1.Time     `json:"buildEnd"`
	BuildDuratioh *metav1.Duration `json:"duration"`
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
