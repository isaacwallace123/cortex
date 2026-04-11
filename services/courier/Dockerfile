# syntax=docker/dockerfile:1

# ---
FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY pkg/observe/go.mod pkg/observe/go.sum ./pkg/observe/
COPY services/nerva/go.mod services/nerva/go.sum ./services/nerva/
COPY services/courier/go.mod services/courier/go.sum ./services/courier/
RUN cd services/courier && GOWORK=off go mod download

COPY pkg/observe ./pkg/observe
COPY services/nerva ./services/nerva
COPY services/courier ./services/courier

RUN cd services/courier && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/courier \
    ./cmd/courier

# ---
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/courier /courier

EXPOSE 2020

ENTRYPOINT ["/courier"]

