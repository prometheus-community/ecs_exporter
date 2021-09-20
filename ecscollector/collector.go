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
	"log"

	"github.com/prometheus-community/ecs_exporter/ecsmetadata"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	cpuTotalDesc = prometheus.NewDesc(
		"ecs_cpu_seconds_total",
		"Total CPU usage in seconds.",
		cpuLabels, nil)

	memUsageDesc = prometheus.NewDesc(
		"ecs_memory_bytes",
		"Memory usage in bytes.",
		labels, nil)

	memMaxUsageDesc = prometheus.NewDesc(
		"ecs_memory_max_bytes",
		"Maximum memory usage in bytes.",
		labels, nil)

	memLimitDesc = prometheus.NewDesc(
		"ecs_memory_limit_bytes",
		"Memory limit in bytes.",
		labels, nil)

	networkRxBytesDesc = prometheus.NewDesc(
		"ecs_network_receive_bytes_total",
		"Network recieved in bytes.",
		networkLabels, nil)

	networkRxPacketsDesc = prometheus.NewDesc(
		"ecs_network_receive_packets_total",
		"Network packets recieved.",
		networkLabels, nil)

	networkRxDroppedDesc = prometheus.NewDesc(
		"ecs_network_receive_dropped_total",
		"Network packets dropped in recieving.",
		networkLabels, nil)

	networkRxErrorsDesc = prometheus.NewDesc(
		"ecs_network_receive_errors_total",
		"Network errors in recieving.",
		networkLabels, nil)

	networkTxBytesDesc = prometheus.NewDesc(
		"ecs_network_transmit_bytes_total",
		"Network transmitted in bytes.",
		networkLabels, nil)

	networkTxPacketsDesc = prometheus.NewDesc(
		"ecs_network_transmit_packets_total",
		"Network packets transmitted.",
		networkLabels, nil)

	networkTxDroppedDesc = prometheus.NewDesc(
		"ecs_network_transmit_dropped_total",
		"Network packets dropped in transmit.",
		networkLabels, nil)

	networkTxErrorsDesc = prometheus.NewDesc(
		"ecs_network_transmit_errors_total",
		"Network errors in transmit.",
		networkLabels, nil)
)

var labels = []string{
	"container",
}

var cpuLabels = append(
	labels,
	"cpu",
)

var networkLabels = append(
	labels,
	"device",
)

// NewCollector returns a new Collector that queries ECS metadata server
// for ECS task and container metrics.
func NewCollector(client *ecsmetadata.Client) prometheus.Collector {
	return &collector{client: client}
}

type collector struct {
	client *ecsmetadata.Client
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- cpuTotalDesc
	ch <- memUsageDesc
	ch <- memMaxUsageDesc
	ch <- memLimitDesc
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
		log.Printf("Failed to retrieve metadata: %v", err)
		return
	}
	stats, err := c.client.RetrieveTaskStats(ctx)
	if err != nil {
		log.Printf("Failed to retrieve container stats: %v", err)
		return
	}
	for _, container := range metadata.Containers {
		s := stats[container.DockerID]
		if s == nil {
			log.Printf("Couldn't find container with ID %q in stats", container.DockerID)
			continue
		}

		labelVals := []string{
			container.Name,
		}

		for i, cpuUsage := range s.CPUStats.CPUUsage.PerCPUUsage {
			cpu := fmt.Sprintf("%d", i)
			ch <- prometheus.MustNewConstMetric(
				cpuTotalDesc,
				prometheus.CounterValue,
				cpuJiffiesToSeconds(cpuUsage),
				append(labelVals, cpu)...,
			)
		}

		for desc, value := range map[*prometheus.Desc]float64{
			memUsageDesc:    s.MemoryStats.Usage,
			memMaxUsageDesc: s.MemoryStats.MaxUsage,
			memLimitDesc:    s.MemoryStats.Limit,
		} {
			ch <- prometheus.MustNewConstMetric(
				desc,
				prometheus.GaugeValue,
				value,
				labelVals...,
			)
		}

		// Network metrics per inteface.
		for iface, netStats := range s.Networks {
			networkLabelVals := append(labelVals, iface)

			for desc, value := range map[*prometheus.Desc]float64{
				networkRxBytesDesc:   netStats.RxBytes,
				networkRxPacketsDesc: netStats.RxPackets,
				networkRxDroppedDesc: netStats.RxDropped,
				networkRxErrorsDesc:  netStats.RxErrors,
				networkTxBytesDesc:   netStats.TxBytes,
				networkTxPacketsDesc: netStats.TxPackets,
				networkTxDroppedDesc: netStats.TxDropped,
				networkTxErrorsDesc:  netStats.TxErrors,
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

// cpuJiffiesToSeconds converts CPU metrics
// in jiffies to seconds.
func cpuJiffiesToSeconds(j float64) float64 {
	return j / float64(clockTick)
}

var clockTick int64 = 100 // Clock ticks are platform dependent, read from system config.
