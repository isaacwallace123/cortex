# syntax=docker/dockerfile:1

# ---
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Download deps before copying source so layer is cached.
COPY pkg/observe/go.mod pkg/observe/go.sum* ./pkg/observe/
COPY services/arsenal/go.mod services/arsenal/go.sum* ./services/arsenal/
COPY services/inference/go.mod services/inference/go.sum* ./services/inference/
COPY services/genesis/go.mod services/genesis/go.sum* ./services/genesis/
RUN cd services/genesis && GOWORK=off go mod download

COPY pkg/observe ./pkg/observe
COPY services/arsenal ./services/arsenal
COPY services/inference ./services/inference
COPY services/genesis ./services/genesis

RUN cd services/genesis && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/genesis \
    ./cmd/genesis

RUN mkdir /data && chown 65532:65532 /data

# ---
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/genesis /genesis
COPY --from=builder --chown=nonroot:nonroot /data /data

EXPOSE 4343

ENTRYPOINT ["/genesis"]
