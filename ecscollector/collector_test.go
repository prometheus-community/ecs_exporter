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
	client, server := fixtureClientFromResponses(taskMetadata, taskStats)
	return client, server, nil
}

func fixtureClientFromResponses(taskMetadata, taskStats []byte) (*ecsmetadata.Client, *httptest.Server) {
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
	return ecsmetadata.NewClient(server.URL), server
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

func TestNormalizedMemoryStat(t *testing.T) {
	tests := []struct {
		name     string
		stats    map[string]uint64
		cgroupV2 bool
		want     uint64
		wantOK   bool
	}{
		{name: "v1 hierarchy", stats: map[string]uint64{"rss": 10, "total_rss": 20}, want: 20, wantOK: true},
		{name: "v1 local fallback", stats: map[string]uint64{"rss": 10}, want: 10, wantOK: true},
		{name: "v2 alias", stats: map[string]uint64{"anon": 30}, cgroupV2: true, want: 30, wantOK: true},
		{name: "missing", stats: map[string]uint64{}, wantOK: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := normalizedMemoryStat(test.stats, test.cgroupV2, "rss", "anon")
			if got != test.want || ok != test.wantOK {
				t.Fatalf("normalizedMemoryStat() = (%d, %t), want (%d, %t)", got, ok, test.want, test.wantOK)
			}
		})
	}
}

func TestConfiguredMemoryLimitMib(t *testing.T) {
	ptr := func(value int64) *int64 { return &value }
	tests := []struct {
		name           string
		containerLimit *int64
		taskLimit      *int64
		want           int64
		wantOK         bool
	}{
		{name: "container limit", containerLimit: ptr(128), taskLimit: ptr(512), want: 128, wantOK: true},
		{name: "zero container falls back to task", containerLimit: ptr(0), taskLimit: ptr(512), want: 512, wantOK: true},
		{name: "missing container falls back to task", taskLimit: ptr(512), want: 512, wantOK: true},
		{name: "no configured limit", wantOK: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := configuredMemoryLimitMib(test.containerLimit, test.taskLimit)
			if got != test.want || ok != test.wantOK {
				t.Fatalf("configuredMemoryLimitMib() = (%d, %t), want (%d, %t)", got, ok, test.want, test.wantOK)
			}
		})
	}
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
