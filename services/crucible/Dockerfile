# syntax=docker/dockerfile:1

# ---
FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY pkg/observe/go.mod pkg/observe/go.sum ./pkg/observe/
COPY services/crucible/go.mod services/crucible/go.sum ./services/crucible/
RUN cd services/crucible && GOWORK=off go mod download

COPY pkg/observe ./pkg/observe
COPY services/crucible ./services/crucible

RUN cd services/crucible && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/crucible \
    ./cmd/crucible

# ---
# Install nerdctl so the service can manage sandbox containers via the
# containerd socket mounted at runtime (/run/k3s/containerd/containerd.sock).
FROM alpine:3.21 AS nerdctl

ARG NERDCTL_VERSION=2.0.3
RUN wget -qO /tmp/nerdctl.tgz \
      "https://github.com/containerd/nerdctl/releases/download/v${NERDCTL_VERSION}/nerdctl-${NERDCTL_VERSION}-linux-amd64.tar.gz" \
    && tar -xzf /tmp/nerdctl.tgz -C /usr/local/bin nerdctl \
    && rm /tmp/nerdctl.tgz

# ---
FROM alpine:3.21

COPY --from=nerdctl /usr/local/bin/nerdctl /usr/local/bin/nerdctl
COPY --from=builder /bin/crucible /crucible

EXPOSE 7272

ENTRYPOINT ["/crucible"]
