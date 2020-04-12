package terratest

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gruntwork-io/gruntwork-cli/collections"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/gruntwork-io/terratest/modules/files"
	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type ChartConfig struct {
	APIVersion  string `yaml:"apiVersion"`
	APPVersion  string `yaml:"appVersion"`
	Description string `yaml:"description"`
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
}

// fields need to be public to unmarshal
type ChartRequirements struct {
	Dependencies []map[string]string `yaml:"dependencies"`
}

// formatSetValuesAsArgs formats the given values as command line args for helm using the given flag (e.g flags of
// the format "--set"/"--set-string" resulting in args like --set/set-string key=value...)
func formatSetValuesAsArgs(setValues map[string]string, flag string) []string {
	args := []string{}

	// To make it easier to test, go through the keys in sorted order
	keys := collections.Keys(setValues)
	for _, key := range keys {
		value := setValues[key]
		argValue := fmt.Sprintf("%s=%s", key, value)
		args = append(args, flag, argValue)
	}

	return args
}

// formatValuesFilesAsArgs formats the given list of values file paths as command line args for helm (e.g of the format
// -f path). This will fail the test if one of the paths do not exist or the absolute path can not be determined.
func formatValuesFilesAsArgs(t *testing.T, valuesFiles []string) []string {
	args, err := formatValuesFilesAsArgsE(t, valuesFiles)
	require.NoError(t, err)
	return args
}

// formatValuesFilesAsArgsE formats the given list of values file paths as command line args for helm (e.g of the format
// -f path). This will error if the file does not exist.
func formatValuesFilesAsArgsE(t *testing.T, valuesFiles []string) ([]string, error) {
	args := []string{}

	for _, valuesFilePath := range valuesFiles {
		// Pass through filepath.Abs to clean the path, and then make sure this file exists
		absValuesFilePath, err := filepath.Abs(valuesFilePath)
		if err != nil {
			return args, errors.WithStackTrace(err)
		}
		if !files.FileExists(absValuesFilePath) {
			return args, errors.WithStackTrace(helm.ValuesFileNotFoundError{valuesFilePath})
		}
		args = append(args, "-f", absValuesFilePath)
	}

	return args, nil
}

// formatSetFilesAsArgs formats the given list of keys and file paths as command line args for helm to set from file
// (e.g of the format --set-file key=path). This will fail the test if one of the paths do not exist or the absolute
// path can not be determined.
func formatSetFilesAsArgs(t *testing.T, setFiles map[string]string) []string {
	args, err := formatSetFilesAsArgsE(t, setFiles)
	require.NoError(t, err)
	return args
}

// formatSetFilesAsArgsE formats the given list of keys and file paths as command line args for helm to set from file
// (e.g of the format --set-file key=path)
func formatSetFilesAsArgsE(t *testing.T, setFiles map[string]string) ([]string, error) {
	args := []string{}

	// To make it easier to test, go through the keys in sorted order
	keys := collections.Keys(setFiles)
	for _, key := range keys {
		setFilePath := setFiles[key]
		// Pass through filepath.Abs to clean the path, and then make sure this file exists
		absSetFilePath, err := filepath.Abs(setFilePath)
		if err != nil {
			return args, errors.WithStackTrace(err)
		}
		if !files.FileExists(absSetFilePath) {
			return args, errors.WithStackTrace(helm.SetFileNotFoundError{setFilePath})
		}
		argValue := fmt.Sprintf("%s=%s", key, absSetFilePath)
		args = append(args, "--set-file", argValue)
	}

	return args, nil
}

// getNamespaceArgs returns the args to append for the namespace, if set in the helm Options struct.
func getNamespaceArgs(options *helm.Options) []string {
	if options.KubectlOptions != nil && options.KubectlOptions.Namespace != "" {
		return []string{"--namespace", options.KubectlOptions.Namespace}
	}
	return []string{}
}

// getValuesArgsE computes the args to pass in for setting values
func getValuesArgsE(t *testing.T, options *helm.Options, args ...string) ([]string, error) {
	args = append(args, formatSetValuesAsArgs(options.SetValues, "--set")...)
	args = append(args, formatSetValuesAsArgs(options.SetStrValues, "--set-string")...)

	valuesFilesArgs, err := formatValuesFilesAsArgsE(t, options.ValuesFiles)
	if err != nil {
		return args, errors.WithStackTrace(err)
	}
	args = append(args, valuesFilesArgs...)

	setFilesArgs, err := formatSetFilesAsArgsE(t, options.SetFiles)
	if err != nil {
		return args, errors.WithStackTrace(err)
	}
	args = append(args, setFilesArgs...)
	return args, nil
}

