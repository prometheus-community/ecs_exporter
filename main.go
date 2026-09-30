// Copyright 2021 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	promcollectors "github.com/prometheus/client_golang/prometheus/collectors"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/exporter-toolkit/bootstrap"

	"github.com/prometheus-community/ecs_exporter/ecscollector"
	"github.com/prometheus-community/ecs_exporter/ecsmetadata"
)

const exporter = "ecs_exporter"

func main() {
	runner := bootstrap.New(bootstrap.Config{
		Name:                  exporter,
		Description:           "Prometheus Exporter for ECS",
		DefaultAddress:        ":9779",
		MetricsHandlerFactory: newMetricsHandler,
	})
	if err := runner.Run(); err != nil {
		if runner.Logger == nil {
			kingpin.CommandLine.Errorf("%s, try --help", err)
		} else {
			runner.Logger.Error("Error running exporter", "err", err)
		}
		os.Exit(1)
	}
}

func newMetricsHandler(b *bootstrap.Bootstrap) (http.Handler, error) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(versioncollector.NewCollector(exporter))

	client, err := ecsmetadata.NewClientFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("creating ECS metadata client: %w", err)
	}
	registry.MustRegister(ecscollector.NewCollector(client, b.Logger))

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		MaxRequestsInFlight: b.MaxRequests,
	})
	if !b.DisableExporterMetrics {
		registry.MustRegister(
			promcollectors.NewProcessCollector(promcollectors.ProcessCollectorOpts{}),
			promcollectors.NewGoCollector(),
		)
		handler = promhttp.InstrumentMetricHandler(registry, handler)
	}

	b.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})

	return handler, nil
}
