package terratest

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)




// PodNotReady is returned when a Kubernetes service is not yet available to accept traffic.
type PodNotReady struct {
	pod *corev1.Pod
}

// Error is a simple function to return a formatted error message as a string
func (err PodNotReady) Error() string {
	return fmt.Sprintf("Pod %s is not available", err.pod.Name)
}

// NewPodNotReadyError returnes a PodNotReady struct when Kubernetes deems a pod is not available
func NewPodNotReadyError(pod *corev1.Pod) PodNotReady {
	return PodNotReady{pod}
}
