// +build kind

// kind integration test - kubernetes in docker

package helm

import (
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/featured.io/acceptance_test/testing/terratest"
	tt "github.com/featured.io/acceptance_test/testing/terratest"
	th "github.com/featured.io/acceptance_test/testing/test-helper"
)

// TestReady tests the operator can be deployed and is ready (we take master image)
func TestReady(t *testing.T) {
	kubet := th.NewKubeTest(t)
	defer kubet.CleanUp()
	tc := kubet.Config

	rootPath := th.GetGitRootPath(t)
	helmChartPath := "./helm/featured-operator"
	options := &helm.Options{
		KubectlOptions: tc.KubeConfig,
		SetValues: map[string]string{
			"appVersion": tc.Namespace,
		},
	}

	// Absolute path
	chartPath := path.Join(rootPath, helmChartPath)
	require.DirExists(t, chartPath)

	// Build
	kubet.Build(chartPath) // Builds docker image with tag of Namespace

	// Ensure helm dependencies
	tt.EnsureHelmDependencies(t, options, chartPath, false)

	// Release name set to namespace (unique)
	if !*th.SkipCleanUp {
		defer helm.Delete(t, options, tc.Namespace, true)
	}
	tt.UpgradeInstall(t, options, chartPath, tc.Namespace)

	filter := metav1.ListOptions{LabelSelector: "app.kubernetes.io/name=featured-operator"}
	k8s.WaitUntilNumPodsCreated(t, tc.KubeConfig, filter, 1, 30, 1*time.Second)

	for _, pod := range k8s.ListPods(t, tc.KubeConfig, filter) {
		logger.Log(t, "Found pod: ", pod.Name)
		k8s.WaitUntilPodAvailable(t, tc.KubeConfig, pod.Name, 30, time.Duration(2)*time.Second)
	}

	fmt.Println("---- Waiting for operator to become ready ----")
	for _, pod := range k8s.ListPods(t, tc.KubeConfig, filter) {
		terratest.WaitUntilPodReady(t, tc.KubeConfig, pod.Name, 60, time.Duration(10)*time.Second)
	}

}
