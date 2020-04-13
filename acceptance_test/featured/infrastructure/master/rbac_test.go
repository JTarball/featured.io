// +build gke
// +build integration

package test

import (
	"os"
	"path"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/stretchr/testify/require"
	authv1 "k8s.io/api/authorization/v1"

	tt "github.com/featured.io/acceptance_test/testing/terratest"
	th "github.com/featured.io/acceptance_test/testing/test-helper"
)

// TestRBAC checks that the correct permissions are set for a namespaced operator
func TestRBACNamespaced(t *testing.T) {
	kubet := th.NewKubeTest(t)
	defer kubet.CleanUp()
	tc := kubet.Config

	const serviceAccountName = "terratest-rbac-service-account"

	rootPath := th.GetGitRootPath(t)
	helmChartPath := "./helm/featured-operator"
	options := &helm.Options{
		KubectlOptions: tc.KubeConfig,
		SetValues: map[string]string{
			"serviceAccount.name": serviceAccountName,
		},
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

	// Setup the kubectl config and context. Here we choose to create a new one because we will be manipulating the
	// entries to be able to add a new authentication option.
	tmpConfigPath := k8s.CopyHomeKubeConfigToTemp(t)
	if !*th.SkipCleanUp {
		defer os.Remove(tmpConfigPath)
	}
	tmpKubeOptions := k8s.NewKubectlOptions("", tmpConfigPath, tc.Namespace)

	// Retrieve authentication token for the newly created ServiceAccount
	token := k8s.GetServiceAccountAuthToken(t, tmpKubeOptions, serviceAccountName)

	// Now update the configuration to add a new context that can be used to make requests as that service account
	require.NoError(t, k8s.AddConfigContextForServiceAccountE(
		t,
		tmpKubeOptions,
		serviceAccountName, // for this test we will name the context after the ServiceAccount
		serviceAccountName,
		token,
	))
	serviceAccountKubectlOptions := k8s.NewKubectlOptions(serviceAccountName, tmpConfigPath, tc.Namespace)

	configmapCreationAction := authv1.ResourceAttributes{
		Namespace: tc.Namespace,
		Verb:      "create",
		Resource:  "configmaps",
	}
	require.True(t, k8s.CanIDo(t, serviceAccountKubectlOptions, configmapCreationAction))

	configmapUpdateAction := authv1.ResourceAttributes{
		Namespace: tc.Namespace,
		Verb:      "update",
		Resource:  "configmaps",
	}
	require.True(t, k8s.CanIDo(t, serviceAccountKubectlOptions, configmapUpdateAction))

	configmapWatchAction := authv1.ResourceAttributes{
		Namespace: tc.Namespace,
		Verb:      "watch",
		Resource:  "configmaps",
	}
	require.True(t, k8s.CanIDo(t, serviceAccountKubectlOptions, configmapWatchAction))

	configmapGetAction := authv1.ResourceAttributes{
		Namespace: tc.Namespace,
		Verb:      "get",
		Resource:  "configmaps",
	}
	require.True(t, k8s.CanIDo(t, serviceAccountKubectlOptions, configmapGetAction))

	configmapListAction := authv1.ResourceAttributes{
		Namespace: tc.Namespace,
		Verb:      "list",
		Resource:  "configmaps",
	}
	require.True(t, k8s.CanIDo(t, serviceAccountKubectlOptions, configmapListAction))

	featureflagCreateAction := authv1.ResourceAttributes{
		Namespace: tc.Namespace,
		Verb:      "create",
		Resource:  "featureflags",
	}
	require.True(t, k8s.CanIDo(t, serviceAccountKubectlOptions, featureflagCreateAction))

	featureflagGetAction := authv1.ResourceAttributes{
		Namespace: tc.Namespace,
		Verb:      "get",
		Resource:  "featureflags",
	}
	require.True(t, k8s.CanIDo(t, serviceAccountKubectlOptions, featureflagGetAction))

	featureflagListAction := authv1.ResourceAttributes{
		Namespace: tc.Namespace,
		Verb:      "list",
		Resource:  "featureflags",
	}
	require.True(t, k8s.CanIDo(t, serviceAccountKubectlOptions, featureflagListAction))
}
