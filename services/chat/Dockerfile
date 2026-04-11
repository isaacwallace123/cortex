# syntax=docker/dockerfile:1

# ---
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Chat depends only on pkg/observe and modernc.org/sqlite.
COPY pkg/observe/go.mod pkg/observe/go.sum ./pkg/observe/
COPY services/chat/go.mod services/chat/go.sum ./services/chat/
RUN cd services/chat && GOWORK=off go mod download

COPY pkg/observe ./pkg/observe
COPY services/chat ./services/chat

RUN cd services/chat && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/chat \
    ./cmd/chat

RUN mkdir /data && chown 65532:65532 /data

# ---
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/chat /chat
COPY --from=builder --chown=nonroot:nonroot /data /data

EXPOSE 9191

ENTRYPOINT ["/chat"]

