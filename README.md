# nodeExporter-Adapter [中文](README-cn.md)

A data adapter located between Prometheus and Exporter, capable of adding custom labels to the data returned by the exporter. It has been tested to work not only with nodeExporter but also with any Exporter that complies with the Prometheus-Exporter rules.

## Supported Features

1. Access custom label files in JSON format.
2. Common command-line parameters, such as `--label-config` to specify the custom label configuration file, `--port` to specify the port number (default is 9001), and `--export-url` to specify the export URL.
3. Support for Chinese characters in value.
4. The custom label configuration file supports hot updates, which means you don't need to restart the program after updating.

## Important Notes

1. The value in the custom label configuration file must be a string.

## Installation and Startup

The adapter is written in GO and can be run directly after being packaged into a binary file.

### 1. Local Startup

>./node_exporter_adapter --label-config=/opt/custom-label/custom-label.json --port=8999

### 2. Kubernetes

It is recommended to run the adapter alongside the Exporter in the same Pod, especially for versions 1.29 and above, where you can consider using `SidecarContainers`.

**Note:** The simplest method is to change the adapter port to 9100 and the nodeExporter port to another value, but remember to change the `name` of the `ports` to `metrics`.

> This is a deployment method suitable for prometheus-operator.

```yaml
apiVersion: apps/v1
kind: DaemonSet
...
containers:
  - args:
      - --label-config=/opt/custom_label/custom_label.json
      - --port=9100
      - --export-url=127.0.0.1:9111/metrics
    image: myregistry/node_exporter_adapter:arm64
    imagePullPolicy: Always
    name: exporter-adapter
    ports:
      - containerPort: 9100
        hostPort: 9100
        name: metrics
        protocol: TCP
    volumeMounts:
      - mountPath: /opt/custom_label
        name: custom
        readOnly: true
  - args:
      - --path.procfs=/host/proc
      - --path.sysfs=/host/sys
      - --web.listen-address=0.0.0.0:9111
      - --collector.filesystem.ignored-fs-types=^(autofs|binfmt_misc|cgroup|configfs|debugfs|devpts|devtmpfs|fusectl|hugetlbfs|mqueue|overlay|proc|procfs|pstore|rpc_pipefs|securityfs|sysfs|tracefs)$
      - --collector.filesystem.ignored-mount-points=^/(dev|proc|sys|var/lib/docker/.+)($|/)
      - --collector.systemd.unit-whitelist=(docker|sshd|rsyslog|registry|kubelet|chronyd|glusterd|haproxy18|mysqld-3306|xinetd).service
      - --no-collector.thermal_zone
    image: myregistry/node-exporter:1.5.0-debian-11-r58
    imagePullPolicy: IfNotPresent
    livenessProbe:
      failureThreshold: 6
      httpGet:
        path: /
        port: metrics-exporter
        scheme: HTTP
      initialDelaySeconds: 120
      periodSeconds: 10
      successThreshold: 1
      timeoutSeconds: 5
    name: node-exporter
    ports:
      - containerPort: 9111
        hostPort: 9111
        name: metrics-exporter
        protocol: TCP
    readinessProbe:
      failureThreshold: 6
      httpGet:
        path: /
        port: metrics-exporter
        scheme: HTTP
      initialDelaySeconds: 30
      periodSeconds: 10
      successThreshold: 1
      timeoutSeconds: 5
...
volumes:
  - hostPath:
      path: /opt/custom_label
      type: ""
    name: custom
```