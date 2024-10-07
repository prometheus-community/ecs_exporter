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
	"io"
	"net/http"
	"net/url"
	"os"

	tmdsv4 "github.com/aws/amazon-ecs-agent/ecs-agent/tmds/handlers/v4/state"
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
		return nil, fmt.Errorf("%s is not set; not running on ECS?", endpointEnv)
	}
	_, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("can't parse %s as URL: %w", endpointEnv, err)
	}
	return NewClient(endpoint), nil
}

func (c *Client) RetrieveTaskStats(ctx context.Context) (map[string]*tmdsv4.StatsResponse, error) {
	// https://github.com/aws/amazon-ecs-agent/blob/cf8c7a6b65043c550533f330b10aef6d0a342214/agent/handlers/v4/tmdsstate.go#L202
	out := make(map[string]*tmdsv4.StatsResponse)
	err := c.request(ctx, c.endpoint+"/task/stats", &out)
	return out, err
}

func (c *Client) RetrieveTaskMetadata(ctx context.Context) (*tmdsv4.TaskResponse, error) {
	// https://github.com/aws/amazon-ecs-agent/blob/cf8c7a6b65043c550533f330b10aef6d0a342214/agent/handlers/v4/tmdsstate.go#L174
	//
	// Note that EC2 and Fargate return slightly different task metadata
	// responses. At time of writing, as per the documentation, only EC2 has `ServiceName`,
	// while only Fargate has `EphemeralStorageMetrics`, `ClockDrift`, and
	// `Containers[].Snapshotter`. Ref:
	// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4-fargate-response.html
	// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4-response.html
	//
	// But `TaskResponse` is the _union_ of these two responses. It has all the
	// fields.
	var out tmdsv4.TaskResponse
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%q: %s %s: %q", uri, resp.Proto, resp.Status, string(body)[:100])
	}

	return json.Unmarshal(body, out)
}
