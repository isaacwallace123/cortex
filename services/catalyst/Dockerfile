# syntax=docker/dockerfile:1

# ---
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Download deps before copying source so layer is cached.
COPY pkg/observe/go.mod pkg/observe/go.sum* ./pkg/observe/
COPY services/brain/go.mod services/brain/go.sum* ./services/brain/
COPY services/catalyst/go.mod services/catalyst/go.sum* ./services/catalyst/
RUN cd services/catalyst && GOWORK=off go mod download

COPY pkg/observe ./pkg/observe
COPY services/brain ./services/brain
COPY services/catalyst ./services/catalyst

RUN cd services/catalyst && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/catalyst \
    ./cmd/catalyst

RUN mkdir /data && chown 65532:65532 /data

# ---
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/catalyst /catalyst
COPY --from=builder --chown=nonroot:nonroot /data /data

EXPOSE 5353

ENTRYPOINT ["/catalyst"]
