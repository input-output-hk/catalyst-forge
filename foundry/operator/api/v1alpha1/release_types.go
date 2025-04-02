/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ReleaseDeploymentSpec defines the desired state of Release.
type ReleaseDeploymentSpec struct {
	// ID is the identifier for this deployment.
	ID string `json:"id"`

	// ReleaseID is the identifier for the release this deployment belongs to.
	ReleaseID string `json:"release_id"`

	// TTL specifies the time to live for this deployment after completion (in seconds).
	// After this period has elapsed since completion, the operator will delete the resource.
	// +optional
	// +kubebuilder:default=300
	TTL int64 `json:"ttl,omitempty"`
}

// GitSpec defines the source Git repository for the release.
type GitSpec struct {
	// Ref is the Git reference to use for the release.
	Ref string `json:"ref"`

	// URL is the URL of the source Git repository for the release.
	URL string `json:"url"`
}

// ReleaseDeploymentStatus defines the observed state of Release.
type ReleaseDeploymentStatus struct {
	// +listType=map
	// +listMapKey=type
	// +patchStrategy=merge
	// +patchMergeKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// State is the current state of the release.
	State string `json:"state"`

	// CompletionTime represents the time when this deployment completed (succeeded or failed).
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ReleaseDeployment is the Schema for the release deployments API.
type ReleaseDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReleaseDeploymentSpec   `json:"spec,omitempty"`
	Status ReleaseDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ReleaseDeploymentList contains a list of ReleaseDeployment.
type ReleaseDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ReleaseDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ReleaseDeployment{}, &ReleaseDeploymentList{})
}
