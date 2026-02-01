package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// HeaderRouteSpec defines the desired state of HeaderRoute
type HeaderRouteSpec struct {
	// HeaderName is the HTTP header name to match (e.g., "X-App", "X-Tenant")
	HeaderName string `json:"headerName"`

	// HeaderValue is the value to match for the header
	HeaderValue string `json:"headerValue"`

	// Backend defines where to route matching requests
	Backend BackendRef `json:"backend"`
}

// BackendRef references a Kubernetes Service
type BackendRef struct {
	// Name of the Service
	Name string `json:"name"`

	// Namespace of the Service (defaults to HeaderRoute namespace)
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Port number on the Service
	Port int32 `json:"port"`
}

// HeaderRouteStatus defines the observed state of HeaderRoute
type HeaderRouteStatus struct {
	// Ready indicates if the route is configured in Envoy
	Ready bool `json:"ready,omitempty"`

	// Message provides additional status information
	Message string `json:"message,omitempty"`

	// LastUpdated is the last time the route was processed
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Header",type=string,JSONPath=`.spec.headerName`
// +kubebuilder:printcolumn:name="Value",type=string,JSONPath=`.spec.headerValue`
// +kubebuilder:printcolumn:name="Backend",type=string,JSONPath=`.spec.backend.name`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`

// HeaderRoute is the Schema for the headerroutes API
type HeaderRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HeaderRouteSpec   `json:"spec,omitempty"`
	Status HeaderRouteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HeaderRouteList contains a list of HeaderRoute
type HeaderRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HeaderRoute `json:"items"`
}

// DeepCopyInto writes a deep copy of the receiver into out
func (in *HeaderRoute) DeepCopyInto(out *HeaderRoute) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy creates a deep copy of HeaderRoute
func (in *HeaderRoute) DeepCopy() *HeaderRoute {
	if in == nil {
		return nil
	}
	out := new(HeaderRoute)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject creates a deep copy as runtime.Object
func (in *HeaderRoute) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}

// DeepCopyInto writes a deep copy of HeaderRouteList into out
func (in *HeaderRouteList) DeepCopyInto(out *HeaderRouteList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]HeaderRoute, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy creates a deep copy of HeaderRouteList
func (in *HeaderRouteList) DeepCopy() *HeaderRouteList {
	if in == nil {
		return nil
	}
	out := new(HeaderRouteList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject creates a deep copy as runtime.Object
func (in *HeaderRouteList) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}
