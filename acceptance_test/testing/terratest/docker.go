package terratest

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/shell"
	"github.com/stretchr/testify/require"
)


// Push runs the 'docker push' command at the given path with the given options and fails the test if there are any errors.
func Push(t *testing.T, tag string) {
	require.NoError(t, PushE(t, tag))
}

// PushE runs the 'docker build' command at the given path with the given options and returns any errors.
func PushE(t *testing.T, tag string) error {
	logger.Log(t, "Running 'docker push' for ", tag)

	cmd := shell.Command{
		Command: "docker",
		Args:    []string{"push", tag},
	}

	_, buildErr := shell.RunCommandAndGetOutputE(t, cmd)
	return buildErr
}
