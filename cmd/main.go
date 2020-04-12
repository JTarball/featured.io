package main

import (
	"os"

	"github.com/featured.io/cmd/app"
	log "github.com/sirupsen/logrus"
)

// Run execs the program.

// Main is the  main runner.
// type Main struct {
// 	flags *CMDFlags
// 	//k8sConfig rest.Config
// 	stopC chan struct{}
// }

// // New returns a Main object.
// func NewMain() Main {
// 	// Init flags.
// 	flags := &CMDFlags{}
// 	flags.Init()

// 	return Main{
// 		flags: flags,
// 	}
// }

// func (m *Main) Run() error {
// 	// // Create signal channels.
// 	// m.stopC = make(chan struct{})
// 	// errC := make(chan error)

// 	// Set debug logging.
// 	if m.flags.DebugMode {
// 		log.SetLevel(log.DebugLevel)
// 		log.Debug("---- debug mode activated ----")
// 	}

// 	var kubeconfig *rest.Config

// 	if flags.DevMode {
// 		config, err := clientcmd.BuildConfigFromFlags("", flags.KubeConfig)
// 		if err != nil {
// 			return nil, fmt.Errorf("could not load configuration: %s", err)
// 		}
// 		kubeconfig = config
// 	} else {
// 		config, err := rest.InClusterConfig()
// 		if err != nil {
// 			return nil, fmt.Errorf("error loading kubernetes configuration inside cluster, check app is running outside kubernetes cluster or run in development mode: %s", err)
// 		}
// 		kubeconfig = config
// 	}

// // Await signals.
// sigC := m.createSignalCapturer()
// var finalErr error
// select {
// case <-sigC:
// 	m.logger.Infof("Signal captured, exiting...")
// case err := <-errC:
// 	m.logger.Errorf("Error received: %s, exiting...", err)
// 	finalErr = err
// }

// m.stop(m.stopC)
// return finalErr
// 	return nil
// }

func main() {
	// Initialise flags
	flags := &app.CMDFlags{}
	flags.Init()

	// Log as JSON instead of the default ASCII formatter.
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	// Only log the warning severity or above.
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	level, err := log.ParseLevel(flags.LogLevel)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
	log.SetLevel(level)

	log.WithFields(log.Fields{"log.SetLevel": log.GetLevel()}).Info("---- Starting the featured.io operator ----")

	if err = app.Run(flags); err != nil {
		log.Info(os.Stderr, "error executing: %v", err)
		os.Exit(1)
	}
}
