---
title: Getting Started
sort_rank: 2
---
# Getting Started

The exporter only works inside an ECS task, because it reads that task's metadata endpoint. You add it as one more container in the task definition of each task you want to monitor.

## Add the exporter to a task

### In the ECS console

1. Go to ECS task definitions and open your task definition.
1. Click "Create new revision".
1. Under "Container definitions", click "Add container".
1. Set the container name to `ecs-exporter`.
1. Set the image to `quay.io/prometheuscommunity/ecs-exporter:v0.7.0`. Use the tag of the [latest release](https://github.com/prometheus-community/ecs_exporter/releases).
1. Add a port mapping for TCP port 9779.
1. Click "Add" to go back to the task definition, then click "Create" to create the new revision.

Then update your service to use the new revision.

### In a task definition file

Add a container like this to the task definition's `containerDefinitions`:

```json
{
  "name": "ecs-exporter",
  "image": "quay.io/prometheuscommunity/ecs-exporter:v0.7.0",
  "portMappings": [
    {
      "containerPort": 9779,
      "protocol": "tcp"
    }
  ]
}
```

The exporter needs no environment variables, command line flags or IAM permissions. ECS sets `ECS_CONTAINER_METADATA_URI_V4` in the container, and the exporter reads it at startup. Flags only change the defaults. See [Configuring the Exporter](configuration.md).

## Container images

Images are published for every release to three registries:

- `quay.io/prometheuscommunity/ecs-exporter`
- `docker.io/prometheuscommunity/ecs-exporter`
- `ghcr.io/prometheus-community/ecs-exporter` (from v0.5.0)

Tags:

- `vX.Y.Z`, such as `v0.7.0`, for a release. Pin one of these in your task definitions.
- `v0` and `latest` for the newest release.
- `main` for the newest build of the main branch.

The images are multi-architecture (`amd64`, `arm64`, `arm/v7`, `ppc64le`, `riscv64` and `s390x`), so they run on both x86 and Graviton tasks. They run as the `nobody` user and expose port 9779.

## Check that it works

Once the task is running, the exporter serves metrics on port 9779 at `/metrics`. With the `awsvpc` network mode (always used on Fargate) that is `http://<task IP address>:9779/metrics`. With the `bridge` or `host` network modes, use the container instance's address and the host port that ECS maps to 9779.

The first lines of a healthy response look like this:

```
# HELP ecs_container_cpu_kernel_seconds_total Cumulative container CPU time spent in kernel mode in seconds. With cgroup v1, this is sampled at clock tick granularity, so user and kernel time may not add up to total CPU usage.
# TYPE ecs_container_cpu_kernel_seconds_total counter
ecs_container_cpu_kernel_seconds_total{container_name="ecs-exporter"} 0.1
ecs_container_cpu_kernel_seconds_total{container_name="main"} 1.09
```

There is one series per container in the task, including the exporter itself. If `/metrics` answers with HTTP 500 instead, see [Errors](configuration.md#errors).

## Scraping

Prometheus has to reach port 9779 of every task you monitor. How it finds the tasks depends on your setup. One way, with the AWS Distro for OpenTelemetry collector and Amazon Managed Service for Prometheus, is described in [this AWS blog post](https://aws.amazon.com/blogs/opensource/metrics-collection-from-amazon-ecs-using-amazon-managed-service-for-prometheus/). The exporter adds task-level metrics to the application metrics that post collects.

Each task has its own exporter, so series from different tasks differ only by the target labels Prometheus adds, such as `instance`. To see which task a series belongs to, join it with `ecs_task_metadata_info`, as shown in [Metrics](metrics.md#labels).
