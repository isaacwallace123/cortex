# syntax=docker/dockerfile:1

# ---
FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY pkg/observe/go.mod pkg/observe/go.sum ./pkg/observe/
COPY services/vault/go.mod services/vault/go.sum ./services/vault/
RUN cd services/vault && GOWORK=off go mod download

COPY pkg/observe ./pkg/observe
COPY services/vault ./services/vault

RUN cd services/vault && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/vault \
    ./cmd/vault

RUN mkdir /data && chown 65532:65532 /data

# ---
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/vault /vault
COPY --from=builder --chown=nonroot:nonroot /data /data

EXPOSE 8181

ENTRYPOINT ["/vault"]

