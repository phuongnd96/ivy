/*
Copyright 2022.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BigTableRestoreSpec defines the desired state of BigTableRestore
type BigTableRestoreSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	BackUpID        string `json:"backUpId"`
	SourceProjectID string `json:"sourceProjectId"`
	TargetProjectID string `json:"targetProjectId"`
	SourceInstance  string `json:"sourceInstance"`
	SourceCluster   string `json:"sourceCluster"`
	TargetInstance  string `json:"targetInstance"`
	TargetCluster   string `json:"targetCluster"`
	TargetTable     string `json:"targetTable"`
}

// BigTableRestoreStatus defines the observed state of BigTableRestore
type BigTableRestoreStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status             string `json:"status,omitempty"`
	ObservedGeneration int64  `json:"observedGeneration,omitempty" protobuf:"varint,1,opt,name=observedGeneration"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BigTableRestore is the Schema for the bigtablerestores API
type BigTableRestore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BigTableRestoreSpec   `json:"spec,omitempty"`
	Status BigTableRestoreStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BigTableRestoreList contains a list of BigTableRestore
type BigTableRestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BigTableRestore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BigTableRestore{}, &BigTableRestoreList{})
}
