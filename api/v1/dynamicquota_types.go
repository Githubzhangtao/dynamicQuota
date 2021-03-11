/*


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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DynamicQuotaSpec defines the desired state of DynamicQuota
type DynamicQuotaSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	//+kubebuilder:validation:MinItems=1

	// A list of name of the specify namespace，不为空
	NameSpaces []string `json:"nameSpaces,omitempty"`

	// Specifies which policy that we will choose to compute the change ratio
	// +optional
	ChangePolicy ChangePolicy `json:"changePolicy"`
}

// ChangePolicy describes which policy that we will choose to compute the change ratio
// +kubebuilder:validation:Enum=min;mean;max

type ChangePolicy string

const (
	MinPolicy  ChangePolicy = "min"
	MeanPolicy ChangePolicy = "mean"
	MaxPolicy  ChangePolicy = "max"
)

// DynamicQuotaStatus defines the observed state of DynamicQuota
type DynamicQuotaStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	TotalCpu *int32   `json:"totalCpu"`
	TotalMem *int32   `json:"totalMem"`
	TotalNet *int32   `json:"totalNet"`
	NodeList []string `json:"nodeList"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status

// DynamicQuota is the Schema for the dynamicquota API
type DynamicQuota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DynamicQuotaSpec   `json:"spec,omitempty"`
	Status DynamicQuotaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DynamicQuotaList contains a list of DynamicQuota
type DynamicQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DynamicQuota `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DynamicQuota{}, &DynamicQuotaList{})
}
