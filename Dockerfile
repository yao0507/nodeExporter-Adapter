FROM alpine:latest
MAINTAINER yao0507
COPY ./node_exporter_adapter /opt/node_exporter_adapter
ENTRYPOINT ["/opt/node_exporter_adapter"]