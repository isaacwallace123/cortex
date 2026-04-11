# syntax=docker/dockerfile:1

# ---
FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY pkg/observe/go.mod pkg/observe/go.sum ./pkg/observe/
COPY services/arsenal/go.mod services/arsenal/go.sum ./services/arsenal/
RUN cd services/arsenal && GOWORK=off go mod download

COPY pkg/observe ./pkg/observe
COPY services/arsenal ./services/arsenal

RUN cd services/arsenal && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/arsenal \
    ./cmd/arsenal

RUN mkdir /data && chown 65532:65532 /data

# ---
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/arsenal /arsenal
COPY --from=builder --chown=nonroot:nonroot /data /data

EXPOSE 7373

ENTRYPOINT ["/arsenal"]

