# syntax=docker/dockerfile:1

# ---
FROM golang:1.25-alpine AS builder

# modernc/sqlite uses CGO-free pure Go â€” no gcc needed.
WORKDIR /build

COPY pkg/observe/go.mod pkg/observe/go.sum ./pkg/observe/
COPY services/nerva/go.mod services/nerva/go.sum ./services/nerva/
RUN cd services/nerva && GOWORK=off go mod download

COPY pkg/observe ./pkg/observe
COPY services/nerva ./services/nerva

RUN cd services/nerva && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/nerva \
    ./cmd/nerva

# Create /data with ownership for distroless nonroot uid (65532)
RUN mkdir /data && chown 65532:65532 /data

# ---
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/nerva /nerva
COPY --from=builder --chown=nonroot:nonroot /data /data

EXPOSE 3030

ENTRYPOINT ["/nerva"]

