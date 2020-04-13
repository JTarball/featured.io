package test_helper

//
// Main Helper file for running kubernetes style tests
//

import (
	"flag"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/shell"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tt "github.com/featured.io/acceptance_test/testing/terratest"
)

// BaseNamespace is the prefix for the kubernetes namespace created
const BaseNamespace = "featured-test" // WARNING: Do not change this unless you know what you are doing

// BaseNamespaceLabel is the label used on the namespace - used for identified for nuking old tests
const BaseNamespaceLabel = "terratest/featured.io"

type BaseTestConfig struct {
	TestID         string              // The unique test ID
	Namespace      string              // The K8S namespace you are targetting the test against
	KubeConfigPath string              // The path to the kube config
	KubeConfig     *k8s.KubectlOptions // The kubeconfig for the test
}

// KubeTest is the base BET test structure
type KubeTest struct {
	// The testing type to manage test state etc.
	T *testing.T

	// Config specifies the test configuration need to run
	// a kubernetes test.
	Config BaseTestConfig
}

var onlyOnce sync.Once

//
// Flags
//

// IsDev set if you want to run on local kubernets
var IsDev = flag.Bool("dev", false, "Run tests on local kubernetes e.g. dfd/k3s (Setup up and point to it yourself)")

// go test <file/dir> --skip-cleanup
// Skips deployment stage - very useful for local validation of integration tests
var SkipCleanUp = flag.Bool("skip-cleanup", false, "Skip cleanup of resources")

var nukeOldTestRuns = flag.Bool("nuke-old", false, "Run func to nuke old test runs (namespace, k8s resourcces, helm releases) (normally older than 1 hr / namespace 1 day) (local k8s 1 min ago)")

// go test <file/dir> --namespace <namespace>
// Target a k8s namespace
var targetNamespace = flag.String("namespace", "", "The kubernetes namespace you want to target")

//
// Helper functions
//

// createNamespace setups and creates k8s namespace
// Sets a label that can be used for nuking "%s=<namespace>"
func createNamespace(t *testing.T, tc *BaseTestConfig) {
	//
	// Set up the k8s namespace, including annotations for rancher (correct project)
	//
	_, err := k8s.GetNamespaceE(t, tc.KubeConfig, tc.Namespace)

	if err != nil {
		logger.Log(t, "---- Creating K8S namepace ----", tc.Namespace)
		k8s.CreateNamespace(t, tc.KubeConfig, tc.Namespace)
		// Add label for identification & nuking later
		cmdArgs := append([]string{"label", "namespaces", tc.Namespace}, fmt.Sprintf("%s=%s", BaseNamespaceLabel, tc.Namespace))
		require.NoError(t, k8s.RunKubectlE(t, tc.KubeConfig, cmdArgs...))
	}
}

// GetGitRootPath gathers the root path for a git repository, if an error we fail the test straight away
func GetGitRootPath(t *testing.T) string {
	output, err := GetGitRootPathE(t)
	require.NoError(t, err)
	return output
}

// GetGitRootPathE gathers the root path for a git repository
func GetGitRootPathE(t *testing.T) (string, error) {
	logger.Log(t, "---- Getting GIT Root Path ----")
	rootPath, err := shell.RunCommandAndGetOutputE(t, shell.Command{
		Command:           "git",
		Args:              []string{"rev-parse", "--show-toplevel"},
		WorkingDir:        ".",
		Env:               make(map[string]string),
		OutputMaxLineSize: 1000,
	})

	logger.Log(t, "GIT Root Path: ", rootPath)

	return rootPath, err
}

