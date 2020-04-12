package terratest_test

import (
	"fmt"
	"testing"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tt "github.com/featured.io/acceptance_test/testing/terratest"
	th "github.com/featured.io/acceptance_test/testing/test-helper"
)

const EXAMPLE_DEPLOYMENT_YAML_TEMPLATE = `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: %s
spec:
  selector:
    matchLabels:
      app: nginx
  replicas: 2 # tells deployment to run 2 pods matching the template
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.15.7
        ports:
        - containerPort: 80
`

// Tests that we get a deployment from a namespace
func TestGetDeployment(t *testing.T) {

	kubet := th.NewKubeTest(t)
	defer kubet.CleanUp()
	tc := kubet.Config

	configData := fmt.Sprintf(EXAMPLE_DEPLOYMENT_YAML_TEMPLATE, tc.Namespace)
	defer func() {
		if !*th.SkipCleanUp {
			k8s.KubectlDeleteFromString(t, tc.KubeConfig, configData)
		}
	}()
	k8s.KubectlApplyFromString(t, tc.KubeConfig, configData)

	filters := metav1.ListOptions{LabelSelector: ""}
	k8s.WaitUntilNumPodsCreated(t, tc.KubeConfig, filters, 2, 2, 5000)

	deploymentName, _ := k8s.RunKubectlAndGetOutputE(t, tc.KubeConfig, "get", "deployment", "-n", tc.Namespace, "-o", "name")
	require.Equal(t, "deployment.apps/nginx-deployment", deploymentName)

	deployment := tt.GetDeployment(t, tc.KubeConfig, "nginx-deployment")
	require.Equal(t, deployment.Namespace, tc.Namespace)
}

// Tests that we can get a deployment from a namespace using GetDeploymentE
func TestGetDeploymentE(t *testing.T) {

	kubet := th.NewKubeTest(t)
	defer kubet.CleanUp()
	tc := kubet.Config

	configData := fmt.Sprintf(EXAMPLE_DEPLOYMENT_YAML_TEMPLATE, tc.Namespace)
	defer func() {
		if !*th.SkipCleanUp {
			k8s.KubectlDeleteFromString(t, tc.KubeConfig, configData)
		}
	}()
	k8s.KubectlApplyFromString(t, tc.KubeConfig, configData)

	filters := metav1.ListOptions{LabelSelector: ""}
	k8s.WaitUntilNumPodsCreated(t, tc.KubeConfig, filters, 2, 2, 5000)

	deploymentName, _ := k8s.RunKubectlAndGetOutputE(t, tc.KubeConfig, "get", "deployment", "-n", tc.Namespace, "-o", "name")
	require.Equal(t, "deployment.apps/nginx-deployment", deploymentName)

	deployment, err := tt.GetDeploymentE(t, tc.KubeConfig, "nginx-deployment")
	require.NoError(t, err)
	require.Equal(t, deployment.Namespace, tc.Namespace)
	require.Equal(t, deployment.Name, "nginx-deployment")
}

// Tests that we can get an image tag from a deployment
func TestGetDeploymentImageTag(t *testing.T) {

	kubet := th.NewKubeTest(t)
	defer kubet.CleanUp()
	tc := kubet.Config

	configData := fmt.Sprintf(EXAMPLE_DEPLOYMENT_YAML_TEMPLATE, tc.Namespace)
	defer func() {
		if !*th.SkipCleanUp {
			k8s.KubectlDeleteFromString(t, tc.KubeConfig, configData)
		}
	}()
	k8s.KubectlApplyFromString(t, tc.KubeConfig, configData)

	filters := metav1.ListOptions{LabelSelector: ""}
	k8s.WaitUntilNumPodsCreated(t, tc.KubeConfig, filters, 2, 2, 5000)

	deploymentName, _ := k8s.RunKubectlAndGetOutputE(t, tc.KubeConfig, "get", "deployment", "-n", tc.Namespace, "-o", "name")
	require.Equal(t, deploymentName, "deployment.apps/nginx-deployment")

	image := tt.GetDeploymentImageTag(t, tc.KubeConfig, "nginx-deployment")
	require.Equal(t, image, "1.15.7")
}

