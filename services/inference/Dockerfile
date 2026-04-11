# syntax=docker/dockerfile:1

# ---
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Inference has no local replace directives â€” only its own module is needed.
# GOWORK=off: bypass go.work (which references sibling modules not present here).
COPY services/inference/go.mod services/inference/go.sum ./services/inference/
RUN cd services/inference && GOWORK=off go mod download

# gen/ is committed â€” no protoc needed at build time
COPY services/inference ./services/inference

RUN cd services/inference && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/inference \
    ./cmd/inference

# ---
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/inference /inference

EXPOSE 9090

ENTRYPOINT ["/inference"]

