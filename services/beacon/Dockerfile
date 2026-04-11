# syntax=docker/dockerfile:1

# ---
FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY services/beacon/go.mod services/beacon/go.sum ./services/beacon/
RUN cd services/beacon && GOWORK=off go mod download

COPY services/beacon ./services/beacon

RUN cd services/beacon && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/beacon \
    ./cmd/beacon

# ---
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/beacon /beacon

EXPOSE 5252

ENTRYPOINT ["/beacon"]

