// Copyright The Prometheus Authors
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

package ecscollector

import (
	"context"
	"log/slog"

	"github.com/prometheus-community/ecs_exporter/ecsmetadata"
	"github.com/prometheus/client_golang/prometheus"
)

// Runtime owns one embeddable instance of the ECS exporter. Ref:
// https://github.com/prometheus/prometheus-opentelemetry-collector/blob/main/docs/embeddable-exporters.md#collector-package-contract
type Runtime struct {
	cancel    context.CancelFunc
	collector prometheus.Collector
}

// NewRuntime returns a Runtime that queries the ECS metadata server through
// client. Shutdown cancels any metadata requests in progress.
func NewRuntime(client *ecsmetadata.Client, logger *slog.Logger) *Runtime {
	ctx, cancel := context.WithCancel(context.Background())
	return &Runtime{
		cancel:    cancel,
		collector: newCollector(ctx, client, logger),
	}
}

// Collectors returns the Prometheus collectors belonging to this Runtime.
// The caller owns the registry into which they are registered.
func (r *Runtime) Collectors() []prometheus.Collector {
	return []prometheus.Collector{r.collector}
}

// Shutdown cancels metadata requests in progress. It is safe to call more
// than once.
func (r *Runtime) Shutdown(context.Context) error {
	r.cancel()
	return nil
}
