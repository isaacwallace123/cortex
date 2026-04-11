# syntax=docker/dockerfile:1

# ---
FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY pkg/observe/go.mod pkg/observe/go.sum ./pkg/observe/
COPY services/policy/go.mod services/policy/go.sum ./services/policy/
RUN cd services/policy && GOWORK=off go mod download

COPY pkg/observe ./pkg/observe
COPY services/policy ./services/policy

RUN cd services/policy && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/policy \
    ./cmd/policy

RUN mkdir /data && chown 65532:65532 /data

# ---
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/policy /policy
COPY --from=builder --chown=nonroot:nonroot /data /data

EXPOSE 6060

ENTRYPOINT ["/policy"]

