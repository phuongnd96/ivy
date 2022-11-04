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

// BigTableBackUpSpec defines the desired state of BigTableBackUp
type BigTableBackUpSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of BigTableBackUp. Edit bigtablebackup_types.go to remove/update
	ProjectID      string `json:"projectId"`
	SourceInstance string `json:"sourceInstance"`
	SourceCluster  string `json:"sourceCluster"`
	SourceTable    string `json:"sourceTable"`
}

// BigTableBackUpStatus defines the observed state of BigTableBackUp
type BigTableBackUpStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status             string `json:"status,omitempty"`
	ObservedGeneration int64  `json:"observedGeneration,omitempty" protobuf:"varint,1,opt,name=observedGeneration"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BigTableBackUp is the Schema for the bigtablebackups API
type BigTableBackUp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BigTableBackUpSpec   `json:"spec,omitempty"`
	Status BigTableBackUpStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BigTableBackUpList contains a list of BigTableBackUp
type BigTableBackUpList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BigTableBackUp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BigTableBackUp{}, &BigTableBackUpList{})
}
