ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

ARG ARCH="amd64"
ARG OS="linux"
COPY .build/${OS}-${ARCH}/ecs_exporter /bin/ecs_exporter

EXPOSE     9779
USER       nobody
ENTRYPOINT [ "/bin/ecs_exporter" ]
