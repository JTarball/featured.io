// +build kind

package terratest_test

import (
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/files"
	"github.com/gruntwork-io/terratest/modules/helm"
	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tt "github.com/featured.io/acceptance_test/testing/terratest"
	th "github.com/featured.io/acceptance_test/testing/test-helper"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// TestUmbrellaDependencies tests that all dependencies are ensured for a umbrella chart
func TestUmbrellaDependencies(t *testing.T) {
	tt.IsHelm3(t, false, false)

	// We modify a subchart & the version requested in
	// the umbrella requirements to check we include
	// the latest changes when build/update dependencies

	// Bump helm subchart
	x := viper.New()
	x.SetConfigName("Chart")
	x.AddConfigPath("./testdata/subcharts/subchartA")

	if err := x.ReadInConfig(); err != nil {
		t.Fatalf("Error reading charts file, %s", err)
	}

	minor := random.Random(1, 100)
	version := fmt.Sprintf("0.1.%d", minor)

	x.Set("version", version)

	if err := x.WriteConfig(); err != nil {
		t.Fatalf("Failed to write, %v", err)
	}

	// Bump helm umbrella dependencies
	y := viper.New()
	y.SetConfigName("requirements")
	y.AddConfigPath("./testdata/umbrella")
	var requirements tt.ChartRequirements

	if err := y.ReadInConfig(); err != nil {
		t.Fatalf("Error reading charts file, %s", err)
	}

	err := y.Unmarshal(&requirements)
	if err != nil {
		t.Fatalf("Unable to decode into struct, %v", err)
	}

	for index, _ := range requirements.Dependencies {
		if value, ok := requirements.Dependencies[index]["name"]; ok {
			if value == "subchartA" {
				requirements.Dependencies[index]["version"] = version
			}
		}
	}

	y.Set("dependencies", requirements.Dependencies)

	if err := y.WriteConfig(); err != nil {
		t.Fatalf("Failed to write, %v", err)
	}

	// rm lock to ensure dependencies in sync
	os.Remove("./testdata/umbrella/requirements.lock")

	// Run actual test against EnsureHelmDependencies
	tt.EnsureHelmDependencies(t, "./testdata/umbrella", false)

	// Check charts folder is created and includes changes from subchart
	require.Equal(t, files.FileExists("./testdata/umbrella/charts"), true)
	require.Equal(t, files.FileExists(fmt.Sprintf("./testdata/umbrella/charts/subchartA-%s.tgz", version)), true)
	require.Equal(t, files.FileExists("./testdata/umbrella/charts/subchartB-0.1.0.tgz"), true)
}

// Test that we can install & upgrade a remote chart (e.g stable/chartmuseum) using UpgradeInstall command
func TestRemoteChartInstallAndUpgrade(t *testing.T) {

	kubet := th.NewKubeTest(t)
	defer kubet.CleanUp()
	tc := kubet.Config

	// ---------
	helmChart := "stable/chartmuseum"

	// Override service type to node port
	options := &helm.Options{
		KubectlOptions: tc.KubeConfig,
		SetValues: map[string]string{
			"service.type": "NodePort",
		},
	}

	// Generate a unique release name so we can defer the delete before installing
	releaseName := fmt.Sprintf(
		"chartmuseum-%s",
		strings.ToLower(random.UniqueId()),
	)
	defer helm.Delete(t, options, releaseName, true)
	tt.UpgradeInstall(t, options, helmChart, releaseName)

	// Get pod and wait for it to be available
	// To get the pod, we need to filter it using the labels that the helm chart creates
	filters := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=chartmuseum,release=%s", releaseName),
	}
	k8s.WaitUntilNumPodsCreated(t, tc.KubeConfig, filters, 1, 30, 5*time.Second)
	pods := k8s.ListPods(t, tc.KubeConfig, filters)
	for _, pod := range pods {
		k8s.WaitUntilPodAvailable(t, tc.KubeConfig, pod.Name, 30, 5*time.Second)
	}
	// Setting replica count to 2 to check the upgrade functionality.
	// After successful upgrade , the count of pods should be equal to 2.
	options.SetValues = map[string]string{
		"replicaCount": "2",
		"service.type": "NodePort",
	}
	tt.UpgradeInstall(t, options, helmChart, releaseName)

	// Get pod and wait for it to be avaialable
	// To get the pod, we need to filter it using the labels that the helm chart creates
	filters = metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=chartmuseum,release=%s", releaseName),
	}
	k8s.WaitUntilNumPodsCreated(t, tc.KubeConfig, filters, 2, 30, 5*time.Second)
	pods = k8s.ListPods(t, tc.KubeConfig, filters)
	for _, pod := range pods {
		k8s.WaitUntilPodAvailable(t, tc.KubeConfig, pod.Name, 30, 5*time.Second)
	}

	// Verify number of pods are equal to 2
	require.Equal(t, len(pods), 2, "The pods count should be equal to 2 post upgrade")

	// Verify service is accessible. Wait for it to become available and then hit the endpoint.
	// Service name is RELEASE_NAME-CHART_NAME
	serviceName := fmt.Sprintf("%s-chartmuseum", releaseName)
	k8s.WaitUntilServiceAvailable(t, tc.KubeConfig, serviceName, 10, 5*time.Second)

	// This method fails:  UCDP has a problem with DNS so we just need the port
	//service := k8s.GetService(t, tc.KubeConfig, serviceName)
	//endpoint := k8s.GetServiceEndpoint(t, tc.KubeConfig, service, 8080)
	//
	// Or this and using 0.0.0.0
	//hostPort := strings.Split(endpoint, ":")

	tunnel := k8s.NewTunnel(tc.KubeConfig, k8s.ResourceTypeService, serviceName, 0, 8080)
	defer tunnel.Close()
	tunnel.ForwardPort(t)

	// Setup a TLS configuration to submit with the helper, a blank struct is acceptable
	tlsConfig := tls.Config{}

	http_helper.HttpGetWithRetryWithCustomValidation(
		t,
		fmt.Sprintf("http://%s", tunnel.Endpoint()),
		&tlsConfig,
		30,
		10*time.Second,
		func(statusCode int, body string) bool {
			return statusCode == 200
		},
	)
}
