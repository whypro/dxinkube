FROM alpine:3.5

WORKDIR /dxinkube

COPY zk-controller /dxinkube/bin/

# Set timezone to Asia/Shanghai.
# Add Common CA certificates PEM files.
RUN set -x \
    && apk add --no-cache ca-certificates \
    && apk add --no-cache tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && apk del tzdata

ENTRYPOINT ["/dxinkube/bin/zk-controller"]
