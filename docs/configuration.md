---
title: Configuring the Exporter
sort_rank: 3
---
# Configuring the Exporter

The exporter works with no configuration. Everything on this page is optional.

## Environment

| Variable | Meaning |
|---|---|
| `ECS_CONTAINER_METADATA_URI_V4` | Address of the ECS task metadata endpoint. ECS sets it in every container of the task, so you don't set it yourself. |

The exporter reads the variable at startup. If it is not set, the exporter exits with this error:

```
creating ECS metadata client: ECS_CONTAINER_METADATA_URI_V4 is not set; not running on ECS?
```

## Flags

Run `ecs_exporter --help` for the flags of the version you are running.

| Flag | Default | Meaning |
|---|---|---|
| `--web.listen-address` | `:9779` | Address to listen on. Repeat the flag to listen on several addresses. |
| `--web.config.file` | none | Path to a [web configuration file](https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md) that enables TLS or authentication. |
| `--web.telemetry-path` | `/metrics` | Path under which the metrics are served. |
| `--web.disable-exporter-metrics` | off | Leave out the `go_*`, `process_*` and `promhttp_*` metrics about the exporter itself. |
| `--web.max-requests` | `40` | Maximum number of scrapes served at once. Further requests get HTTP 503 until one finishes. `0` means no limit. |
| `--web.systemd-socket` | off | Use systemd socket activation instead of a port (Linux only). |
| `--log.level` | `info` | Lowest level that is logged: `debug`, `info`, `warn` or `error`. |
| `--log.format` | `logfmt` | Log format: `logfmt` or `json`. |
| `--version` | | Print the version and exit. |

In a task definition, flags go in the container's `command`, for example `"command": ["--web.disable-exporter-metrics"]`.

## HTTP endpoints

| Path | What it serves |
|---|---|
| `/metrics` | The metrics. The path changes with `--web.telemetry-path`. |
| `/` | A landing page with a link to the metrics. |
| `/health` | `ok` with HTTP 200 while the exporter is running. It does not check the task metadata endpoint. |

## TLS and authentication

To serve the endpoints over HTTPS or behind basic authentication, write a [web configuration file](https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md) and pass its path with `--web.config.file`. The file has to be in the container, for example from a volume or a custom image.

## Exporter metrics

Besides the [ECS metrics](metrics.md), `/metrics` carries:

- `ecs_exporter_build_info`, with the version, revision, branch and Go version of the build as labels.
- The standard Go runtime (`go_*`), process (`process_*`) and HTTP handler (`promhttp_*`) metrics.

`--web.disable-exporter-metrics` removes the `go_*`, `process_*` and `promhttp_*` metrics. `ecs_exporter_build_info` stays.

## Errors

If the exporter can't read `/task` or `/task/stats` from the task metadata endpoint, `/metrics` answers with HTTP 500 and the error, for example:

```
An error has occurred while serving metrics:

error collecting metric Desc{fqName: "ecs_task_metadata_info", [...]}: failed to retrieve task metadata: [...]
```

Prometheus then marks the scrape as failed, so `up` is `0` for that target.

These failures are only logged at the debug level. Run the exporter with `--log.level=debug` to see them in the container's logs, along with the full metadata and stats responses.
