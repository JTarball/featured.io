package helm

import (
	"path"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/require"

	tt "github.com/featured.io/acceptance_test/testing/terratest"
	th "github.com/featured.io/acceptance_test/testing/test-helper"
)

// TestHelmLinting runs `helm lint` on a chart and its dependencies
func TestHelmLinting(t *testing.T) {
	t.Parallel() // Watch this - but hopefully should be ok
	tt.IsHelm3(t, true, true)

	rootPath := th.GetGitRootPath(t)
	helmChartPath := "./helm/featured-operator"
	options := &helm.Options{}

	// Absolute path
	// Sanity check: Check folder exists
	chartPath := path.Join(rootPath, helmChartPath)
	require.DirExists(t, chartPath)

	// Ensure helm dependencies
	tt.EnsureHelmDependencies(t, options, chartPath, false)

	tt.EnsureHelmLinting(t, options, chartPath, false)
}
