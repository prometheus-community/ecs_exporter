# ecs_exporter

[![CircleCI](https://circleci.com/gh/prometheus-community/ecs_exporter/tree/main.svg?style=svg)](https://circleci.com/gh/prometheus-community/ecs_exporter/tree/main)
[![Go package](https://pkg.go.dev/badge/github.com/prometheus-community/ecs_exporter?status.svg)](https://pkg.go.dev/github.com/prometheus-community/ecs_exporter)

🚧 🚧 🚧 This repo is still work in progress and is subject to change.

This repo contains a Prometheus exporter for
Amazon Elastic Container Service (ECS) that publishes
ECS task infra metrics in Prometheus format.

Run the following container as a sidecar on ECS tasks:

```
quay.io/prometheuscommunity/ecs-exporter:latest
```

An example Fargate task definition that includes the container
is [available](#example-task-definition).

To add ECS exporter to your existing ECS task:

1. Go to ECS task definitions.
1. Click on "Create new revision".
1. Scroll down to "Container definitions" and click on "Add container".
1. Set "ecs-exporter" as container name.
1. Copy the container image URL from above.
1. Add tcp/9779 as a port mapping.
1. Click on "Add" to return back to task definition page.
1. Click on "Create" to create a new revision.

By default, it publishes Prometheus metrics on ":9779/metrics". The exporter in this repo can be a useful complementary sidecar for the scenario described in [this blog post](https://aws.amazon.com/blogs/opensource/metrics-collection-from-amazon-ecs-using-amazon-managed-service-for-prometheus/). Adding this sidecar to the ECS task definition would export task-level metrics in addition to the custom metrics described in the blog.

The sidecar process is also supported on [AWS App Runner](https://aws.amazon.com/apprunner/)
and can be used to publish infra metrics in Prometheus format
from App Runner services.

## Labels

### On task-level metrics
None.

### On container-level metrics

* **container_name**: Name of the container (as in the ECS task definition) associated with a metric.
* **interface**: Network interface device associated with the metric. Only
  available for several network metrics.

## Example output

Check out the [metrics snapshots](./ecscollector/testdata/snapshots) which
contain sample metrics emitted by ecs_exporter in the [Prometheus text
format](https://prometheus.io/docs/instrumenting/exposition_formats/#text-based-format)
you should expect to see on /metrics. Note that these snapshots behave as if
`--web.disable-exporter-metrics` were passed when running ecs_exporter, such
that standard [client_golang](https://github.com/prometheus/client_golang)
metrics are not included.
