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
# docker:cli provides the docker CLI binary needed to manage sandboxes.
# The Docker socket is mounted at runtime via docker-compose volume.
FROM docker:cli

COPY --from=builder /bin/crucible /crucible

EXPOSE 7272

ENTRYPOINT ["/crucible"]

