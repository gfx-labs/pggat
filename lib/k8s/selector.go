package k8s

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NamespaceMatcher provides namespace and label-based filtering for Kubernetes resources
type NamespaceMatcher struct {
	// Namespace to filter resources
	// If empty, matches all namespaces
	Namespace string `json:"namespace,omitempty"`

	// Labels is a map of label key-value pairs to filter resources
	// Only resources matching ALL specified labels will match
	Labels map[string]string `json:"labels,omitempty"`
}

// LabelSelector returns a Kubernetes label selector string suitable for use with ListOptions
// Returns empty string if no labels are specified
func (nm *NamespaceMatcher) LabelSelector() string {
	if len(nm.Labels) == 0 {
		return ""
	}

	parts := make([]string, 0, len(nm.Labels))
	for key, value := range nm.Labels {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(parts, ",")
}

// TweakListOptions applies namespace and label filtering to list operations
// Can be used with Kubernetes informer factories and list calls
func (nm *NamespaceMatcher) TweakListOptions(opts *metav1.ListOptions) {
	if len(nm.Labels) > 0 {
		selector := nm.LabelSelector()
		if opts.LabelSelector != "" {
			// Combine with existing selector
			opts.LabelSelector = opts.LabelSelector + "," + selector
		} else {
			opts.LabelSelector = selector
		}
	}
}

// MatchesNamespace checks if a namespace name matches this matcher's namespace filter
func (nm *NamespaceMatcher) MatchesNamespace(namespace string) bool {
	// Check namespace match (empty namespace in matcher means match all)
	if nm.Namespace != "" && nm.Namespace != namespace {
		return false
	}
	return true
}

// MatchesNamespaceLabels checks if namespace labels match this matcher's label requirements
func (nm *NamespaceMatcher) MatchesNamespaceLabels(namespaceLabels map[string]string) bool {
	// Check label match (all labels in matcher must be present and match)
	for key, value := range nm.Labels {
		if namespaceLabels[key] != value {
			return false
		}
	}
	return true
}
