# syntax=docker/dockerfile:1

# ---
FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY pkg/observe/go.mod pkg/observe/go.sum ./pkg/observe/
COPY services/shell/go.mod services/shell/go.sum ./services/shell/
RUN cd services/shell && GOWORK=off go mod download

COPY pkg/observe ./pkg/observe
COPY services/shell ./services/shell

RUN cd services/shell && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/shell \
    ./cmd/shell

# ---
# Alpine (not distroless) â€” Shell needs /bin/sh to execute commands.
FROM alpine:3.19

RUN addgroup -S cortex && adduser -S -G cortex cortex

COPY --from=builder /bin/shell /shell

EXPOSE 4040

USER cortex
ENTRYPOINT ["/shell"]

