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

//+kubebuilder:validation:MinProperties=1

type EventResource struct {
	// API version of the involved object.
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`

	// Kind of the involved object.
	// +optional
	Kind string `json:"kind,omitempty"`

	// Name of the involved object.
	// +optional
	Name string `json:"name"`

	// Namespace of the involved object.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

type EventFilter struct {
	// Type of events to watch.
	// +kubebuilder:validation:Enum=Normal;Warning
	// +optional
	Type string `json:"type,omitempty"`

	// Reasons list of event reasons to watch.
	// +optional
	Reasons []string `json:"reasons,omitempty"`

	// Resources list of event's involved objects to watch.
	// +optional
	Resources []EventResource `json:"resources,omitempty"`
}

func (f *EventFilter) Match(event *v1.Event) bool {
	if f.Type != "" {
		if f.Type != event.Type {
			return false
		}
	}
	if f.Reasons != nil {
		var matched bool
		for _, reason := range f.Reasons {
			if reason == event.Reason {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if f.Resources != nil {
		var matched bool
		for _, resource := range f.Resources {
			m := true
			if resource.APIVersion != "" {
				if resource.APIVersion != event.InvolvedObject.APIVersion {
					m = false
				}
			}
			if resource.Kind != "" {
				if resource.Kind != event.InvolvedObject.Kind {
					m = false
				}
			}
			if resource.Name != "" {
				if resource.Name != event.InvolvedObject.Name {
					m = false
				}
			}
			if resource.Namespace != "" {
				if resource.Namespace != event.InvolvedObject.Namespace {
					m = false
				}
			}
			if m {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// EventSetSpec defines the desired state of EventSet
type EventSetSpec struct {
	//+required
	Match EventFilter `json:"match"`
	//+required
	SinkRefs []v1.LocalObjectReference `json:"sinkRefs,omitempty"`
}

// EventSetStatus defines the observed state of EventSet
type EventSetStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// EventSet is the Schema for the eventsets API
type EventSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventSetSpec   `json:"spec,omitempty"`
	Status EventSetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster

// EventSetList contains a list of EventSet
type EventSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EventSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EventSet{}, &EventSetList{})
}
