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

(With `--web.disable-exporter-metrics` passed, such that standard Go metrics are not included here.)

```
# HELP ecs_container_cpu_usage_seconds_total Cumulative total container CPU usage in seconds.
# TYPE ecs_container_cpu_usage_seconds_total counter
ecs_container_cpu_usage_seconds_total{container_name="ecs-exporter"} 0.028057878
# HELP ecs_container_memory_limit_bytes Configured container memory limit in bytes, set from the container-level limit in the task definition if any, otherwise the task-level limit.
# TYPE ecs_container_memory_limit_bytes gauge
ecs_container_memory_limit_bytes{container_name="ecs-exporter"} 5.36870912e+08
# HELP ecs_container_memory_page_cache_size_bytes Current container memory page cache size in bytes. This is not a subset of used bytes.
# TYPE ecs_container_memory_page_cache_size_bytes gauge
ecs_container_memory_page_cache_size_bytes{container_name="ecs-exporter"} 0
# HELP ecs_container_memory_usage_bytes Current container memory usage in bytes.
# TYPE ecs_container_memory_usage_bytes gauge
ecs_container_memory_usage_bytes{container_name="ecs-exporter"} 4.243456e+06
# HELP ecs_exporter_build_info A metric with a constant '1' value labeled by version, revision, branch, goversion from which ecs_exporter was built, and the goos and goarch for the build.
# TYPE ecs_exporter_build_info gauge
ecs_exporter_build_info{branch="",goarch="arm64",goos="linux",goversion="go1.23.2",revision="unknown",tags="unknown",version=""} 1
# HELP ecs_network_receive_bytes_total Cumulative total size of network packets received in bytes.
# TYPE ecs_network_receive_bytes_total counter
ecs_network_receive_bytes_total{interface="eth1"} 1.1172419e+07
# HELP ecs_network_receive_errors_total Cumulative total count of network errors in receiving.
# TYPE ecs_network_receive_errors_total counter
ecs_network_receive_errors_total{interface="eth1"} 0
# HELP ecs_network_receive_packets_dropped_total Cumulative total count of network packets dropped in receiving.
# TYPE ecs_network_receive_packets_dropped_total counter
ecs_network_receive_packets_dropped_total{interface="eth1"} 0
# HELP ecs_network_receive_packets_total Cumulative total count of network packets received.
# TYPE ecs_network_receive_packets_total counter
ecs_network_receive_packets_total{interface="eth1"} 8084
# HELP ecs_network_transmit_bytes_total Cumulative total size of network packets transmitted in bytes.
# TYPE ecs_network_transmit_bytes_total counter
ecs_network_transmit_bytes_total{interface="eth1"} 178817
# HELP ecs_network_transmit_dropped_total Cumulative total count of network packets dropped in transmit.
# TYPE ecs_network_transmit_dropped_total counter
ecs_network_transmit_dropped_total{interface="eth1"} 0
# HELP ecs_network_transmit_errors_total Cumulative total count of network errors in transmit.
# TYPE ecs_network_transmit_errors_total counter
ecs_network_transmit_errors_total{interface="eth1"} 0
# HELP ecs_network_transmit_packets_total Cumulative total count of network packets transmitted.
# TYPE ecs_network_transmit_packets_total counter
ecs_network_transmit_packets_total{interface="eth1"} 897
# HELP ecs_task_cpu_limit_vcpus Configured task CPU limit in vCPUs (1 vCPU = 1024 CPU units). This is optional when running on EC2; if no limit is set, this metric has no value.
# TYPE ecs_task_cpu_limit_vcpus gauge
ecs_task_cpu_limit_vcpus 0.25
# HELP ecs_task_ephemeral_storage_allocated_bytes Configured Fargate task ephemeral storage allocated size in bytes.
# TYPE ecs_task_ephemeral_storage_allocated_bytes gauge
ecs_task_ephemeral_storage_allocated_bytes 2.1491613696e+10
# HELP ecs_task_ephemeral_storage_used_bytes Current Fargate task ephemeral storage usage in bytes.
# TYPE ecs_task_ephemeral_storage_used_bytes gauge
ecs_task_ephemeral_storage_used_bytes 3.7748736e+07
# HELP ecs_task_image_pull_start_timestamp_seconds The time at which the task started pulling docker images for its containers.
# TYPE ecs_task_image_pull_start_timestamp_seconds gauge
ecs_task_image_pull_start_timestamp_seconds 1.737156015124145e+09
# HELP ecs_task_image_pull_stop_timestamp_seconds The time at which the task stopped (i.e. completed) pulling docker images for its containers.
# TYPE ecs_task_image_pull_stop_timestamp_seconds gauge
ecs_task_image_pull_stop_timestamp_seconds 1.7371560172684324e+09
# HELP ecs_task_memory_limit_bytes Configured task memory limit in bytes. This is optional when running on EC2; if no limit is set, this metric has no value.
# TYPE ecs_task_memory_limit_bytes gauge
ecs_task_memory_limit_bytes 5.36870912e+08
# HELP ecs_task_metadata_info ECS task metadata, sourced from the task metadata endpoint version 4.
# TYPE ecs_task_metadata_info gauge
ecs_task_metadata_info{availability_zone="us-east-1a",cluster="arn:aws:ecs:us-east-1:829490980523:cluster/prom-ecs-exporter-sandbox",desired_status="RUNNING",family="prom-ecs-exporter-sandbox-isker-fix-network-metrics-fargate",known_status="RUNNING",launch_type="FARGATE",revision="1",task_arn="arn:aws:ecs:us-east-1:829490980523:task/prom-ecs-exporter-sandbox/c8387acdc4884a0fa13dae78e68a989f"} 1
```

