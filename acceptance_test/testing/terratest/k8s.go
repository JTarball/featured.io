package terratest

import (
	"fmt"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/stretchr/testify/require"
)

// WaitUntilPodReady waits until all of the containers within the pod are ready and started, retrying the check for the specified amount of times, sleeping
// for the provided duration between each try. This will fail the test if there is an error or if the check times out.
func WaitUntilPodReady(t *testing.T, options *k8s.KubectlOptions, podName string, retries int, sleepBetweenRetries time.Duration) {
	require.NoError(t, WaitUntilPodReadyE(t, options, podName, retries, sleepBetweenRetries))
}

// WaitUntilPodReadyE waits until all of the containers within the pod are ready and started, retrying the check for the specified amount of times, sleeping
// for the provided duration between each try.
func WaitUntilPodReadyE(t *testing.T, options *k8s.KubectlOptions, podName string, retries int, sleepBetweenRetries time.Duration) error {
	statusMsg := fmt.Sprintf("Wait for pod %s to be ready.", podName)
	message, err := retry.DoWithRetryE(
		t,
		statusMsg,
		retries,
		sleepBetweenRetries,
		func() (string, error) {
			pod, err := k8s.GetPodE(t, options, podName)
			if err != nil {
				return "", err
			}
			if !IsPodReady(pod) {
				return "", NewPodNotReadyError(pod)
			}
			return "Pod is now ready", nil
		},
	)
	if err != nil {
		logger.Logf(t, "Timedout waiting for Pod to be ready: %s", err)
		return err
	}
	logger.Logf(t, message)
	return nil
}

// IsPodReady returns true if the all of the containers within the pod are ready and started
func IsPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		ctype := condition.Type
		cstatus := condition.Status

		if ctype == corev1.PodReady && cstatus == "True" {
			return true
		}
	}
	return false
}



// GetDeployment returns a Kubernetes pod resource in the provided namespace with the given name. This will
// fail the test if there is an error.
func GetDeployment(t *testing.T, options *k8s.KubectlOptions, deploymentName string) *appsv1.Deployment {
	pod, err := GetDeploymentE(t, options, deploymentName)
	require.NoError(t, err)
	return pod
}

// GetDeploymentE returns a Kubernetes pod resource in the provided namespace with the given name.
func GetDeploymentE(t *testing.T, options *k8s.KubectlOptions, deploymentName string) (*appsv1.Deployment, error) {
	clientset, err := k8s.GetKubernetesClientFromOptionsE(t, options)
	if err != nil {
		return nil, err
	}

	return clientset.AppsV1().Deployments(options.Namespace).Get(deploymentName, metav1.GetOptions{})
}

// GetDeploymentImageTag gets the image tag from a deployment given its name, failing the test if we can't find the deployment
// Current only assumes one container per deployment
func GetDeploymentImageTag(t *testing.T, options *k8s.KubectlOptions, deploymentName string) string {
	deployment := GetDeployment(t, options, deploymentName)
	tag := strings.SplitN(deployment.Spec.Template.Spec.Containers[0].Image, ":", -1)
	return tag[1]
}

// NamespaceLabelE is a wrapper for annotating a k8s kubectl label namespace with a label, failing the test if there is an error
func NamespaceLabel(t *testing.T, options *k8s.KubectlOptions, namespace string, args ...string) {
	require.NoError(t, NamespaceLabelE(t, options, namespace, args...))
}

// NamespaceLabelE is a wrapper for annotating a k8s kubectl label namespace with a label.
func NamespaceLabelE(t *testing.T, options *k8s.KubectlOptions, namespace string, args ...string) error {
	cmdArgs := append([]string{"label", "namespaces", namespace}, args...)
	return k8s.RunKubectlE(t, options, cmdArgs...)
}

// NamespaceAnnotate is a wrapper for annotating a k8s kubectl label namespace with a label, failing the test if there is an error
func NamespaceAnnotate(t *testing.T,  options *k8s.KubectlOptions, namespace string, args ...string) {
	require.NoError(t, NamespaceAnnotateE(t, options, namespace, args...))
}

// NamespaceAnnotate is a wrapper for annotating a k8s kubectl label namespace with a label.
func NamespaceAnnotateE(t *testing.T,  options *k8s.KubectlOptions, namespace string, args ...string) error {
	cmdArgs := append([]string{"annotate", "namespace", namespace}, args...)
	return k8s.RunKubectlE(t, options, cmdArgs...)
}
