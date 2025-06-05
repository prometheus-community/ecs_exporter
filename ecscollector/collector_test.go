// Copyright 2025 The Prometheus Authors
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
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/prometheus-community/ecs_exporter/ecsmetadata"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// Create a metadata client that will always receive the given fixture API
// responses.
func fixtureClient(taskMetadataPath, taskStatsPath string) (*ecsmetadata.Client, *httptest.Server, error) {
	taskMetadata, err := os.ReadFile(taskMetadataPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read task metadata fixture: %w", err)
	}
	taskStats, err := os.ReadFile(taskStatsPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read task stats fixture: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /task", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		w.Write(taskMetadata)
	})
	mux.HandleFunc("GET /task/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		w.Write(taskStats)
	})

	server := httptest.NewServer(mux)
	return ecsmetadata.NewClient(server.URL), server, nil
}

// Renders metrics from the given collector to the prometheus text exposition
// format.
func renderMetrics(collector prometheus.Collector) ([]byte, error) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	// It seems that the only way to really get full /metrics output is with
	// promhttp. There is testutil.CollectAndFormat but it requires you to
	// specify every metric name you want in the output, which seems to be not
	// worth it compared to this.
	promServer := httptest.NewServer(promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	defer promServer.Close()
	resp, err := http.Get(promServer.URL)
	if err != nil {
		return nil, fmt.Errorf("metrics request failed: %w", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("non-200 metrics response: %v", resp.StatusCode)
	}
	metrics, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read metrics response body: %w", err)
	}
	return metrics, nil
}

var updateSnapshots = flag.Bool("update-snapshots", false, "update snapshot files")

func assertSnapshot(t *testing.T, collector prometheus.Collector, path string) {
	if *updateSnapshots {
		metrics, err := renderMetrics(collector)
		if err != nil {
			t.Fatalf("failed to render new snapshot %s: %v", path, err)
		}
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("failed to create snapshot output directory %s: %v", dir, err)
		} else if err := os.WriteFile(path, metrics, 0666); err != nil {
			t.Fatalf("failed to write snapshot file %s: %v", path, err)
		} else {
			t.Logf("updated snapshot: %s", path)
		}
	}

	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("snapshot file does not exist, set the -update-snapshots flag to update: %v", err)
	} else if err != nil {
		t.Fatalf("failed to open snapshot file: %v", err)
	} else if err := testutil.CollectAndCompare(collector, file); err != nil {
		t.Fatalf("snapshot outdated, set the -update-snapshots flag to update\n%v", err)
	}
}

func TestFargateMetrics(t *testing.T) {
	metadataClient, metadataServer, err := fixtureClient(
		"testdata/fixtures/fargate_task_metadata.json",
		"testdata/fixtures/fargate_task_stats.json",
	)
	if err != nil {
		t.Fatalf("failed to load test fixtures: %v", err)
	}
	defer metadataServer.Close()
	collector := NewCollector(metadataClient, slog.Default())
	assertSnapshot(t, collector, "testdata/snapshots/fargate_metrics.txt")
}

func TestEc2Metrics(t *testing.T) {
	metadataClient, metadataServer, err := fixtureClient(
		"testdata/fixtures/ec2_task_metadata.json",
		"testdata/fixtures/ec2_task_stats.json",
	)
	if err != nil {
		t.Fatalf("failed to load test fixtures: %v", err)
	}
	defer metadataServer.Close()
	collector := NewCollector(metadataClient, slog.Default())
	assertSnapshot(t, collector, "testdata/snapshots/ec2_metrics.txt")
}

func TestApiErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /task", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
	})
	mux.HandleFunc("GET /task/stats", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
	})
	metadataServer := httptest.NewServer(mux)
	defer metadataServer.Close()
	metadataClient := ecsmetadata.NewClient(metadataServer.URL)
	collector := NewCollector(metadataClient, slog.Default())

	_, err := renderMetrics(collector)
	if err == nil || err.Error() != "non-200 metrics response: 500" {
		t.Fatalf("expected 500 error but got err: %v", err)
	}
}
