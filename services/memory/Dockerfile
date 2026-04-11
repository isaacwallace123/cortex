# syntax=docker/dockerfile:1

# ---
FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY services/memory/go.mod services/memory/go.sum ./services/memory/
RUN cd services/memory && GOWORK=off go mod download

COPY services/memory ./services/memory

RUN cd services/memory && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/memory \
    ./cmd/memory

# Create /data with ownership for distroless nonroot uid (65532)
RUN mkdir /data && chown 65532:65532 /data

# ---
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/memory /memory
COPY --from=builder --chown=nonroot:nonroot /data /data

EXPOSE 7070

ENTRYPOINT ["/memory"]

