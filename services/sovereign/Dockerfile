# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY services/sovereign/go.mod services/sovereign/go.sum ./services/sovereign/
RUN cd services/sovereign && GOWORK=off go mod download

COPY services/sovereign ./services/sovereign

RUN cd services/sovereign && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/sovereign \
    ./cmd/sovereign

RUN mkdir /data && chown 65532:65532 /data

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/sovereign /sovereign
COPY --from=builder --chown=nonroot:nonroot /data /data

EXPOSE 4141

ENTRYPOINT ["/sovereign"]
