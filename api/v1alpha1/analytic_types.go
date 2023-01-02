/*
Copyright 2023.

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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AnalyticSpec defines the desired state of Analytic
type AnalyticSpec struct {
	Query   string                  `json:"query"`
	SinkRef v1.LocalObjectReference `json:"sinkRef"`
}

// AnalyticStatus defines the observed state of Analytic
type AnalyticStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Analytic is the Schema for the analytics API
type Analytic struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AnalyticSpec   `json:"spec,omitempty"`
	Status AnalyticStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AnalyticList contains a list of Analytic
type AnalyticList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Analytic `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Analytic{}, &AnalyticList{})
}
