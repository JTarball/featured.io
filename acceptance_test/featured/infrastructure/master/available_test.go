// +build kind

// kind integration test - kubernetes in docker

package helm

import (
	"path"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/require"

	tt "github.com/featured.io/acceptance_test/testing/terratest"
	th "github.com/featured.io/acceptance_test/testing/test-helper"
)

// TestAvailable tests the operator can be deployed and is ready (we take master image)
func TestAvailable(t *testing.T) {
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
	require.DirExists(t, chartPath)

	// Ensure helm dependencies
	tt.EnsureHelmDependencies(t, options, chartPath, false)

	// Release name set to namespace (unique)
	if !*th.SkipCleanUp {
		defer helm.Delete(t, options, tc.Namespace, true)
	}
	tt.UpgradeInstall(t, options, chartPath, tc.Namespace)

}
