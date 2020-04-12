// Copyright 2020 Danvir Guram. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	// "math/rand"

	"net/http"
	"os"
	"time"

	// "sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	featurecontroller "github.com/featured.io/pkg/controllers/feature"
	featureclientset "github.com/featured.io/pkg/generated/clientset/versioned"
	featureinformers "github.com/featured.io/pkg/generated/informers/externalversions"
)

// Run starts the mysql-operator controllers. This should never exit.
func Run(flags *CMDFlags) error {

	// Load kubernetes config
	kubeconfig, err := SetKubeConfig(flags)
	if err != nil {
		return err
	}

	// Initialise the operator metrics.
	featurecontroller.RegisterMetrics()
	http.Handle(flags.MetricsPath, promhttp.Handler())
	go http.ListenAndServe(flags.MetricsListenAddr, nil)

	// Set up signals so we handle the first shutdown signal gracefully.
	stopCh := SetupSignalHandler()

	// Set up clients
	kubeClient := kubernetes.NewForConfigOrDie(kubeconfig)
	featureClient := featureclientset.NewForConfigOrDie(kubeconfig)

	// Shared informers (non namespace specific).
	// TODO: Add ResyncPeriod(flags)() so we have a minimum resync period that we can set with a flag
	// Informers trigger events on resource changes, which typically queues
	// a reconciliation of that resource.  The resync period acts as a
	// safety net to protect against dropped events, and the informer will
	// signal a "nop" event to trigger a reconciliation of every resource
	// on this interval.  Small values here have an assortment of problems:
	//  1. It can mask unhandled events.  If an event is not handled then
	//    the associated resources will still be reconciled quickly and
	//    likely not caught by e2e testing.
	//  2. Doesn't scale well.  The default rest QPS limit is 5, so assuming
	//    one call per reconcile a 30s period starts to strain at 150
	//    resources (just with nop reconciles).  With 10h that number
	//     becomes 180,000 resources.
	// 10 hours is the resync period used by sigs.k8s.io/controller-runtime.
	var noResyncPeriodFunc = func() time.Duration { return 10 * time.Minute }
	i := featureinformers.NewFilteredSharedInformerFactory(featureClient, noResyncPeriodFunc(), flags.Namespace, nil)
	k8sI := kubeinformers.NewFilteredSharedInformerFactory(kubeClient, noResyncPeriodFunc(), flags.Namespace, nil)

	featureController := featurecontroller.NewFeatureController(
		kubeClient,
		featureClient,
		k8sI.Core().V1().ConfigMaps(),
		i.Featurecontroller().V1alpha1().FeatureFlags(),
	)

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	k8sI.Start(stopCh)
	i.Start(stopCh)

	if err = featureController.Run(2, stopCh); err != nil {
		log.Fatalf("Error running controller: %s", err.Error())
		os.Exit(1)
	}

	return nil
}
