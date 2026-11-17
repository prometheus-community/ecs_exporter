---
title: ecs_exporter Documentation
nav_title: Overview
sort_rank: 1
---
# ecs_exporter Documentation

`ecs_exporter` is a [Prometheus](https://prometheus.io/) exporter for [Amazon Elastic Container Service](https://aws.amazon.com/ecs/) (ECS). It runs as a sidecar container in an ECS task, reads the task's resource usage from the [ECS task metadata endpoint version 4](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4.html), and exposes it as Prometheus metrics over HTTP.

It is a [Prometheus Community](https://github.com/prometheus-community) project.

## How it works

ECS gives every container in a task the address of the task metadata endpoint in the `ECS_CONTAINER_METADATA_URI_V4` environment variable. On each scrape, the exporter reads two paths from it:

- `/task` for the task's metadata, limits and clock status.
- `/task/stats` for the Docker-style resource stats of every container in the task.

The exporter makes no AWS API calls, so it needs no IAM permissions or AWS credentials. It only reports on the task it runs in, so you add one exporter container to each task you want to monitor.

## What it collects

All metrics start with `ecs_`. They come at three levels:

- **Task.** Task metadata as labels on `ecs_task_metadata_info`, the task's CPU and memory limits, ephemeral storage use, image pull times and clock synchronization status.
- **Container.** CPU time, memory usage and limits, a breakdown of cgroup memory accounting, page faults and restarts, labeled by `container_name`.
- **Network.** Bytes, packets, drops and errors received and transmitted, labeled by `interface`.

See [Metrics](metrics.md) for the full list and when each metric is present. By default `/metrics` also carries the exporter's own `ecs_exporter_build_info` and the standard Go runtime, process and `promhttp` metrics.

## Supported platforms

The exporter works with tasks on all three ECS capacity types: EC2, Fargate and Managed Instances. Each one serves a slightly different metadata response, so a few metrics are only present on some of them. The [test fixtures](https://github.com/prometheus-community/ecs_exporter/tree/main/ecscollector/testdata) are recorded from real tasks on each platform.

The task metadata endpoint version 4 is available on Fargate platform version 1.4.0 and later, and on EC2 Linux instances running version 1.39.0 or later of the ECS container agent.

## Compatibility guarantees

All metrics come from the task metadata endpoint. AWS can make, and on occasion [has made](https://github.com/prometheus-community/ecs_exporter/issues/74#issuecomment-2395293862), unannounced breaking changes to the data it serves, especially the container stats from `/task/stats`. Metrics may break as a result, and the exporter may need breaking changes to keep up.

Because of this there are no plans for a 1.0 release. Breaking changes can land in minor releases, and the [release notes](https://github.com/prometheus-community/ecs_exporter/releases) list them.

## Documentation

**Start here**

- [Getting Started](getting-started.md): add the exporter to a task, the container images, and scraping.

**Reference**

- [Configuring the Exporter](configuration.md): flags, HTTP endpoints, TLS and authentication, and errors.
- [Metrics](metrics.md): every metric, its labels, and when it is present.

## Getting help

- Bugs and feature requests: [GitHub issues](https://github.com/prometheus-community/ecs_exporter/issues).
- Questions and discussion: the [Prometheus community](https://prometheus.io/community/) channels.