// Tests that an error is returned if deployment doesnt exist
func TestGetDeploymentEReturnsErrorForNotExistantDeployment(t *testing.T) {

	kubet := th.NewKubeTest(t)
	defer kubet.CleanUp()
	tc := kubet.Config

	deployment, err := tt.GetDeploymentE(t, tc.KubeConfig, "nginx-deployment-doesnt-exist")
	require.Error(t, err)
	require.Equal(t, deployment.Name, "")
}

// Tests that we error if deployment is not in the correct namespace
func TestGetDeploymentEReturnsErrorForDeploymentInWrongNamespace(t *testing.T) {

	kubet := th.NewKubeTest(t)
	defer kubet.CleanUp()
	tc := kubet.Config

	wrongNameSpaceConfig := k8s.NewKubectlOptions("", tc.KubeConfig.ConfigPath, "wrong-namespace")
	deployment, err := tt.GetDeploymentE(t, wrongNameSpaceConfig, "nginx-deployment")
	require.Error(t, err)
	require.Equal(t, deployment.Name, "")
}

// Tests that we can set a label on a namespace
func TestNamespaceLabel(t *testing.T) {

	kubet := th.NewKubeTest(t)
	defer kubet.CleanUp()
	tc := kubet.Config

	tt.NamespaceLabel(t, tc.KubeConfig, tc.Namespace, "test=true")

	namespace := k8s.GetNamespace(t, tc.KubeConfig, tc.Namespace)
	require.Equal(t, namespace.Name, tc.Namespace)
	require.Equal(t, namespace.Labels["test"], "true")
}

// Tests that we can set a label on a namespace
func TestNamespaceLabelE(t *testing.T) {

	kubet := th.NewKubeTest(t)
	defer kubet.CleanUp()
	tc := kubet.Config

	err := tt.NamespaceLabelE(t, tc.KubeConfig, tc.Namespace, "test=true")

	require.NoError(t, err)
	namespace := k8s.GetNamespace(t, tc.KubeConfig, tc.Namespace)
	require.Equal(t, namespace.Name, tc.Namespace)
	require.Equal(t, namespace.Labels["test"], "true")
}

// Tests that we error on failing to provide a key and value for the label
func TestErrorNamespaceLabelE(t *testing.T) {

	kubet := th.NewKubeTest(t)
	defer kubet.CleanUp()
	tc := kubet.Config

	err := tt.NamespaceLabelE(t, tc.KubeConfig, tc.Namespace, "test")

	require.Error(t, err)
}

// Tests that we can set an annotation on a namespace
func TestNamespaceAnnotate(t *testing.T) {

	kubet := th.NewKubeTest(t)
	defer kubet.CleanUp()
	tc := kubet.Config

	tt.NamespaceAnnotate(t, tc.KubeConfig, tc.Namespace, "test/annotation='true'")

	namespace := k8s.GetNamespace(t, tc.KubeConfig, tc.Namespace)
	require.Equal(t, namespace.Name, tc.Namespace)
	require.Equal(t, namespace.Annotations["test/annotation"], "'true'")
}

// Tests that we can set an annotation on a namespace
func TestNamespaceAnnotateE(t *testing.T) {

	kubet := th.NewKubeTest(t)
	defer kubet.CleanUp()
	tc := kubet.Config

	err := tt.NamespaceAnnotateE(t, tc.KubeConfig, tc.Namespace, "test/annotation='true'")

	require.NoError(t, err)
	namespace := k8s.GetNamespace(t, tc.KubeConfig, tc.Namespace)
	require.Equal(t, namespace.Name, tc.Namespace)
	require.Equal(t, namespace.Annotations["test/annotation"], "'true'")

	err = tt.NamespaceAnnotateE(t, tc.KubeConfig, tc.Namespace, "test/annotation2='true'")

	require.NoError(t, err)
	namespace = k8s.GetNamespace(t, tc.KubeConfig, tc.Namespace)
	require.Equal(t, namespace.Name, tc.Namespace)
	require.Equal(t, namespace.Annotations["test/annotation2"], "'true'")
}
