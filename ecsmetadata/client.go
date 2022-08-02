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

// Package ecsmetadata queries ECS Metadata Server for ECS task metrics.
// This package is currently experimental and is subject to change.
package ecsmetadata

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	dockertypes "github.com/docker/docker/api/types"
)

type Client struct {
	// HTTClient is the client to use when making HTTP requests when set.
	HTTPClient *http.Client

	// metadata server endpoint
	endpoint string
}

// NewClient returns a new Client. endpoint is the metadata server endpoint.
func NewClient(endpoint string) *Client {
	return &Client{
		HTTPClient: &http.Client{},
		endpoint:   endpoint,
	}
}

// NewClientFromEnvironment is like NewClient but endpoint
// is discovered from the environment.
func NewClientFromEnvironment() (*Client, error) {
	const endpointEnv = "ECS_CONTAINER_METADATA_URI_V4"
	endpoint := os.Getenv(endpointEnv)
	if endpoint == "" {
		return nil, fmt.Errorf("%q environmental variable is not set; not running on ECS", endpointEnv)
	}
	_, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("can't parse %q as URL: %w", endpointEnv, err)
	}
	return NewClient(endpoint), nil
}

func (c *Client) RetrieveTaskStats(ctx context.Context) (map[string]*ContainerStats, error) {
	out := make(map[string]*ContainerStats)
	err := c.request(ctx, c.endpoint+"/task/stats", &out)
	return out, err
}

func (c *Client) RetrieveTaskMetadata(ctx context.Context) (*TaskMetadata, error) {
	var out TaskMetadata
	err := c.request(ctx, c.endpoint+"/task", &out)
	return &out, err
}

func (c *Client) request(ctx context.Context, uri string, out interface{}) error {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

type ContainerStats struct {
	Name     string  `json:"name"`
	ID       string  `json:"id"`
	NumProcs float64 `json:"num_procs"`

	CPUStats    dockertypes.CPUStats
	PreCPUStats dockertypes.CPUStats
	MemoryStats dockertypes.MemoryStats

	Networks map[string]struct {
		RxBytes   float64 `json:"rx_bytes"`
		RxPackets float64 `json:"rx_packets"`
		RxErrors  float64 `json:"rx_errors"`
		RxDropped float64 `json:"rx_dropped"`
		TxBytes   float64 `json:"tx_bytes"`
		TxPackets float64 `json:"tx_packets"`
		TxErrors  float64 `json:"tx_errors"`
		TxDropped float64 `json:"tx_dropped"`
	} `json:"networks"`

	NetworkRateStats struct {
		RxBytesPerSec float64 `json:"rx_bytes_per_sec"`
		TxBytesPerSec float64 `json:"tx_bytes_per_sec"`
	} `json:"network_rate_stats"`
}

// TODO(jbd): Add storage stats.

type TaskMetadata struct {
	Cluster          string `json:"Cluster"`
	TaskARN          string `json:"TaskARN"`
	Family           string `json:"Family"`
	Revision         string `json:"Revision"`
	DesiredStatus    string `json:"DesiredStatus"`
	KnownStatus      string `json:"KnownStatus"`
	AvailabilityZone string `json:"AvailabilityZone"`
	LaunchType       string `json:"LaunchType"`
	Containers       []struct {
		DockerID      string            `json:"DockerId"`
		Name          string            `json:"Name"`
		DockerName    string            `json:"DockerName"`
		Image         string            `json:"Image"`
		ImageID       string            `json:"ImageID"`
		Labels        map[string]string `json:"Labels"`
		DesiredStatus string            `json:"DesiredStatus"`
		KnownStatus   string            `json:"KnownStatus"`
		Type          string            `json:"Type"`
		ContainerARN  string            `json:"ContainerARN"`
	} `json:"Containers"`
}
