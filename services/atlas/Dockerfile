# syntax=docker/dockerfile:1

# ---
FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY pkg/observe/go.mod pkg/observe/go.sum ./pkg/observe/
COPY services/atlas/go.mod services/atlas/go.sum ./services/atlas/
RUN cd services/atlas && GOWORK=off go mod download

COPY pkg/observe ./pkg/observe
COPY services/atlas ./services/atlas

RUN cd services/atlas && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/atlas \
    ./cmd/atlas

# ---
# bitnami/kubectl provides a minimal image with kubectl pre-installed.
FROM bitnami/kubectl:latest AS kubectl
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=kubectl /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/kubectl
COPY --from=builder /bin/atlas /atlas

EXPOSE 6161

ENTRYPOINT ["/atlas"]

