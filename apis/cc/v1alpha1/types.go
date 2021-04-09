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

// ZeebeClusterParameters are the configurable fields of a ZeebeCluster.
type ZeebeClusterParameters struct {
	// +kubebuilder:validation:Optional
	Region string `json:"region"`
	// +kubebuilder:validation:Optional
	ChannelName string `json:"channelName"`
	// +kubebuilder:validation:Optional
	GenerationName string `json:"generationName"`
	// +kubebuilder:validation:Optional
	PlanName string `json:"planName"`
}

// ZeebeClusterObservation are the observable fields of a ZeebeCluster.
type ZeebeClusterObservation struct {
	ClusterId string `json:"clusterId"`
	ClusterStatus cc.ClusterStatus `json:"clusterStatus"`
}

// A ZeebeClusterSpec defines the desired state of a ZeebeCluster.
type ZeebeClusterSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ZeebeClusterParameters `json:"forProvider"`
}

// A ZeebeClusterStatus represents the observed state of a ZeebeCluster.
type ZeebeClusterStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ZeebeClusterObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// A ZeebeCluster is a remote ZeebeCluster in Camunda Cloud API type
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.atProvider.clusterStatus.ready"
// +kubebuilder:printcolumn:name="CLUSTER ID",type="string",JSONPath=".status.atProvider.clusterId"
// +kubebuilder:printcolumn:name="PLAN",type="string",JSONPath=".spec.forProvider.planName"
// +kubebuilder:printcolumn:name="CHANNEL",type="string",JSONPath=".spec.forProvider.channelName"
// +kubebuilder:printcolumn:name="GENERATION",type="string",JSONPath=".spec.forProvider.generationName"
// +kubebuilder:printcolumn:name="REGION",type="string",JSONPath=".spec.forProvider.region"
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
