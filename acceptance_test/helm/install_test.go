// +build kind

// kind integration test - kubernetes in docker

// install_test.go - Tests basic helm commands will be successful does NOT check
//  functionality / upgrade with new features /service availability

// The main here to keep these tests as general as possible!

package helm

import (
	"path"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/stretchr/testify/require"

	tt "github.com/featured.io/acceptance_test/testing/terratest"
	th "github.com/featured.io/acceptance_test/testing/test-helper"
)

// TestHelmInstall tests that we can successfully install the chart.
// Availablity / Readiness is NOT checked
func TestHelmInstall(t *testing.T) {
	kubet := th.NewKubeTest(t)
	defer kubet.CleanUp()
	tc := kubet.Config

	rootPath := th.GetGitRootPath(t)
	helmChartPath := "./helm/featured-operator"
	options := &helm.Options{
		KubectlOptions: tc.KubeConfig,
	}
	// Absolute path
	chartPath := path.Join(rootPath, helmChartPath)

	// Sanity check: Check folder exists
	require.DirExists(t, chartPath)

	// Ensure helm dependencies
	tt.EnsureHelmDependencies(t, options, chartPath, false)

	// Release name set to namespace (unique)
	defer helm.Delete(t, options, tc.Namespace, true)
	helm.Install(t, options, chartPath, tc.Namespace)

}

// TestHelmTemplateKubectlApply tests that we can successfully use kubectl to install the chart (if helm is not supported in cluster).
// Availablity / Readiness is NOT checked
func TestHelmTemplateKubectlApply(t *testing.T) {
	kubet := th.NewKubeTest(t)
	defer kubet.CleanUp()
	tc := kubet.Config

	rootPath := th.GetGitRootPath(t)
	helmChartPath := "./helm/featured-operator"
	options := &helm.Options{
		KubectlOptions: tc.KubeConfig,
	}
	// Absolute path
	chartPath := path.Join(rootPath, helmChartPath)

	// Sanity check: Check folder exists
	require.DirExists(t, chartPath)

	// Ensure helm dependencies
	tt.EnsureHelmDependencies(t, options, chartPath, false)

	// Run RenderTemplate to render the template and capture the output.
	output := helm.RenderTemplate(t, options, chartPath, tc.Namespace, []string{})

	// Make sure to delete the resources at the end of the test
	defer k8s.KubectlDeleteFromString(t, tc.KubeConfig, output)

	// Now use kubectl to apply the rendered template
	k8s.KubectlApplyFromString(t, tc.KubeConfig, output)

}

// TestHelmUpgradeInstall tests that we can successfully upgrade the chart when the chart hasn't existed.
// Availablity / Readiness is NOT checked
func TestHelmUpgradeInstall(t *testing.T) {
	kubet := th.NewKubeTest(t)
	defer kubet.CleanUp()
	tc := kubet.Config

	rootPath := th.GetGitRootPath(t)
	helmChartPath := "./helm/featured-operator"
	options := &helm.Options{
		KubectlOptions: tc.KubeConfig,
	}
	// Absolute path
	chartPath := path.Join(rootPath, helmChartPath)

	// Sanity check: Check folder exists
	require.DirExists(t, chartPath)

	// Ensure helm dependencies
	tt.EnsureHelmDependencies(t, options, chartPath, false)

	// Release name set to namespace (unique)
	defer helm.Delete(t, options, tc.Namespace, true)
	tt.UpgradeInstall(t, options, chartPath, tc.Namespace)

}

// TestHelmUpgrade tests that we can successfully upgrade the chart.
// Availablity / Readiness is NOT checked
func TestHelmUpgrade(t *testing.T) {
	kubet := th.NewKubeTest(t)
	defer kubet.CleanUp()
	tc := kubet.Config

	rootPath := th.GetGitRootPath(t)
	helmChartPath := "./helm/featured-operator"
	options := &helm.Options{
		KubectlOptions: tc.KubeConfig,
	}
	// Absolute path
	chartPath := path.Join(rootPath, helmChartPath)

	// Sanity check: Check folder exists
	require.DirExists(t, chartPath)

	// Ensure helm dependencies
	tt.EnsureHelmDependencies(t, options, chartPath, false)

	// Release name set to namespace (unique)
	defer helm.Delete(t, options, tc.Namespace, true)
	helm.Install(t, options, chartPath, tc.Namespace)

	// Change helm version
	options.Version = "21.4.0"
	helm.Upgrade(t, options, chartPath, tc.Namespace)
}

// func TestHelmRollback() tests we can successfully rollback
func TestHelmRollback(t *testing.T) {
	kubet := th.NewKubeTest(t)
	defer kubet.CleanUp()
	tc := kubet.Config

	rootPath := th.GetGitRootPath(t)
	helmChartPath := "./helm/featured-operator"
	options := &helm.Options{
		Version:        "0.0.1",
		KubectlOptions: tc.KubeConfig,
	}
	// Absolute path
	chartPath := path.Join(rootPath, helmChartPath)

	// Sanity check: Check folder exists
	require.DirExists(t, chartPath)

	// Ensure helm dependencies
	tt.EnsureHelmDependencies(t, options, chartPath, false)

	// Release name set to namespace (unique)
	defer helm.DeleteE(t, options, tc.Namespace, true)
	helm.Install(t, options, chartPath, tc.Namespace)

	// Change helm version
	options.Version = "21.4.0"
	helm.Upgrade(t, options, chartPath, tc.Namespace)

	// Rollback
	tt.Rollback(t, options, tc.Namespace, "1")
}
