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
	"fmt"
	"log/slog"

	"github.com/docker/docker/api/types/container"
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

	restartTotalDesc = prometheus.NewDesc(
		"ecs_container_restarts_total",
		"Cumulative total count of container restarts. Only has a value if the container has been configured to restart on failure.",
		containerLabels, nil)

	cpuTotalDesc = prometheus.NewDesc(
		"ecs_container_cpu_usage_seconds_total",
		"Cumulative total container CPU usage in seconds.",
		containerLabels, nil)

	memUsageDesc = prometheus.NewDesc(
		"ecs_container_memory_usage_bytes",
		"Current container memory usage reported by the runtime in bytes, including page cache and accounted kernel memory.",
		containerLabels, nil)

	memLimitDesc = prometheus.NewDesc(
		"ecs_container_memory_limit_bytes",
		"Configured memory limit applicable to the container in bytes, using the positive container setting when present, otherwise the task setting shared by the task's containers.",
		containerLabels, nil)

	memCacheSizeDesc = prometheus.NewDesc(
		"ecs_container_memory_page_cache_size_bytes",
		"Current page cache memory accounted to the container in bytes, including tmpfs and shared memory.",
		containerLabels, nil)

	memAnonymousDesc = prometheus.NewDesc(
		"ecs_container_memory_anonymous_bytes",
		"Current anonymous memory charged to the container cgroup in bytes, excluding file-backed memory.",
		containerLabels, nil)

	memAnonymousTHPDesc = prometheus.NewDesc(
		"ecs_container_memory_anonymous_thp_bytes",
		"Current anonymous memory backed by transparent huge pages in bytes.",
		containerLabels, nil)

	memMappedFileDesc = prometheus.NewDesc(
		"ecs_container_memory_mapped_file_bytes",
		"Current mapped file memory accounted to the container in bytes, including mapped tmpfs and shared memory.",
		containerLabels, nil)

	memActiveFileDesc = prometheus.NewDesc(
		"ecs_container_memory_active_file_bytes",
		"Current memory on the container cgroup's active file LRU list in bytes.",
		containerLabels, nil)

	memInactiveFileDesc = prometheus.NewDesc(
		"ecs_container_memory_inactive_file_bytes",
		"Current memory on the container cgroup's inactive file LRU list in bytes.",
		containerLabels, nil)

	memDirtyDesc = prometheus.NewDesc(
		"ecs_container_memory_dirty_bytes",
		"Current dirty memory accounted to the container awaiting writeback in bytes.",
		containerLabels, nil)

	memWritebackDesc = prometheus.NewDesc(
		"ecs_container_memory_writeback_bytes",
		"Current memory accounted to the container under writeback in bytes.",
		containerLabels, nil)

	memUnevictableDesc = prometheus.NewDesc(
		"ecs_container_memory_unevictable_bytes",
		"Current memory on the container cgroup's unevictable LRU list in bytes.",
		containerLabels, nil)

	memSharedDesc = prometheus.NewDesc(
		"ecs_container_memory_shared_bytes",
		"Current tmpfs, shared memory, and shared anonymous memory accounted to the container in bytes. Only available with cgroup v2.",
		containerLabels, nil)

	memKernelStackDesc = prometheus.NewDesc(
		"ecs_container_memory_kernel_stack_bytes",
		"Current memory allocated to the container's kernel stacks in bytes. Only available with cgroup v2.",
		containerLabels, nil)

	memSocketDesc = prometheus.NewDesc(
		"ecs_container_memory_socket_bytes",
		"Current network transmission buffer memory accounted to the container in bytes. Only available with cgroup v2.",
		containerLabels, nil)

	memSlabReclaimableDesc = prometheus.NewDesc(
		"ecs_container_memory_slab_reclaimable_bytes",
		"Current reclaimable kernel slab memory accounted to the container in bytes. Only available with cgroup v2.",
		containerLabels, nil)

	memSlabUnreclaimableDesc = prometheus.NewDesc(
		"ecs_container_memory_slab_unreclaimable_bytes",
		"Current unreclaimable kernel slab memory accounted to the container in bytes. Only available with cgroup v2.",
		containerLabels, nil)

	memMaxUsageDesc = prometheus.NewDesc(
		"ecs_container_memory_max_usage_bytes",
		"Maximum container memory usage recorded by the runtime in bytes.",
		containerLabels, nil)

	memPageFaultsDesc = prometheus.NewDesc(
		"ecs_container_memory_page_faults_total",
		"Cumulative number of container cgroup page faults, including major faults.",
		containerLabels, nil)

	memMajorPageFaultsDesc = prometheus.NewDesc(
		"ecs_container_memory_major_page_faults_total",
		"Cumulative number of container cgroup major page faults.",
		containerLabels, nil)

	memPagesScannedDesc = prometheus.NewDesc(
		"ecs_container_memory_pages_scanned_total",
		"Cumulative number of container cgroup pages scanned from the inactive LRU list. Only available with cgroup v2.",
		containerLabels, nil)

	memPagesReclaimedDesc = prometheus.NewDesc(
		"ecs_container_memory_pages_reclaimed_total",
		"Cumulative number of container cgroup pages reclaimed. Only available with cgroup v2.",
		containerLabels, nil)

	memLimitFailuresDesc = prometheus.NewDesc(
		"ecs_container_memory_limit_failures_total",
		"Cumulative number of times a container cgroup memory charge encountered its limit. This is not an OOM kill count and is only available with cgroup v1.",
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
		"ecs_network_transmit_packets_dropped_total",
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
	ch <- restartTotalDesc
	ch <- cpuTotalDesc
	ch <- memUsageDesc
	ch <- memLimitDesc
	ch <- memCacheSizeDesc
	ch <- memAnonymousDesc
	ch <- memAnonymousTHPDesc
	ch <- memMappedFileDesc
	ch <- memActiveFileDesc
	ch <- memInactiveFileDesc
	ch <- memDirtyDesc
	ch <- memWritebackDesc
	ch <- memUnevictableDesc
	ch <- memSharedDesc
	ch <- memKernelStackDesc
	ch <- memSocketDesc
	ch <- memSlabReclaimableDesc
	ch <- memSlabUnreclaimableDesc
	ch <- memMaxUsageDesc
	ch <- memPageFaultsDesc
	ch <- memMajorPageFaultsDesc
	ch <- memPagesScannedDesc
	ch <- memPagesReclaimedDesc
	ch <- memLimitFailuresDesc
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
		c.logger.Debug("Failed to retrieve task metadata", "error", err)
		// Signal that this Collect has failed. This ultimately results in an
		// HTTP 500 being served for the /metrics response.
		//
		// While it would be most technically correct to do `NewInvalidMetric`
		// for all of the Descs here, it just results in N identical error
		// messages being printed in the /metrics response, so it seems a bit
		// absurd to do that.
		ch <- prometheus.NewInvalidMetric(taskMetadataDesc, fmt.Errorf("failed to retrieve task metadata: %w", err))
		return
	}
	c.logger.Debug("Got ECS task metadata response", "metadata", metadata)

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
		c.logger.Debug("Failed to retrieve task stats", "error", err)
		// Signal that this Collect has failed. This ultimately results in an
		// HTTP 500 being served for the /metrics response.
		//
		// While it would be most technically correct to do `NewInvalidMetric`
		// for all of the Descs here, it just results in N identical error
		// messages being printed in the /metrics response, so it seems a bit
		// absurd to do that.
		ch <- prometheus.NewInvalidMetric(memUsageDesc, fmt.Errorf("failed to retrieve task stats: %w", err))
		return
	}
	c.logger.Debug("Got ECS task stats response", "stats", stats)

	networks := make(map[string]*container.NetworkStats)
	for _, container := range metadata.Containers {
		s := stats[container.ID]
		if s == nil || s.StatsJSON == nil {
			// This can happen if the container is stopped; if it's
			// nonessential, the task goes on.
			c.logger.Debug("Couldn't find stats for container", "id", container.ID)
			continue
		}

		containerLabelVals := []string{
			container.Name,
		}

		if container.RestartCount != nil {
			ch <- prometheus.MustNewConstMetric(
				restartTotalDesc,
				prometheus.CounterValue,
				float64(*container.RestartCount),
				containerLabelVals...,
			)
		}

		ch <- prometheus.MustNewConstMetric(
			cpuTotalDesc,
			prometheus.CounterValue,
			float64(s.CPUStats.CPUUsage.TotalUsage)*nanoseconds,
			containerLabelVals...,
		)

		ch <- prometheus.MustNewConstMetric(
			memUsageDesc,
			prometheus.GaugeValue,
			float64(s.MemoryStats.Usage),
			containerLabelVals...,
		)
		if s.MemoryStats.MaxUsage > 0 {
			ch <- prometheus.MustNewConstMetric(
				memMaxUsageDesc,
				prometheus.GaugeValue,
				float64(s.MemoryStats.MaxUsage),
				containerLabelVals...,
			)
		}

		// A positive container memory setting is a dedicated hard limit. Zero
		// means that no container-level setting was configured, so the shared
		// task setting is the best configuration-level upper bound available.
		var taskMemoryLimit *int64
		if metadata.Limits != nil {
			taskMemoryLimit = metadata.Limits.Memory
		}
		if configuredLimitMib, ok := configuredMemoryLimitMib(container.Limits.Memory, taskMemoryLimit); ok {
			ch <- prometheus.MustNewConstMetric(
				memLimitDesc,
				prometheus.GaugeValue,
				float64(configuredLimitMib)*mebibytes,
				containerLabelVals...,
			)
		}

		memoryStats := s.MemoryStats.Stats
		// Moby's cgroup v2 response always has anon; its v1 response has
		// active_anon and inactive_anon, but no unqualified anon key.
		_, cgroupV2 := memoryStats["anon"]
		if !cgroupV2 {
			ch <- prometheus.MustNewConstMetric(
				memLimitFailuresDesc,
				prometheus.CounterValue,
				float64(s.MemoryStats.Failcnt),
				containerLabelVals...,
			)
		}
		for _, metric := range []struct {
			desc      *prometheus.Desc
			valueType prometheus.ValueType
			v1Key     string
			v2Key     string
		}{
			{memCacheSizeDesc, prometheus.GaugeValue, "cache", "file"},
			{memAnonymousDesc, prometheus.GaugeValue, "rss", "anon"},
			{memAnonymousTHPDesc, prometheus.GaugeValue, "rss_huge", "anon_thp"},
			{memMappedFileDesc, prometheus.GaugeValue, "mapped_file", "file_mapped"},
			{memActiveFileDesc, prometheus.GaugeValue, "active_file", "active_file"},
			{memInactiveFileDesc, prometheus.GaugeValue, "inactive_file", "inactive_file"},
			{memDirtyDesc, prometheus.GaugeValue, "dirty", "file_dirty"},
			{memWritebackDesc, prometheus.GaugeValue, "writeback", "file_writeback"},
			{memUnevictableDesc, prometheus.GaugeValue, "unevictable", "unevictable"},
			{memSharedDesc, prometheus.GaugeValue, "", "shmem"},
			{memKernelStackDesc, prometheus.GaugeValue, "", "kernel_stack"},
			{memSocketDesc, prometheus.GaugeValue, "", "sock"},
			{memSlabReclaimableDesc, prometheus.GaugeValue, "", "slab_reclaimable"},
			{memSlabUnreclaimableDesc, prometheus.GaugeValue, "", "slab_unreclaimable"},
			{memPageFaultsDesc, prometheus.CounterValue, "pgfault", "pgfault"},
			{memMajorPageFaultsDesc, prometheus.CounterValue, "pgmajfault", "pgmajfault"},
			{memPagesScannedDesc, prometheus.CounterValue, "", "pgscan"},
			{memPagesReclaimedDesc, prometheus.CounterValue, "", "pgsteal"},
		} {
			value, ok := normalizedMemoryStat(memoryStats, cgroupV2, metric.v1Key, metric.v2Key)
			if !ok {
				continue
			}
			ch <- prometheus.MustNewConstMetric(metric.desc, metric.valueType, float64(value), containerLabelVals...)
		}

		// Network metrics per interface.
		for iface, netStats := range s.Networks {
			// While the API response attaches network stats to each container,
			// the container is in fact not a relevant dimension; only the
			// interface is. This means that if multiple containers use the same
			// network (extremely likely), we are redundantly writing this
			// metric with "last one wins" semantics. This is fine: the values
			// for an interface are the same across all containers.
			//
			// The collection process will error if you report the same metric
			// multiple times, however, so we have to stash this data in the
			// `netStats` map to ensure that we only send one metric per
			// interface.
			networks[iface] = &netStats
		}
	}

	for iface, netStats := range networks {
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

func normalizedMemoryStat(stats map[string]uint64, cgroupV2 bool, v1Key, v2Key string) (uint64, bool) {
	if cgroupV2 {
		if v2Key == "" {
			return 0, false
		}
		value, ok := stats[v2Key]
		return value, ok
	}

	if v1Key == "" {
		return 0, false
	}
	if value, ok := stats["total_"+v1Key]; ok {
		return value, true
	}
	value, ok := stats[v1Key]
	return value, ok
}

func configuredMemoryLimitMib(containerLimit, taskLimit *int64) (int64, bool) {
	if containerLimit != nil && *containerLimit > 0 {
		return *containerLimit, true
	}
	if taskLimit == nil || *taskLimit <= 0 {
		return 0, false
	}
	return *taskLimit, true
}
