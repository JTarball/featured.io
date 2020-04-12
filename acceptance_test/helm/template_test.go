package helm

import (
	"path"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/require"

	tt "github.com/featured.io/acceptance_test/testing/terratest"
	th "github.com/featured.io/acceptance_test/testing/test-helper"
)

// This file contains examples of how to use terratest to test helm chart template logic by rendering the templates
// using `helm template`, and then reading in the rendered templates.

// Run helm template with no value changes and check for an error
// A basic test for template errors
func TestHelmTemplate(t *testing.T) {
	t.Parallel() // Watch this - but hopefully should be ok

	rootPath := th.GetGitRootPath(t)
	helmChartPath := "./helm/featured-operator"
	releaseName := "template-test"
	options := &helm.Options{}

	// Absolute path
	chartPath := path.Join(rootPath, helmChartPath)

	// Sanity check: Check folder exists
	require.DirExists(t, chartPath)

	// Ensure helm dependencies
	logger.Log(t, "---- Adding Helm Repositories ----")
	helm.RunHelmCommandAndGetOutputE(t, options, "repo", "add", "ambassador", "https://getambassador.io")
	tt.EnsureHelmDependencies(t, chartPath, false)

	// Run helm template with no values changes - We should expect no errors
	_, err := helm.RenderTemplateE(t, options, chartPath, releaseName, []string{})
	require.NoError(t, err)
}

// FUTURE: We could add check for expected values perhaps?
// See here: https://github.com/gruntwork-io/terratest/blob/master/test/helm_basic_example_template_test.go
// TODO: We are becoming more mature with our services can we start check that each service is configured correctly?
