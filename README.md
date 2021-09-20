# ecs_exporter

🚧 🚧 🚧 This repo is still work in progress and is subject to change.

This repo contains a Prometheus exporter for
Amazon Elastic Container Service (ECS) that publishes
ECS task infra metrics in Prometheus format.

You need to run it as a sidecar on your ECS tasks.

By default, it publishes Prometheus metrics on ":9779/metrics". The exporter in this repo can be a useful complementary sidecar for the scenario described in [this blog post](https://aws.amazon.com/blogs/opensource/metrics-collection-from-amazon-ecs-using-amazon-managed-service-for-prometheus/). Adding this sidecar to the ECS task definition would export task-level metrics in addition to the custom metrics described in the blog.  

The sidecar process is also supported on [AWS App Runner](https://aws.amazon.com/apprunner/)
and can be used to publish infra metrics in Prometheus format
from App Runner services.

## Labels

* **container**: Container associated with a metric.
* **cpu**: Available to CPU metrics, helps to breakdown metrics by CPU.
* **device**: Network interface device associated with the metric. Only
  available for several network metrics.

## Example output

```
# HELP ecs_cpu_seconds_total Total CPU usage in seconds.
# TYPE ecs_cpu_seconds_total counter
ecs_cpu_seconds_total{container="ecs-metadata-proxy",cpu="0"} 1.746774278e+08
ecs_cpu_seconds_total{container="ecs-metadata-proxy",cpu="1"} 1.7417992266e+08
# HELP ecs_memory_bytes Memory usage in bytes.
# TYPE ecs_memory_bytes gauge
ecs_memory_bytes{container="ecs-metadata-proxy"} 4.440064e+06
# HELP ecs_memory_limit_bytes Memory limit in bytes.
# TYPE ecs_memory_limit_bytes gauge
ecs_memory_limit_bytes{container="ecs-metadata-proxy"} 9.223372036854772e+18
# HELP ecs_memory_max_bytes Maximum memory usage in bytes.
# TYPE ecs_memory_max_bytes gauge
ecs_memory_max_bytes{container="ecs-metadata-proxy"} 9.023488e+06
# HELP ecs_network_receive_bytes_total Network recieved in bytes.
# TYPE ecs_network_receive_bytes_total counter
ecs_network_receive_bytes_total{container="ecs-metadata-proxy",device="eth1"} 4.2851757e+07
# HELP ecs_network_receive_dropped_total Network packets dropped in recieving.
# TYPE ecs_network_receive_dropped_total counter
ecs_network_receive_dropped_total{container="ecs-metadata-proxy",device="eth1"} 0
# HELP ecs_network_receive_errors_total Network errors in recieving.
# TYPE ecs_network_receive_errors_total counter
ecs_network_receive_errors_total{container="ecs-metadata-proxy",device="eth1"} 0
# HELP ecs_network_receive_packets_total Network packets recieved.
# TYPE ecs_network_receive_packets_total counter
ecs_network_receive_packets_total{container="ecs-metadata-proxy",device="eth1"} 516239
# HELP ecs_network_transmit_bytes_total Network transmitted in bytes.
# TYPE ecs_network_transmit_bytes_total counter
ecs_network_transmit_bytes_total{container="ecs-metadata-proxy",device="eth1"} 1.28412758e+08
# HELP ecs_network_transmit_dropped_total Network packets dropped in transmit.
# TYPE ecs_network_transmit_dropped_total counter
ecs_network_transmit_dropped_total{container="ecs-metadata-proxy",device="eth1"} 0
# HELP ecs_network_transmit_errors_total Network errors in transmit.
# TYPE ecs_network_transmit_errors_total counter
ecs_network_transmit_errors_total{container="ecs-metadata-proxy",device="eth1"} 0
# HELP ecs_network_transmit_packets_total Network packets transmitted.
# TYPE ecs_network_transmit_packets_total counter
ecs_network_transmit_packets_total{container="ecs-metadata-proxy",device="eth1"} 429472
# HELP go_gc_duration_seconds A summary of the pause duration of garbage collection cycles.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 0
go_gc_duration_seconds{quantile="0.25"} 0
go_gc_duration_seconds{quantile="0.5"} 0
go_gc_duration_seconds{quantile="0.75"} 0
go_gc_duration_seconds{quantile="1"} 0
go_gc_duration_seconds_sum 0
go_gc_duration_seconds_count 0
# HELP go_goroutines Number of goroutines that currently exist.
# TYPE go_goroutines gauge
go_goroutines 8
# HELP go_info Information about the Go environment.
# TYPE go_info gauge
go_info{version="go1.16.3"} 1
# HELP go_memstats_alloc_bytes Number of bytes allocated and still in use.
# TYPE go_memstats_alloc_bytes gauge
go_memstats_alloc_bytes 595760
# HELP go_memstats_alloc_bytes_total Total number of bytes allocated, even if freed.
# TYPE go_memstats_alloc_bytes_total counter
go_memstats_alloc_bytes_total 595760
# HELP go_memstats_buck_hash_sys_bytes Number of bytes used by the profiling bucket hash table.
# TYPE go_memstats_buck_hash_sys_bytes gauge
go_memstats_buck_hash_sys_bytes 4092
# HELP go_memstats_frees_total Total number of frees.
# TYPE go_memstats_frees_total counter
go_memstats_frees_total 123
# HELP go_memstats_gc_cpu_fraction The fraction of this program's available CPU time used by the GC since the program started.
# TYPE go_memstats_gc_cpu_fraction gauge
go_memstats_gc_cpu_fraction 0
# HELP go_memstats_gc_sys_bytes Number of bytes used for garbage collection system metadata.
# TYPE go_memstats_gc_sys_bytes gauge
go_memstats_gc_sys_bytes 3.97448e+06
# HELP go_memstats_heap_alloc_bytes Number of heap bytes allocated and still in use.
# TYPE go_memstats_heap_alloc_bytes gauge
go_memstats_heap_alloc_bytes 595760
# HELP go_memstats_heap_idle_bytes Number of heap bytes waiting to be used.
# TYPE go_memstats_heap_idle_bytes gauge
go_memstats_heap_idle_bytes 6.508544e+07
# HELP go_memstats_heap_inuse_bytes Number of heap bytes that are in use.
# TYPE go_memstats_heap_inuse_bytes gauge
go_memstats_heap_inuse_bytes 1.59744e+06
# HELP go_memstats_heap_objects Number of allocated objects.
# TYPE go_memstats_heap_objects gauge
go_memstats_heap_objects 2439
# HELP go_memstats_heap_released_bytes Number of heap bytes released to OS.
# TYPE go_memstats_heap_released_bytes gauge
go_memstats_heap_released_bytes 6.508544e+07
# HELP go_memstats_heap_sys_bytes Number of heap bytes obtained from system.
# TYPE go_memstats_heap_sys_bytes gauge
go_memstats_heap_sys_bytes 6.668288e+07
# HELP go_memstats_last_gc_time_seconds Number of seconds since 1970 of last garbage collection.
# TYPE go_memstats_last_gc_time_seconds gauge
go_memstats_last_gc_time_seconds 0
# HELP go_memstats_lookups_total Total number of pointer lookups.
# TYPE go_memstats_lookups_total counter
go_memstats_lookups_total 0
# HELP go_memstats_mallocs_total Total number of mallocs.
# TYPE go_memstats_mallocs_total counter
go_memstats_mallocs_total 2562
# HELP go_memstats_mcache_inuse_bytes Number of bytes in use by mcache structures.
# TYPE go_memstats_mcache_inuse_bytes gauge
go_memstats_mcache_inuse_bytes 9600
# HELP go_memstats_mcache_sys_bytes Number of bytes used for mcache structures obtained from system.
# TYPE go_memstats_mcache_sys_bytes gauge
go_memstats_mcache_sys_bytes 16384
# HELP go_memstats_mspan_inuse_bytes Number of bytes in use by mspan structures.
# TYPE go_memstats_mspan_inuse_bytes gauge
go_memstats_mspan_inuse_bytes 37400
# HELP go_memstats_mspan_sys_bytes Number of bytes used for mspan structures obtained from system.
# TYPE go_memstats_mspan_sys_bytes gauge
go_memstats_mspan_sys_bytes 49152
# HELP go_memstats_next_gc_bytes Number of heap bytes when next garbage collection will take place.
# TYPE go_memstats_next_gc_bytes gauge
go_memstats_next_gc_bytes 4.473924e+06
# HELP go_memstats_other_sys_bytes Number of bytes used for other system allocations.
# TYPE go_memstats_other_sys_bytes gauge
go_memstats_other_sys_bytes 497348
# HELP go_memstats_stack_inuse_bytes Number of bytes in use by the stack allocator.
# TYPE go_memstats_stack_inuse_bytes gauge
go_memstats_stack_inuse_bytes 425984
# HELP go_memstats_stack_sys_bytes Number of bytes obtained from system for stack allocator.
# TYPE go_memstats_stack_sys_bytes gauge
go_memstats_stack_sys_bytes 425984
# HELP go_memstats_sys_bytes Number of bytes obtained from system.
# TYPE go_memstats_sys_bytes gauge
go_memstats_sys_bytes 7.165032e+07
# HELP go_threads Number of OS threads created.
# TYPE go_threads gauge
go_threads 7
# HELP promhttp_metric_handler_requests_in_flight Current number of scrapes being served.
# TYPE promhttp_metric_handler_requests_in_flight gauge
promhttp_metric_handler_requests_in_flight 1
# HELP promhttp_metric_handler_requests_total Total number of scrapes by HTTP status code.
# TYPE promhttp_metric_handler_requests_total counter
promhttp_metric_handler_requests_total{code="200"} 0
promhttp_metric_handler_requests_total{code="500"} 0
promhttp_metric_handler_requests_total{code="503"} 0
```
