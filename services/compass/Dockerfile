# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY pkg/observe/go.mod pkg/observe/go.sum ./pkg/observe/
COPY services/compass/go.mod services/compass/go.sum ./services/compass/
RUN cd services/compass && GOWORK=off go mod download

COPY pkg/observe ./pkg/observe
COPY services/compass ./services/compass

RUN cd services/compass && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/compass \
    ./cmd/compass

RUN mkdir /data && chown 65532:65532 /data

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/compass /compass
COPY --from=builder --chown=nonroot:nonroot /data /data

EXPOSE 5151

ENTRYPOINT ["/compass"]
