# syntax=docker/dockerfile:1

# ---
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Overwatch depends on nerva (for gen/nerva/v1 proto bindings).
COPY services/nerva/go.mod services/nerva/go.sum ./services/nerva/
COPY services/overwatch/go.mod services/overwatch/go.sum ./services/overwatch/
RUN cd services/overwatch && GOWORK=off go mod download

COPY services/nerva ./services/nerva
COPY services/overwatch ./services/overwatch

RUN cd services/overwatch && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/overwatch \
    ./cmd/overwatch

# ---
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/overwatch /overwatch

EXPOSE 8484

ENTRYPOINT ["/overwatch"]
