ARG OS="linux"
ARG ARCH="amd64"

FROM golang:1.18 as builder

WORKDIR /app

ENV CGO_ENABLED=0
ENV GOOS=${OS}
ENV GOARCH=${ARCH}

COPY . .
RUN make build

# second step to build minimal image
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest

COPY --from=builder /app/bin/peggo-exporter /go/bin/peggo-exporter

EXPOSE     5566
USER       nobody
ENTRYPOINT [ "/go/bin/peggo-exporter" ]
