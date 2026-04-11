# syntax=docker/dockerfile:1

# ---
FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY pkg/observe/go.mod pkg/observe/go.sum ./pkg/observe/
COPY services/forge/go.mod services/forge/go.sum ./services/forge/
RUN cd services/forge && GOWORK=off go mod download

COPY pkg/observe ./pkg/observe
COPY services/forge ./services/forge

RUN cd services/forge && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/forge \
    ./cmd/forge

# ---
# Alpine (not distroless) â€” Forge needs to write files and manage permissions.
FROM alpine:3.19

RUN addgroup -S cortex && adduser -S -G cortex cortex && \
    mkdir -p /workspace && chown cortex:cortex /workspace

COPY --from=builder /bin/forge /forge

EXPOSE 5050

USER cortex
ENTRYPOINT ["/forge"]

