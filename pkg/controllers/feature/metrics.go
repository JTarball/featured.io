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

package feature

import (
	"github.com/prometheus/client_golang/prometheus"
)

func newGauge(namespace string, subsystem string, name string, help string, labels []string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)
}

func newCounter(namespace string, subsystem string, name string, help string, labels []string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)
}

var (
	configmapTotalCount   = newCounter("featured_operator", "featureflag", "configmaps", "Total number of configmap managed", []string{})
	configmapCreatedCount = newCounter("featured_operator", "featureflag", "configmap_created", "Total number of configmap created", []string{})
	configmapDeletedCount = newCounter("featured_operator", "featureflag", "configmap_deleted", "Total number of configmap deleted", []string{})
)

// RegisterMetrics registers the featurecontroller CRUD metrics.
func RegisterMetrics() {
	prometheus.MustRegister(configmapTotalCount)
	prometheus.MustRegister(configmapCreatedCount)
	prometheus.MustRegister(configmapDeletedCount)
}
