package main

import (
	"os"

	"github.com/featured.io/cmd/app"
	log "github.com/sirupsen/logrus"
)

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
		log.Errorf("error executing main: %v", err)
		os.Exit(1)
	}
}