## Example task definition

```
{
  "ipcMode": null,
  "executionRoleArn": "arn:aws:iam::ACCOUNT_ID:role/ecsTaskExecutionRole",
  "containerDefinitions": [
    {
      "dnsSearchDomains": null,
      "environmentFiles": null,
      "logConfiguration": {
        "logDriver": "awslogs",
        "secretOptions": null,
        "options": {
          "awslogs-group": "/ecs/ecs-exporter",
          "awslogs-region": "us-west-2",
          "awslogs-stream-prefix": "ecs"
        }
      },
      "entryPoint": null,
      "portMappings": [
        {
          "hostPort": 9779,
          "protocol": "tcp",
          "containerPort": 9779
        }
      ],
      "command": null,
      "linuxParameters": null,
      "cpu": 0,
      "environment": [],
      "resourceRequirements": null,
      "ulimits": null,
      "dnsServers": null,
      "mountPoints": [],
      "workingDirectory": null,
      "secrets": null,
      "dockerSecurityOptions": null,
      "memory": null,
      "memoryReservation": null,
      "volumesFrom": [],
      "stopTimeout": null,
      "image": "quay.io/prometheuscommunity/ecs-exporter:v0.1.0",
      "startTimeout": null,
      "firelensConfiguration": null,
      "dependsOn": null,
      "disableNetworking": null,
      "interactive": null,
      "healthCheck": null,
      "essential": true,
      "links": null,
      "hostname": null,
      "extraHosts": null,
      "pseudoTerminal": null,
      "user": null,
      "readonlyRootFilesystem": null,
      "dockerLabels": null,
      "systemControls": null,
      "privileged": null,
      "name": "ecs-exporter"
    }
  ],
  "placementConstraints": [],
  "memory": "512",
  "taskRoleArn": "arn:aws:iam::ACCOUNT_ID:role/ecsTaskExecutionRole",
  "compatibilities": [
    "EC2",
    "FARGATE"
  ],
  "taskDefinitionArn": "arn:aws:ecs:us-west-2:ACCOUNT_ID:task-definition/ecs-exporter:1",
  "family": "ecs-exporter",
  "requiresAttributes": [
    {
      "targetId": null,
      "targetType": null,
      "value": null,
      "name": "com.amazonaws.ecs.capability.logging-driver.awslogs"
    },
    {
      "targetId": null,
      "targetType": null,
      "value": null,
      "name": "ecs.capability.execution-role-awslogs"
    },
    {
      "targetId": null,
      "targetType": null,
      "value": null,
      "name": "com.amazonaws.ecs.capability.docker-remote-api.1.19"
    },
    {
      "targetId": null,
      "targetType": null,
      "value": null,
      "name": "com.amazonaws.ecs.capability.task-iam-role"
    },
    {
      "targetId": null,
      "targetType": null,
      "value": null,
      "name": "com.amazonaws.ecs.capability.docker-remote-api.1.18"
    },
    {
      "targetId": null,
      "targetType": null,
      "value": null,
      "name": "ecs.capability.task-eni"
    }
  ],
  "pidMode": null,
  "requiresCompatibilities": [
    "FARGATE"
  ],
  "networkMode": "awsvpc",
  "cpu": "256",
  "revision": 1,
  "status": "ACTIVE",
  "inferenceAccelerators": null,
  "proxyConfiguration": null,
  "volumes": []
}
```
