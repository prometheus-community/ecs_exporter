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

// Package ecscollector implements a Prometheus collector for Amazon ECS
// metrics available at the ECS metadata server.
package ecscollector

import (
	"context"
	"log/slog"

	"github.com/prometheus-community/ecs_exporter/ecsmetadata"
	"github.com/prometheus/client_golang/prometheus"
)

// ECS cpu_stats are from upstream docker/moby. These values are in nanoseconds.
// https://github.com/moby/moby/blob/49f021ebf00a76d74f5ce158244083e2dfba26fb/api/types/stats.go#L18-L40
const nanoseconds = 1 / 1.0e9

// Task definition memory parameters are defined in MiB, while Prometheus
// standard metrics use bytes.
const mebibytes = 1024 * 1024

var (
	taskMetadataDesc = prometheus.NewDesc(
		"ecs_task_metadata_info",
		"ECS task metadata, sourced from the task metadata endpoint version 4.",
		taskMetadataLabels, nil)

	taskCpuLimitDesc = prometheus.NewDesc(
		"ecs_task_cpu_limit_vcpus",
		"Configured task CPU limit in vCPUs (1 vCPU = 1024 CPU units). This is optional when running on EC2; if no limit is set, this metric has no value.",
		taskLabels, nil)

	taskMemLimitDesc = prometheus.NewDesc(
		"ecs_task_memory_limit_bytes",
		"Configured task memory limit in bytes. This is optional when running on EC2; if no limit is set, this metric has no value.",
		taskLabels, nil)

	taskEphemeralStorageUsedDesc = prometheus.NewDesc(
		"ecs_task_ephemeral_storage_used_bytes",
		"Current Fargate task ephemeral storage usage in bytes.",
		taskLabels, nil)

	taskEphemeralStorageAllocatedDesc = prometheus.NewDesc(
		"ecs_task_ephemeral_storage_allocated_bytes",
		"Configured Fargate task ephemeral storage allocated size in bytes.",
		taskLabels, nil)

	taskImagePullStartDesc = prometheus.NewDesc(
		"ecs_task_image_pull_start_timestamp_seconds",
		"The time at which the task started pulling docker images for its containers.",
		taskLabels, nil)

	taskImagePullStopDesc = prometheus.NewDesc(
		"ecs_task_image_pull_stop_timestamp_seconds",
		"The time at which the task stopped (i.e. completed) pulling docker images for its containers.",
		taskLabels, nil)

	cpuTotalDesc = prometheus.NewDesc(
		"ecs_container_cpu_usage_seconds_total",
		"Cumulative total container CPU usage in seconds.",
		containerLabels, nil)

	memUsageDesc = prometheus.NewDesc(
		"ecs_container_memory_usage_bytes",
		"Current container memory usage in bytes.",
		containerLabels, nil)

	memLimitDesc = prometheus.NewDesc(
		"ecs_container_memory_limit_bytes",
		"Configured container memory limit in bytes, set from the container-level limit in the task definition if any, otherwise the task-level limit.",
		containerLabels, nil)

	memCacheSizeDesc = prometheus.NewDesc(
		"ecs_container_memory_page_cache_size_bytes",
		"Current container memory page cache size in bytes. This is not a subset of used bytes.",
		containerLabels, nil)

	networkRxBytesDesc = prometheus.NewDesc(
		"ecs_network_receive_bytes_total",
		"Cumulative total size of network packets received in bytes.",
		networkLabels, nil)

	networkRxPacketsDesc = prometheus.NewDesc(
		"ecs_network_receive_packets_total",
		"Cumulative total count of network packets received.",
		networkLabels, nil)

	networkRxDroppedDesc = prometheus.NewDesc(
		"ecs_network_receive_packets_dropped_total",
		"Cumulative total count of network packets dropped in receiving.",
		networkLabels, nil)

	networkRxErrorsDesc = prometheus.NewDesc(
		"ecs_network_receive_errors_total",
		"Cumulative total count of network errors in receiving.",
		networkLabels, nil)

	networkTxBytesDesc = prometheus.NewDesc(
		"ecs_network_transmit_bytes_total",
		"Cumulative total size of network packets transmitted in bytes.",
		networkLabels, nil)

	networkTxPacketsDesc = prometheus.NewDesc(
		"ecs_network_transmit_packets_total",
		"Cumulative total count of network packets transmitted.",
		networkLabels, nil)

	networkTxDroppedDesc = prometheus.NewDesc(
		"ecs_network_transmit_dropped_total",
		"Cumulative total count of network packets dropped in transmit.",
		networkLabels, nil)

	networkTxErrorsDesc = prometheus.NewDesc(
		"ecs_network_transmit_errors_total",
		"Cumulative total count of network errors in transmit.",
		networkLabels, nil)
)

var containerLabels = []string{
	"container_name",
}

var taskLabels = []string{}

var taskMetadataLabels = []string{
	"cluster",
	"task_arn",
	"family",
	"revision",
	"desired_status",
	"known_status",
	"availability_zone",
	"launch_type",
}

var networkLabels = []string{
	"interface",
}

// NewCollector returns a new Collector that queries ECS metadata server
// for ECS task and container metrics.
func NewCollector(client *ecsmetadata.Client, logger *slog.Logger) prometheus.Collector {
	return &collector{client: client, logger: logger}
}