// NewKubeTest setups up the environment for a kubernetes "Namespaced" test
// This allows has support for local kubernetes with k3s if the correct flag has been set up.
// You must call the cleanup func CleanUp() after
// Example:
//   kubet := th.NewKubeTest(t)
//   defer kubet.CleanUp()
func NewKubeTest(t *testing.T) *KubeTest {

	// Only supports Helm 3
	tt.IsHelm3(t, false, false)

	// Setup a unique test through some test configuration
	// A K8S unique namespace to ensure "Namespaced" resources - https://terratest.gruntwork.io/docs/testing-best-practices/namespacing/)
	// A unique release name that we can refer to later for cleanup ~ `helm delete RELEASE_NAME`
	uniqueTestID := strings.ToLower(random.UniqueId())
	uniqueNamespace := fmt.Sprintf("%s-%s", BaseNamespace, uniqueTestID)

	if *targetNamespace != "" {
		// TODO: Add defensive you only been used for a single test
		uniqueNamespace = *targetNamespace
	}

	kubeConfigPath := ""
	kubeConfig := k8s.NewKubectlOptions("", kubeConfigPath, uniqueNamespace)

	//
	// Special flag to Duke Nuke'em  (inspired by https://github.com/gruntwork-io/cloud-nuke)
	// Cleanup Job of old tests
	//
	if *nukeOldTestRuns {

		onlyOnce.Do(func() {
			logger.Log(t, "---- Nuke'em flag detected ----")
			logger.Log(t, "---- Searching for and cleaning up old test runs ----")

			now := metav1.Now()

			releaseAgo := metav1.Date(
				now.Year(), now.Month(), now.Day(),
				now.Hour()-1, now.Minute(), now.Second(), now.Nanosecond(), now.Location(),
			)
			NamespaceAgo := metav1.Date(
				now.Year(), now.Month(), now.Day()-1,
				now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location(),
			)

			if *IsDev {
				logger.Log(t, "---- On k3s/dfd will nuke anything from now ----")
				releaseAgo = metav1.Date(
					now.Year(), now.Month(), now.Day(),
					now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location(),
				)
				NamespaceAgo = metav1.Date(
					now.Year(), now.Month(), now.Day(),
					now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location(),
				)
			}

			// Find all namespaces with label BaseNamespaceLabel
			cmdArgs := append([]string{"get", "namespaces", "--sort-by=.metadata.creationTimestamp", "-l", BaseNamespaceLabel, "-o=name"})
			namespaces, err := k8s.RunKubectlAndGetOutputE(t, kubeConfig, cmdArgs...)
			require.NoError(t, err)

			for _, namespace := range strings.Fields(namespaces) {
				namespaceFullName := strings.SplitN(namespace, "/", -1)
				namespaceName := namespaceFullName[len(namespaceFullName)-1]
				namespaceV1 := k8s.GetNamespace(t, kubeConfig, namespaceName)

				creationTime := namespaceV1.ObjectMeta.CreationTimestamp

				// Paranoid defensive check (the namespace)
				if !strings.HasPrefix(namespaceName, BaseNamespace) {
					t.Fatalf("DANGER: The namespace you were about to delete was not a featured test namespace... Bailing for safety")
				}

				if creationTime.Before(&releaseAgo) {
					logger.Log(t, fmt.Sprintf("---- Found old namespace '%s' created %s to cleanup ----", namespaceName, namespaceV1.ObjectMeta.CreationTimestamp))

					// Find all helm releases for namespace and delete
					releases, err := helm.RunHelmCommandAndGetOutputE(t, &helm.Options{KubectlOptions: kubeConfig}, "ls", "--namespace", namespaceName, "--short")
					require.NoError(t, err)

					for _, release := range strings.Fields(releases) {
						logger.Log(t, fmt.Sprintf("---- Deleting release '%s' ----", release))
						helm.DeleteE(t, &helm.Options{KubectlOptions: kubeConfig}, release, true) // ignore errors - we don't care if fails
					}

				}

				// Allow helm releases to be deleted first (so dont get conflict when deleting) and use this as secondary cleanup - Nuke namespace
				if creationTime.Before(&NamespaceAgo) {
					logger.Log(t, fmt.Sprintf("---- Deleting namespace '%s' ----", namespaceName))
					k8s.DeleteNamespace(t, kubeConfig, namespaceName)
				}

			}
		})
	}

	// The basic test configuration to run a test on kubernetes
	tc := BaseTestConfig{
		TestID:         uniqueTestID,
		Namespace:      uniqueNamespace,
		KubeConfigPath: kubeConfigPath,
		KubeConfig:     kubeConfig,
	}

	createNamespace(t, &tc)

	return &KubeTest{
		T:      t,
		Config: tc,
	}
}

// CleanUp cleanup the kubernetes resources. Best to call this with a defer.
func (kt *KubeTest) CleanUp() {
	if !*SkipCleanUp {
		defer func() {
			k8s.DeleteNamespace(kt.T, kt.Config.KubeConfig, kt.Config.Namespace)
			namespace := k8s.GetNamespace(kt.T, kt.Config.KubeConfig, kt.Config.Namespace)
			require.Equal(kt.T, namespace.Status.Phase, corev1.NamespaceTerminating)
		}()
	} else {
		logger.Log(kt.T, "---- Skipping  cleanup ----")
		logger.Log(kt.T, "---- Copy this should you want to investigate the namespace: ----")
		logger.Log(kt.T, fmt.Sprintf("    kubectl config set-context --current --namespace %s", kt.Config.Namespace))
	}

}
