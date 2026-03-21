/*
Copyright 2026.

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

// AuthentikSource defines which Authentik application to read from
type AuthentikSource struct {
	// ApplicationSlug is the slug of the Authentik application
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	ApplicationSlug string `json:"applicationSlug"`
}

// SecretTarget defines where to write the OIDC credentials
type SecretTarget struct {
	// Namespace to create the secret in
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`

	// SecretName is the name of the Secret to create/update
	// +kubebuilder:validation:Required
	SecretName string `json:"secretName"`
}

// RolloutTargetRef identifies the workload to restart
type RolloutTargetRef struct {
	// Kind of the target (Deployment or StatefulSet)
	// +kubebuilder:validation:Enum=Deployment;StatefulSet
	Kind string `json:"kind"`

	// Name of the target resource
	Name string `json:"name"`

	// Namespace of the target resource
	Namespace string `json:"namespace"`
}

// ConfigMapTarget defines a ConfigMap key to patch with profile-generated content.
// The profile must produce a key matching SourceKey (e.g. "service_conf_yaml" for ragflow).
// The operator patches the ConfigMap's DataKey with the value from that profile key.
type ConfigMapTarget struct {
	// Namespace of the ConfigMap
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`

	// Name of the ConfigMap to patch
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// DataKey is the key inside the ConfigMap's data to write to
	// +kubebuilder:validation:Required
	DataKey string `json:"dataKey"`

	// SourceKey is the profile-generated secret key to read the value from
	// +kubebuilder:validation:Required
	SourceKey string `json:"sourceKey"`
}

// RolloutRestart configures automatic workload restart on secret changes
type RolloutRestart struct {
	// Enabled controls whether rollout restart is active
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	// TargetRef identifies the workload to restart
	// +optional
	TargetRef *RolloutTargetRef `json:"targetRef,omitempty"`
}

// OIDCClientSpec defines the desired state of OIDCClient
type OIDCClientSpec struct {
	// Authentik defines which Authentik application to read from
	// +kubebuilder:validation:Required
	Authentik AuthentikSource `json:"authentik"`

	// Target defines where to write the OIDC credentials
	// +kubebuilder:validation:Required
	Target SecretTarget `json:"target"`

	// SecretProfile selects a built-in key mapping profile
	// +kubebuilder:validation:Enum=grafana;openwebui;argocd;ragflow;generic
	// +kubebuilder:default=generic
	SecretProfile string `json:"secretProfile"`

	// SecretOverrides adds or overrides keys in the generated secret
	// +optional
	SecretOverrides map[string]string `json:"secretOverrides,omitempty"`

	// ConfigMapTarget patches a ConfigMap with a profile-generated value
	// +optional
	ConfigMapTarget *ConfigMapTarget `json:"configMapTarget,omitempty"`

	// RolloutRestart configures automatic workload restart on secret changes
	// +optional
	RolloutRestart *RolloutRestart `json:"rolloutRestart,omitempty"`
}

// OIDCClientStatus defines the observed state of OIDCClient
type OIDCClientStatus struct {
	// Conditions represent the latest observations of the resource's state
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastSyncTime is the last time the operator synced with Authentik
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// SecretHash is the SHA256 hash of the current secret data
	// +optional
	SecretHash string `json:"secretHash,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Slug",type=string,JSONPath=`.spec.authentik.applicationSlug`
// +kubebuilder:printcolumn:name="Profile",type=string,JSONPath=`.spec.secretProfile`
// +kubebuilder:printcolumn:name="Target NS",type=string,JSONPath=`.spec.target.namespace`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="AuthentikProviderFound")].status`
// +kubebuilder:printcolumn:name="Synced",type=string,JSONPath=`.status.conditions[?(@.type=="SecretSynced")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:resource:shortName=oidc

// OIDCClient is the Schema for the oidcclients API
type OIDCClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OIDCClientSpec   `json:"spec,omitempty"`
	Status OIDCClientStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OIDCClientList contains a list of OIDCClient
type OIDCClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OIDCClient `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OIDCClient{}, &OIDCClientList{})
}