type collector struct {
	client *ecsmetadata.Client
	logger *slog.Logger
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- taskMetadataDesc
	ch <- taskCpuLimitDesc
	ch <- taskMemLimitDesc
	ch <- taskEphemeralStorageUsedDesc
	ch <- taskEphemeralStorageAllocatedDesc
	ch <- taskImagePullStartDesc
	ch <- taskImagePullStopDesc
	ch <- cpuTotalDesc
	ch <- memUsageDesc
	ch <- memLimitDesc
	ch <- memCacheSizeDesc
	ch <- networkRxBytesDesc
	ch <- networkRxPacketsDesc
	ch <- networkRxDroppedDesc
	ch <- networkRxErrorsDesc
	ch <- networkTxBytesDesc
	ch <- networkTxPacketsDesc
	ch <- networkTxDroppedDesc
	ch <- networkTxErrorsDesc
}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()
	metadata, err := c.client.RetrieveTaskMetadata(ctx)
	if err != nil {
		c.logger.Debug("Failed to retrieve metadata", "error", err)
		return
	}
	c.logger.Debug("Got ECS task metadata response", "stats", metadata)

	ch <- prometheus.MustNewConstMetric(
		taskMetadataDesc,
		prometheus.GaugeValue,
		1.0,
		metadata.Cluster,
		metadata.TaskARN,
		metadata.Family,
		metadata.Revision,
		metadata.DesiredStatus,
		metadata.KnownStatus,
		metadata.AvailabilityZone,
		metadata.LaunchType,
	)

	// Task CPU/memory limits are optional when running on EC2 - the relevant
	// limits may only exist at the container level.
	if metadata.Limits != nil {
		if metadata.Limits.CPU != nil {
			ch <- prometheus.MustNewConstMetric(
				taskCpuLimitDesc,
				prometheus.GaugeValue,
				*metadata.Limits.CPU,
			)
		}
		if metadata.Limits.Memory != nil {
			ch <- prometheus.MustNewConstMetric(
				taskMemLimitDesc,
				prometheus.GaugeValue,
				float64(*metadata.Limits.Memory*mebibytes),
			)
		}
	}

	if metadata.EphemeralStorageMetrics != nil {
		ch <- prometheus.MustNewConstMetric(
			taskEphemeralStorageUsedDesc,
			prometheus.GaugeValue,
			float64(metadata.EphemeralStorageMetrics.UtilizedMiBs*mebibytes),
		)
		ch <- prometheus.MustNewConstMetric(
			taskEphemeralStorageAllocatedDesc,
			prometheus.GaugeValue,
			float64(metadata.EphemeralStorageMetrics.ReservedMiBs*mebibytes),
		)
	}

	if metadata.PullStartedAt != nil {
		ch <- prometheus.MustNewConstMetric(
			taskImagePullStartDesc,
			prometheus.GaugeValue,
			float64(metadata.PullStartedAt.UnixNano())*nanoseconds,
		)
	}
	if metadata.PullStoppedAt != nil {
		ch <- prometheus.MustNewConstMetric(
			taskImagePullStopDesc,
			prometheus.GaugeValue,
			float64(metadata.PullStoppedAt.UnixNano())*nanoseconds,
		)
	}

	stats, err := c.client.RetrieveTaskStats(ctx)
	if err != nil {
		c.logger.Debug("Failed to retrieve container stats", "error", err)
		return
	}
	c.logger.Debug("Got ECS task stats response", "stats", stats)

	for _, container := range metadata.Containers {
		s := stats[container.ID]
		if s == nil {
			c.logger.Debug("Couldn't find container with ID in stats", "id", container.ID)
			continue
		}

		containerLabelVals := []string{
			container.Name,
		}

		ch <- prometheus.MustNewConstMetric(
			cpuTotalDesc,
			prometheus.CounterValue,
			float64(s.CPUStats.CPUUsage.TotalUsage)*nanoseconds,
			containerLabelVals...,
		)

		cacheValue := 0.0
		if val, ok := s.MemoryStats.Stats["cache"]; ok {
			cacheValue = float64(val)
		}

		// Report the container's memory limit as its own, if any, otherwise the
		// task's limit. This is correct in that this is the precise logic used
		// to configure the cgroups limit for the container.
		var containerMemoryLimitMib int64
		if container.Limits.Memory != nil {
			containerMemoryLimitMib = *container.Limits.Memory
		} else {
			// This must be set if the container limit is not set, and thus is
			// safe to dereference.
			containerMemoryLimitMib = *metadata.Limits.Memory
		}
		for desc, value := range map[*prometheus.Desc]float64{
			memUsageDesc:     float64(s.MemoryStats.Usage),
			memLimitDesc:     float64(containerMemoryLimitMib * mebibytes),
			memCacheSizeDesc: cacheValue,
		} {
			ch <- prometheus.MustNewConstMetric(
				desc,
				prometheus.GaugeValue,
				value,
				containerLabelVals...,
			)
		}

		// Network metrics per interface.
		for iface, netStats := range s.Networks {
			// While the API response attaches network stats to each container,
			// the container is in fact not a relevant dimension; only the
			// interface is. This means that if multiple containers use the same
			// network (extremely likely), we are redundantly writing this
			// metric with "last one wins" semantics. This is fine: the values
			// for an interface are the same across all containers.
			networkLabelVals := []string{
				iface,
			}

			for desc, value := range map[*prometheus.Desc]float64{
				networkRxBytesDesc:   float64(netStats.RxBytes),
				networkRxPacketsDesc: float64(netStats.RxPackets),
				networkRxDroppedDesc: float64(netStats.RxDropped),
				networkRxErrorsDesc:  float64(netStats.RxErrors),
				networkTxBytesDesc:   float64(netStats.TxBytes),
				networkTxPacketsDesc: float64(netStats.TxPackets),
				networkTxDroppedDesc: float64(netStats.TxDropped),
				networkTxErrorsDesc:  float64(netStats.TxErrors),
			} {
				ch <- prometheus.MustNewConstMetric(
					desc,
					prometheus.CounterValue,
					value,
					networkLabelVals...,
				)
			}
		}
	}
}
