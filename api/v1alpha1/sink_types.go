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
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SinkReadyCondition = "Ready"
)

type FileSink struct {
	// Path file path
	// +required
	Path string `json:"path"`
}

type SqliteSink struct {
	// Path database file path
	// +required
	Path string `json:"path"`
}

type WebhookSink struct {
	// Endpoint webhook url
	// +required
	Endpoint string `json:"endpoint"`

	// Endpoint webhook url
	// +required
	Headers map[string]string `json:"headers"`
}

type ElasticSink struct {
	// Endpoint elastic address.
	// +required
	Address string `json:"address"`

	// IndexName elastic index name to write the events to.
	// +required
	IndexName string `json:"indexName"`

	// Username elastic username, can be put in the secret referenced in secretRef.
	// +optional
	Username string `json:"username"`

	// Password elastic password, can be put in the secret referenced in secretRef.
	// +optional
	Password string `json:"password"`

	// Mode elastic inertion mode.
	// +optional
	Mode string `json:"mode"`

	// BatchSize bulk create batch size.
	// +kubebuilder:default:=10
	// +optional
	BatchSize int `json:"batchSize"`

	// BatchExpiry bulk create batch exprity in seconds
	//+kubebuilder:default:=10
	// +optional
	BatchExpiry int `json:"batchExpiry"`
}

func (os *ElasticSink) SetSecretConf(secretConf map[string]string) {
	if username, ok := secretConf["username"]; ok {
		os.Username = username
	}
	if password, ok := secretConf["password"]; ok {
		os.Password = password
	}
}

// SinkSpec defines the desired state of Sink
type SinkSpec struct {
	// File save events to file.
	// +optional
	File *FileSink `json:"file,omitempty"`

	// SQLite save events to sqlite database.
	// +optional
	SQLite *SqliteSink `json:"sqlite,omitempty"`

	// Webhook send events to generic webhook.
	// +optional
	Webhook *WebhookSink `json:"webhook,omitempty"`

	// Elastic save events to elastic.
	// +optional
	Elastic *ElasticSink `json:"elastic,omitempty"`

	// SecretRef secret reference to get secret configs from.
	// +optional
	SecretRef *v1.SecretReference `json:"secretRef,omitempty"`
}

// SinkStatus defines the observed state of Sink
type SinkStatus struct {
	// ObservedGeneration is the last observed generation of the Sink
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions holds the conditions for the Sink.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=`.status.conditions[?(@.type=="Ready")].reason`

// Sink is the Schema for the sinks API
type Sink struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SinkSpec   `json:"spec,omitempty"`
	Status SinkStatus `json:"status,omitempty"`
}

func (s *Sink) MarkAsReady(message, reason string) {
	cond := metav1.Condition{
		Type:               SinkReadyCondition,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: s.Generation,
		Message:            message,
		Reason:             reason,
		LastTransitionTime: metav1.Now(),
	}
	apimeta.SetStatusCondition(&s.Status.Conditions, cond)
}

func (s *Sink) MarkAsNotReady(message, reason string) {
	cond := metav1.Condition{
		Type:               SinkReadyCondition,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: s.Generation,
		Message:            message,
		Reason:             reason,
		LastTransitionTime: metav1.Now(),
	}
	apimeta.SetStatusCondition(&s.Status.Conditions, cond)
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster

// SinkList contains a list of Sink
type SinkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Sink `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Sink{}, &SinkList{})
}
