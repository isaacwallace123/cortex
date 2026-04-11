# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY services/workspace/go.mod services/workspace/go.sum ./services/workspace/
RUN cd services/workspace && GOWORK=off go mod download

COPY services/workspace ./services/workspace

RUN cd services/workspace && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/workspace \
    ./cmd/workspace

RUN mkdir /data && chown 65532:65532 /data

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/workspace /workspace
COPY --from=builder --chown=nonroot:nonroot /data /data

EXPOSE 6262

ENTRYPOINT ["/workspace"]
