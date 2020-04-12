// +build !windows
package app

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var onlyOneSignalHandler = make(chan struct{})
var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

// kubernetes/client-go uses rate-limiting to prevent spamming the apiserver.
const (
	clientQPS   = 100
	clientBURST = 100
)

// SetKubeConfig sets the kubeconfig based on the flags.
// If run in local development we are outside the cluster
func SetKubeConfig(flags *CMDFlags) (*rest.Config, error) {
	var cfg *rest.Config

	// If devel mode then use configuration flag path.
	if flags.DevMode {
		config, err := clientcmd.BuildConfigFromFlags("", flags.KubeConfig)
		if err != nil {
			log.Errorf("could not load configuration: %s", err)
			return nil, err
		}
		cfg = config
	} else {
		config, err := rest.InClusterConfig()
		if err != nil {
			log.Errorf("error loading kubernetes configuration inside cluster: %s", err)
			return nil, err
		}
		cfg = config
	}

	cfg.QPS = clientQPS
	cfg.Burst = clientBURST

	return cfg, nil
}

// SetupSignalHandler registered for SIGTERM and SIGINT. A stop channel is returned
// which is closed on one of these signals. If a second signal is caught, the program
// is terminated with exit code 1.
func SetupSignalHandler() (stopCh <-chan struct{}) {
	close(onlyOneSignalHandler) // panics when called twice

	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, shutdownSignals...)
	go func() {
		<-c
		close(stop)
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	return stop
}

// ResyncPeriod computes the time interval a shared informer waits before
// resyncing with the api server.
func ResyncPeriod(flags *CMDFlags) func() time.Duration {
	return func() time.Duration {
		factor := rand.Float64() + 1
		return time.Duration(float64(flags.MinResyncPeriod.Nanoseconds()) * factor)
	}
}
