# ecs_exporter

[![CircleCI](https://circleci.com/gh/prometheus-community/ecs_exporter/tree/main.svg?style=svg)](https://circleci.com/gh/prometheus-community/ecs_exporter/tree/main)
[![Go package](https://pkg.go.dev/badge/github.com/prometheus-community/ecs_exporter?status.svg)](https://pkg.go.dev/github.com/prometheus-community/ecs_exporter)

This repo contains a Prometheus exporter for Amazon Elastic Container Service
(ECS) that publishes [ECS task infra
metrics](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4.html)
in the [Prometheus exposition
formats](https://prometheus.io/docs/instrumenting/exposition_formats/).

Run the following container as a sidecar on ECS tasks:

```
quay.io/prometheuscommunity/ecs-exporter:v0.4.0
```

An example Fargate task definition that includes the container
is [available](#example-task-definition).

To add ECS exporter to your existing ECS task:

1. Go to ECS task definitions.
1. Click on "Create new revision".
1. Scroll down to "Container definitions" and click on "Add container".
1. Set "ecs-exporter" as container name.
1. Copy the container image URL from above. (Use the tag for the [latest
   release](https://github.com/prometheus-community/ecs_exporter/releases).)
1. Add tcp/9779 as a port mapping.
1. Click on "Add" to return back to task definition page.
1. Click on "Create" to create a new revision.

By default, it publishes Prometheus metrics on ":9779/metrics". The exporter in this repo can be a useful complementary sidecar for the scenario described in [this blog post](https://aws.amazon.com/blogs/opensource/metrics-collection-from-amazon-ecs-using-amazon-managed-service-for-prometheus/). Adding this sidecar to the ECS task definition would export task-level metrics in addition to the custom metrics described in the blog.

## Compatibility guarantees
All metrics exported by ecs_exporter are sourced from the [ECS task metadata
API](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4.html)
accessible from within every ECS task. AWS can make, and on occasion [has
made](https://github.com/prometheus-community/ecs_exporter/issues/74#issuecomment-2395293862),
unannounced (or at least unversioned) breaking changes to the data served from
this API, especially the container-level stats served from the `/task/stats`
endpoint. Metrics emitted by ecs_exporter may spontaneously break as a result,
in which case we may need to make breaking changes to ecs_exporter to keep up.

In light of these conditions, we currently do not have plans to cut a 1.0
release. When necessary, breaking changes will continue to land in minor version
releases. The [release
notes](https://github.com/prometheus-community/ecs_exporter/releases) will
document any breaking changes as they come.

## Labels

### On task-level metrics

None. You may
[join](https://grafana.com/blog/2021/08/04/how-to-use-promql-joins-for-more-effective-queries-of-prometheus-metrics-at-scale/)
to `ecs_task_metadata_info` to add task-level metadata (such as the task ARN) to
task-level or any other metrics emitted by ecs_exporter.

### On container-level metrics

* **container_name**: Name of the container (as in the ECS task definition) associated with the metric.

### On network-level metrics

* **interface**: Network interface device associated with the metric.

## Example output

Check out the [metrics snapshots](./ecscollector/testdata/snapshots) which
contain sample metrics emitted by ecs_exporter in the [Prometheus text
format](https://prometheus.io/docs/instrumenting/exposition_formats/#text-based-format)
you should expect to see on /metrics. Note that these snapshots behave as if
`--web.disable-exporter-metrics` were passed when running ecs_exporter, such
that standard [client_golang](https://github.com/prometheus/client_golang)
metrics are not included.
