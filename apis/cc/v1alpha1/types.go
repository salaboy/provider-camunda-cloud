/*
Copyright 2020 The Crossplane Authors.

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
	cc "github.com/camunda-community-hub/camunda-cloud-go-client/pkg/cc/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// MyTypeParameters are the configurable fields of a MyType.
type MyTypeParameters struct {
	ConfigurableField string `json:"configurableField"`
}

// MyTypeObservation are the observable fields of a MyType.
type MyTypeObservation struct {
	ObservableField string `json:"observableField,omitempty"`
}

// A ZeebeClusterSpec defines the desired state of a ZeebeCluster.
type ZeebeClusterSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       MyTypeParameters `json:"forProvider"`
	// +kubebuilder:validation:Optional
	Region string `json:"region"`
	// +kubebuilder:validation:Optional
	ChannelName string `json:"channelName"`
	// +kubebuilder:validation:Optional
	GenerationName string `json:"generationName"`
	// +kubebuilder:validation:Optional
	PlanName string `json:"planName"`
}

// A ZeebeClusterStatus represents the observed state of a ZeebeCluster.
type ZeebeClusterStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          MyTypeObservation `json:"atProvider,omitempty"`
	ClusterId string `json:"clusterId"`
	ClusterStatus cc.ClusterStatus `json:"clusterStatus"`
}

// +kubebuilder:object:root=true
// A ZeebeCluster is a remote ZeebeCluster in Camunda Cloud API type
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.clusterStatus.ready"
// +kubebuilder:printcolumn:name="CLUSTER ID",type="string",JSONPath=".status.clusterId"
// +kubebuilder:printcolumn:name="PLAN",type="string",JSONPath=".spec.planName"
// +kubebuilder:printcolumn:name="CHANNEL",type="string",JSONPath=".spec.channelName"
// +kubebuilder:printcolumn:name="GENERATION",type="string",JSONPath=".spec.generationName"
// +kubebuilder:printcolumn:name="REGION",type="string",JSONPath=".spec.region"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,shortName=zb
type ZeebeCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ZeebeClusterSpec   `json:"spec"`
	Status ZeebeClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// ZeebeClusterList contains a list of ZeebeClusters
type ZeebeClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ZeebeCluster `json:"items"`
}
