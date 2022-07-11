package tools

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AddFinalizer add a finalizer in ObjectMeta.
func AddFinalizer(meta *metav1.ObjectMeta, finalizer string) {
	if HasFinalizer(meta, finalizer) {
		return
	}
	meta.Finalizers = append(meta.Finalizers, finalizer)
}

// HasFinalizer returns true if ObjectMeta has the finalizer.
func HasFinalizer(meta *metav1.ObjectMeta, finalizer string) bool {
	return sliceContainsString(meta.Finalizers, finalizer)
}

// RemoveFinalizer removes the finalizer from ObjectMeta.
func RemoveFinalizer(meta *metav1.ObjectMeta, finalizer string) {
	meta.Finalizers = removeStringFromSlice(meta.Finalizers, finalizer)
}

// sliceContainsString whether slice contains special string.
func sliceContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}

	return false
}

// removeStringFromSlice remove special string from slice.
func removeStringFromSlice(slice []string, s string) []string {
	result := make([]string, 0, len(slice))

	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}

	return result
}
