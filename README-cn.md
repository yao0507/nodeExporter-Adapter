# nodeExporter-Adapter

一个位于Prometheus与Exporter中间的数据适配器，可以往exporter返回的数据中添加自定义标签，经测试不仅仅适用于nodeExporter，所有的符合Prometheus-Exporter规则的Exporter都能添加。

## 支持的功能

1. 使用json格式访问自定义标签文件。
2. 常用的命令行参数，`--lable-config`指定自定义标签配置文件，`--port`指定端口号，默认9001，`--export-url`指定exportURL。
3. 支持中文的value。
4. 自定义标签配置文件支持热更新，每次更新后无需重启程序。

## 注意事项

1. 自定义标签配置文件中value要使用字符串。

## 安装及启动

使用GO编写打包成二进制文件后直接运行即可。

### 1. 本地启动

> ./node_exporter_adapter --label-config=/opt/custom-label/custom-label.json --port=8999

### 2. kubernetes

建议将Adapter放入ExporterPod中一并启动，1.29+可以考虑`SidecarContainers`启动。

**注意：最简单的方法就是将适配器端口改为9100，nodeExporter改为其他端口，但注意的是`ports`的`name`需要更改，适配器改为`metrics`**

> 适用于prometheus-operator的部署方式

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




