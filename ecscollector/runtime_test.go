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
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus-community/ecs_exporter/ecsmetadata"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRuntimeMetrics(t *testing.T) {
	metadataClient, metadataServer, err := fixtureClient(
		"testdata/fixtures/fargate_task_metadata.json",
		"testdata/fixtures/fargate_task_stats.json",
	)
	if err != nil {
		t.Fatalf("failed to load test fixtures: %v", err)
	}
	defer metadataServer.Close()

	runtime := NewRuntime(metadataClient, slog.Default())
	defer runtime.Shutdown(context.Background())
	assertSnapshot(t, runtime.Collectors()[0], "testdata/snapshots/fargate_metrics.txt")
}

func TestRuntimeInstancesAreIsolated(t *testing.T) {
	runtimeA, registryA, serverA := newFixtureRuntime(t, "cluster-a")
	defer serverA.Close()
	runtimeB, registryB, serverB := newFixtureRuntime(t, "cluster-b")
	defer serverB.Close()
	defer runtimeB.Shutdown(context.Background())

	collectorsA := runtimeA.Collectors()
	if len(collectorsA) != 1 || collectorsA[0] != runtimeA.Collectors()[0] {
		t.Fatal("Collectors did not return the Runtime's stable collector instance")
	}

	results := make(chan error, 2)
	for _, registry := range []*prometheus.Registry{registryA, registryB} {
		go func() {
			_, err := registry.Gather()
			results <- err
		}()
	}

	for range 2 {
		if err := <-results; err != nil {
			t.Fatalf("failed to gather metrics: %v", err)
		}
	}
	assertClusterMetric(t, registryA, "cluster-a")
	assertClusterMetric(t, registryB, "cluster-b")

	// We test isolation just through shutdown.
	if err := runtimeA.Shutdown(context.Background()); err != nil {
		t.Fatalf("failed to shut down runtime A: %v", err)
	}
	assertClusterMetric(t, registryB, "cluster-b")
}

func TestRuntimeShutdownCancelsCollection(t *testing.T) {
	for _, blockedEndpoint := range []string{"/task", "/task/stats"} {
		t.Run(blockedEndpoint, func(t *testing.T) {
			started := make(chan struct{})
			canceled := make(chan struct{})
			mux := http.NewServeMux()
			mux.HandleFunc("GET /task", func(w http.ResponseWriter, r *http.Request) {
				if blockedEndpoint == "/task" {
					close(started)
					<-r.Context().Done()
					close(canceled)
					return
				}
				fmt.Fprint(w, taskMetadataResponse("cluster"))
			})
			mux.HandleFunc("GET /task/stats", func(_ http.ResponseWriter, r *http.Request) {
				close(started)
				<-r.Context().Done()
				close(canceled)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			runtime := NewRuntime(ecsmetadata.NewClient(server.URL), slog.Default())
			registry := prometheus.NewRegistry()
			registry.MustRegister(runtime.Collectors()...)
			gathered := make(chan error, 1)
			go func() {
				_, err := registry.Gather()
				gathered <- err
			}()

			waitForSignal(t, started, "metadata request to start")
			shutdownCtx, cancel := context.WithCancel(context.Background())
			cancel()
			if err := runtime.Shutdown(shutdownCtx); err != nil {
				t.Fatalf("failed to shut down runtime: %v", err)
			}
			if err := runtime.Shutdown(context.Background()); err != nil {
				t.Fatalf("failed to shut down runtime again: %v", err)
			}
			waitForSignal(t, canceled, "metadata request to be canceled")
			select {
			case err := <-gathered:
				if err == nil {
					t.Fatal("gather succeeded after its metadata request was canceled")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("gather did not return after shutdown")
			}
		})
	}
}

func newFixtureRuntime(t *testing.T, cluster string) (*Runtime, *prometheus.Registry, *httptest.Server) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /task", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, taskMetadataResponse(cluster))
	})
	mux.HandleFunc("GET /task/stats", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{}`)
	})
	server := httptest.NewServer(mux)
	runtime := NewRuntime(ecsmetadata.NewClient(server.URL), slog.Default())
	registry := prometheus.NewRegistry()
	registry.MustRegister(runtime.Collectors()...)
	return runtime, registry, server
}

func taskMetadataResponse(cluster string) string {
	return fmt.Sprintf(`{
		"Cluster": %q,
		"TaskARN": "task-arn",
		"Family": "family",
		"Revision": "1",
		"DesiredStatus": "RUNNING",
		"KnownStatus": "RUNNING",
		"AvailabilityZone": "zone",
		"LaunchType": "FARGATE",
		"Containers": []
	}`, cluster)
}

func assertClusterMetric(t *testing.T, registry *prometheus.Registry, cluster string) {
	t.Helper()
	expected := fmt.Sprintf(`# HELP ecs_task_metadata_info ECS task metadata, sourced from the task metadata endpoint version 4.
# TYPE ecs_task_metadata_info gauge
ecs_task_metadata_info{availability_zone="zone",cluster=%q,desired_status="RUNNING",family="family",known_status="RUNNING",launch_type="FARGATE",revision="1",task_arn="task-arn"} 1
`, cluster)
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected), "ecs_task_metadata_info"); err != nil {
		t.Fatalf("unexpected metadata metric for %s: %v", cluster, err)
	}
}

func waitForSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func TestRuntimeConcurrentShutdown(t *testing.T) {
	runtime, _, server := newFixtureRuntime(t, "cluster")
	defer server.Close()

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			if err := runtime.Shutdown(context.Background()); err != nil {
				t.Errorf("failed to shut down runtime: %v", err)
			}
		})
	}
	wg.Wait()
}
