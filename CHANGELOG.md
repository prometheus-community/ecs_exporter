## 0.4.0 / 2025-02-XY

* [ENHANCEMENT] Add a counter for container restarts #87
* [ENHANCEMENT] Add a flag to disable standard client_golang exporter metrics
  #82
* [CHANGE] Overhaul all metrics. Many metric names, labels, and semantics have
  changed to improve correctness. See the [README](./README.md#example-output)
  for current /metrics output. #81

## 0.3.0 / 2024-10-13

* [CHANGE] Use upstream ecs-agent types for deserializing API responses #75
* [CHANGE] Update exporter boilerplate #77
* [ENHANCEMENT] Add additional metrics #53

## 0.2.1 / 2023-01-24

* [BUGFIX] Fix cpu and memory stats #49

## 0.2.0 / 2022-08-02

* [BUGFIX] Fix CPU usage metrics #37

## 0.1.1 / 2022-02-03

* [FEATURE] Expose memory cache metric #25
* [ENHANCEMENT] Allow ecsmetadata to work outside of ECS #18

## 0.1.0 / 2021-09-21

* [FEATURE] Initial release.
