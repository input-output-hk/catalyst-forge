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
	// Git defines the source Git repository for the release.
	Git GitSpec `json:"git"`

	// ID is the unique identifier of the release.
	ID string `json:"id"`

	// Project is the name of the project within the source Git repository.
	Project string `json:"project"`

	// ProjectPath is the path to the project within the source Git repository.
	ProjectPath string `json:"project_path"`
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