// HelmDep will build or update the helm dependencies for the selected helm chart. This will fail
// the test if there is an error.
func HelmDep(t *testing.T, options *helm.Options, chartDir string) {
	require.NoError(t, HelmDepE(t, options, chartDir))
}

// HelmDep will build or update the helm dependencies for the selected helm chart.
func HelmDepE(t *testing.T, options *helm.Options, chartDir string) error {
	var buildOrUpdate string
	if files.FileExists(chartDir + "/requirements.lock") {
		buildOrUpdate = "build"
	} else {
		buildOrUpdate = "update"
	}

	_, err := helm.RunHelmCommandAndGetOutputE(t, options, "dependency", buildOrUpdate, chartDir)
	return err
}

// EnsureHelmDependencies() builds dependencies from the requirements.yaml file.
func EnsureHelmDependencies(t *testing.T, chartDir string, skipIfChartsFolderExists bool) {
	if !files.FileExists(chartDir) {
		require.FailNow(t, "Chart not found: "+chartDir)
	}

	if !files.FileExists(chartDir + "/requirements.yaml") {
		logger.Log(t, "Skip chart dependencies - no requirements.yaml")
		return
	}

	if files.FileExists(chartDir+"/charts") && skipIfChartsFolderExists {
		logger.Log(t, "Skip chart dependencies - charts folder already present")
		return
	}

	viper.SetConfigName("requirements")
	viper.AddConfigPath(chartDir)
	var requirements ChartRequirements

	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("Error reading requirements file, %s", err)
	}

	err := viper.Unmarshal(&requirements)
	if err != nil {
		t.Fatalf("Unable to decode into struct, %v", err)
	}

	// Find any file dependencies in the subcharts so we can ensure
	// they are up to date
	for index, _ := range requirements.Dependencies {
		if value, ok := requirements.Dependencies[index]["repository"]; ok {
			if strings.HasPrefix(value, "file:") {
				subChartDir := strings.Trim(value, "file://")
				HelmDep(t, &helm.Options{}, chartDir+"/"+subChartDir)
			}
		}
	}

	HelmDep(t, &helm.Options{}, chartDir)
}

// UpgradeInstall will upgrade the release and chart will be deployed with the lastest configuration.This will fail
// the test if there is an error.
func UpgradeInstall(t *testing.T, options *helm.Options, chart string, releaseName string) {
	require.NoError(t, UpgradeInstallE(t, options, chart, releaseName))
}

// UpgradeE will upgrade the release and chart will be deployed with the lastest configuration.
func UpgradeInstallE(t *testing.T, options *helm.Options, chart string, releaseName string) error {
	// If the chart refers to a path, convert to absolute path. Otherwise, pass straight through as it may be a remote
	// chart.
	if files.FileExists(chart) {
		absChartDir, err := filepath.Abs(chart)
		if err != nil {
			return errors.WithStackTrace(err)
		}
		chart = absChartDir
	}

	var err error
	args := []string{"--install", "--force"}
	args = append(args, getNamespaceArgs(options)...)
	args, err = getValuesArgsE(t, options, args...)
	if err != nil {
		return err
	}

	args = append(args, releaseName, chart)
	_, err = helm.RunHelmCommandAndGetOutputE(t, options, "upgrade", args...)
	return err
}

// IsHelm3 will check whether the helm version is 3 failing the test if not
func IsHelm3(t *testing.T, onlySkip bool) {
	t.Helper()

	version, err := helm.RunHelmCommandAndGetOutputE(
		t, &helm.Options{}, "version", "--short", "--client",
	)
	logger.Logf(t, "---- The helm version is %s ----", version)
	require.NoError(t, err)
	if !strings.HasPrefix(version, "v3") && onlySkip {
		t.Skip("---- Skipping Test (The helm version has to be v3+) ----")
	}

	require.Equal(t, true, strings.HasPrefix(version, "v3"), "This test only supports Helm 3")
}
