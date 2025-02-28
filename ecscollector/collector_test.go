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
	"bytes"
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

// Renders ecs_exporter metrics from the given metadata client to the prometheus
// text exposition format.
func renderMetrics(client *ecsmetadata.Client) ([]byte, error) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(NewCollector(client, slog.Default()))

	// It seems that the only way to really get full /metrics output is with
	// promhttp.
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

func assertSnapshot(t *testing.T, path string, actual []byte) {
	snapshot, _ := os.ReadFile(path)
	if !bytes.Equal(actual, snapshot) {
		if *updateSnapshots {
			os.MkdirAll(filepath.Dir(path), 0750)
			os.WriteFile(path, actual, 0666)
			t.Logf("updated snapshot: %s", path)
		} else {
			t.Fatalf("snapshot outdated, set the -update-snapshots flag to update: %s", path)
		}
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
	metrics, err := renderMetrics(metadataClient)
	if err != nil {
		t.Fatalf("failed to render metrics: %v", err)
	}
	assertSnapshot(t, "testdata/snapshots/fargate_metrics.txt", metrics)
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
	metrics, err := renderMetrics(metadataClient)
	if err != nil {
		t.Fatalf("failed to render metrics: %v", err)
	}
	assertSnapshot(t, "testdata/snapshots/ec2_metrics.txt", metrics)
}
