---
title: Metrics
sort_rank: 4
---
# Metrics

All metrics come from the [ECS task metadata endpoint version 4](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4.html). The descriptions below are the metrics' own help texts.

For real output from tasks on EC2, Fargate and Managed Instances, see the [metrics snapshots](https://github.com/prometheus-community/ecs_exporter/tree/main/ecscollector/testdata/snapshots) in the repository. They are generated from recorded metadata responses and leave out the exporter's own metrics (see [Exporter metrics](configuration.md#exporter-metrics)).

## Labels

- **Task metrics** have no labels, except `ecs_task_metadata_info`.
- **Container metrics** have `container_name`, the container's name in the task definition.
- **Network metrics** have `interface`, the network interface the numbers belong to.

`ecs_task_metadata_info` always has the value 1 and carries the task's metadata as labels: `cluster`, `task_arn`, `family`, `revision`, `desired_status`, `known_status`, `availability_zone` and `launch_type`. Join it to any other metric from the same exporter to add those labels. For example, memory use per container with the task's family and ARN:

```promql
ecs_container_memory_usage_bytes
  * on (job, instance) group_left (family, task_arn)
  ecs_task_metadata_info
```

## Task metrics

| Metric | Type | Description |
|---|---|---|
| `ecs_task_metadata_info` | gauge | ECS task metadata, sourced from the task metadata endpoint version 4. |
| `ecs_task_cpu_limit_vcpus` | gauge | Configured task CPU limit in vCPUs (1 vCPU = 1024 CPU units). This is optional when running on EC2 or Managed Instances; if no limit is set, this metric has no value. |
| `ecs_task_memory_limit_bytes` | gauge | Configured task memory limit in bytes. This is optional when running on EC2 or Managed Instances; if no limit is set, this metric has no value. |
| `ecs_task_ephemeral_storage_used_bytes` | gauge | Current Fargate task ephemeral storage usage in bytes. |
| `ecs_task_ephemeral_storage_allocated_bytes` | gauge | Configured Fargate task ephemeral storage allocated size in bytes. |
| `ecs_task_image_pull_start_timestamp_seconds` | gauge | The time at which the task started pulling docker images for its containers. |
| `ecs_task_image_pull_stop_timestamp_seconds` | gauge | The time at which the task stopped (i.e. completed) pulling docker images for its containers. |
| `ecs_task_clock_error_bound_seconds` | gauge | Upper bound on the task clock error relative to UTC, as reported by Amazon Time Sync. This is a bound, not a measured offset. Absent when metadata omits ClockDrift. |
| `ecs_task_clock_reference_timestamp_seconds` | gauge | Unix time of the reference measurement behind the reported clock error bound. Absent when metadata omits ClockDrift. |
| `ecs_task_clock_synchronized` | gauge | 1 if the most recent synchronization of the task clock with the reference time succeeded, otherwise 0. Absent when metadata omits ClockDrift. |

- The CPU and memory limits are required on Fargate and optional on EC2 and Managed Instances. Without a limit, the metric is absent.
- The ephemeral storage metrics are only present when the task metadata reports ephemeral storage, which Fargate does.
- The image pull timestamps are only present when the task metadata has them.

### Task clock metrics

Fargate and ECS Managed Instances report Amazon Time Sync clock drift in the task metadata `ClockDrift` block. EC2 leaves that block out, so the clock metrics are absent on EC2 rather than reported as zero. The error bound is an [upper bound](https://aws.amazon.com/blogs/containers/run-time-sensitive-workloads-on-ecs-fargate-with-clock-accuracy-tracking/) on clock error, not a measured offset.

## Container metrics

| Metric | Type | Description |
|---|---|---|
| `ecs_container_restarts_total` | counter | Cumulative total count of container restarts. Only has a value if the container has been configured to restart on failure. |
| `ecs_container_cpu_usage_seconds_total` | counter | Cumulative total container CPU usage in seconds. |
| `ecs_container_cpu_user_seconds_total` | counter | Cumulative container CPU time spent in user mode in seconds. With cgroup v1, this is sampled at clock tick granularity, so user and kernel time may not add up to total CPU usage. |
| `ecs_container_cpu_kernel_seconds_total` | counter | Cumulative container CPU time spent in kernel mode in seconds. With cgroup v1, this is sampled at clock tick granularity, so user and kernel time may not add up to total CPU usage. |
| `ecs_container_memory_usage_bytes` | gauge | Current container memory usage reported by the runtime in bytes, including page cache and accounted kernel memory. |
| `ecs_container_memory_limit_bytes` | gauge | Configured memory limit applicable to the container in bytes, using the positive container setting when present, otherwise the task setting shared by the task's containers. |
| `ecs_container_memory_page_cache_size_bytes` | gauge | Current page cache memory accounted to the container in bytes, including tmpfs and shared memory. |
| `ecs_container_memory_anonymous_bytes` | gauge | Current anonymous memory charged to the container cgroup in bytes, excluding file-backed memory. |
| `ecs_container_memory_anonymous_thp_bytes` | gauge | Current anonymous memory backed by transparent huge pages in bytes. |
| `ecs_container_memory_mapped_file_bytes` | gauge | Current mapped file memory accounted to the container in bytes, including mapped tmpfs and shared memory. |
| `ecs_container_memory_active_file_bytes` | gauge | Current memory on the container cgroup's active file LRU list in bytes. |
| `ecs_container_memory_inactive_file_bytes` | gauge | Current memory on the container cgroup's inactive file LRU list in bytes. |
| `ecs_container_memory_dirty_bytes` | gauge | Current dirty memory accounted to the container awaiting writeback in bytes. |
| `ecs_container_memory_writeback_bytes` | gauge | Current memory accounted to the container under writeback in bytes. |
| `ecs_container_memory_unevictable_bytes` | gauge | Current memory on the container cgroup's unevictable LRU list in bytes. |
| `ecs_container_memory_shared_bytes` | gauge | Current tmpfs, shared memory, and shared anonymous memory accounted to the container in bytes. Only available with cgroup v2. |
| `ecs_container_memory_kernel_stack_bytes` | gauge | Current memory allocated to the container's kernel stacks in bytes. Only available with cgroup v2. |
| `ecs_container_memory_socket_bytes` | gauge | Current network transmission buffer memory accounted to the container in bytes. Only available with cgroup v2. |
| `ecs_container_memory_slab_reclaimable_bytes` | gauge | Current reclaimable kernel slab memory accounted to the container in bytes. Only available with cgroup v2. |
| `ecs_container_memory_slab_unreclaimable_bytes` | gauge | Current unreclaimable kernel slab memory accounted to the container in bytes. Only available with cgroup v2. |
| `ecs_container_memory_max_usage_bytes` | gauge | Maximum container memory usage recorded by the runtime in bytes. |
| `ecs_container_memory_page_faults_total` | counter | Cumulative number of container cgroup page faults, including major faults. |
| `ecs_container_memory_major_page_faults_total` | counter | Cumulative number of container cgroup major page faults. |
| `ecs_container_memory_pages_scanned_total` | counter | Cumulative number of container cgroup pages scanned from the inactive LRU list. Only available with cgroup v2. |
| `ecs_container_memory_pages_reclaimed_total` | counter | Cumulative number of container cgroup pages reclaimed. Only available with cgroup v2. |
| `ecs_container_memory_limit_failures_total` | counter | Cumulative number of times a container cgroup memory charge encountered its limit. This is not an OOM kill count and is only available with cgroup v1. |

A container without stats, for example a stopped non-essential container, has no container metrics.

### Container memory metrics

The task stats endpoint exposes Docker-compatible Linux cgroup memory accounting. See the kernel documentation for the [cgroup v1 memory statistics](https://docs.kernel.org/admin-guide/cgroup-v1/memory.html#stat-file) and the [cgroup v2 memory interface](https://docs.kernel.org/admin-guide/cgroup-v2.html#memory-interface-files), and Docker's explanation of the difference between [raw API usage and the working-set-like value shown by `docker stats`](https://docs.docker.com/reference/cli/docker/container/stats/#description).

- `ecs_container_memory_usage_bytes` is the raw usage. It includes page cache and accounted kernel memory.
- The breakdown metrics overlap and do not sum to a meaningful number.
- Some breakdown metrics exist with only one cgroup version. Metrics that the cgroup version doesn't support are absent rather than reported as zero. Currently, Fargate always uses cgroup v1, Managed Instances uses cgroup v2, and EC2 depends on the instance's configuration. For example, Amazon Linux 2 uses v1 and Amazon Linux 2023 uses v2.
- `ecs_container_memory_max_usage_bytes` is only present when the runtime reports a peak usage.
- `ecs_container_memory_limit_failures_total` counts memory charges that hit the limit, not OOM kills. Its rate shows pressure on the limit, but a nonzero value does not mean an allocation or a container failed.
- `ecs_container_memory_limit_bytes` comes from the ECS configuration. If the container has no memory setting of its own, it is the task's limit. That value is shared by all the task's containers, so don't add it up across containers as if each had its own. With no container or task memory setting, the metric is absent.

## Network metrics

| Metric | Type | Description |
|---|---|---|
| `ecs_network_receive_bytes_total` | counter | Cumulative total size of network packets received in bytes. |
| `ecs_network_receive_packets_total` | counter | Cumulative total count of network packets received. |
| `ecs_network_receive_packets_dropped_total` | counter | Cumulative total count of network packets dropped in receiving. |
| `ecs_network_receive_errors_total` | counter | Cumulative total count of network errors in receiving. |
| `ecs_network_transmit_bytes_total` | counter | Cumulative total size of network packets transmitted in bytes. |
| `ecs_network_transmit_packets_total` | counter | Cumulative total count of network packets transmitted. |
| `ecs_network_transmit_packets_dropped_total` | counter | Cumulative total count of network packets dropped in transmit. |
| `ecs_network_transmit_errors_total` | counter | Cumulative total count of network errors in transmit. |

The stats endpoint reports network numbers per container, but containers that share a network see the same interfaces. The exporter reports each interface once.
