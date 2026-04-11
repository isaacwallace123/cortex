# syntax=docker/dockerfile:1

# ---
FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY pkg/observe/go.mod pkg/observe/go.sum ./pkg/observe/
COPY services/nerva/go.mod services/nerva/go.sum ./services/nerva/
COPY services/courier/go.mod services/courier/go.sum ./services/courier/
COPY apps/relay/go.mod apps/relay/go.sum ./apps/relay/
RUN cd apps/relay && GOWORK=off go mod download

COPY pkg/observe ./pkg/observe
COPY services/nerva ./services/nerva
COPY services/courier ./services/courier
COPY apps/relay ./apps/relay

RUN cd apps/relay && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/relay \
    ./cmd/relay

# ---
# Alpine â€” relay executes shell commands on the host, needs sh.
FROM alpine:3.19

RUN addgroup -S cortex && adduser -S -G cortex cortex

RUN mkdir -p /data && chown cortex:cortex /data

COPY --from=builder /bin/relay /relay

USER cortex
ENTRYPOINT ["/relay"]

