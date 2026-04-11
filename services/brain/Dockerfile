# syntax=docker/dockerfile:1

# ---
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Brain depends on beacon, inference, memory, policy, shell, forge, nerva, courier, crucible, vault, arsenal, and observe.
# GOWORK=off: use go.mod replace directives, not go.work.
COPY pkg/observe/go.mod pkg/observe/go.sum ./pkg/observe/
COPY services/brain/go.mod services/brain/go.sum ./services/brain/
COPY services/beacon/go.mod services/beacon/go.sum ./services/beacon/
COPY services/arsenal/go.mod services/arsenal/go.sum ./services/arsenal/
COPY services/atlas/go.mod services/atlas/go.sum ./services/atlas/
COPY services/compass/go.mod services/compass/go.sum ./services/compass/
COPY services/courier/go.mod services/courier/go.sum ./services/courier/
COPY services/crucible/go.mod services/crucible/go.sum ./services/crucible/
COPY services/forge/go.mod services/forge/go.sum ./services/forge/
COPY services/inference/go.mod services/inference/go.sum ./services/inference/
COPY services/memory/go.mod services/memory/go.sum ./services/memory/
COPY services/nerva/go.mod services/nerva/go.sum ./services/nerva/
COPY services/policy/go.mod services/policy/go.sum ./services/policy/
COPY services/shell/go.mod services/shell/go.sum ./services/shell/
COPY services/vault/go.mod services/vault/go.sum ./services/vault/
RUN cd services/brain && GOWORK=off go mod download

COPY pkg/observe ./pkg/observe
COPY services/brain ./services/brain
COPY services/beacon ./services/beacon
COPY services/arsenal ./services/arsenal
COPY services/atlas ./services/atlas
COPY services/compass ./services/compass
COPY services/courier ./services/courier
COPY services/crucible ./services/crucible
COPY services/forge ./services/forge
COPY services/inference ./services/inference
COPY services/memory ./services/memory
COPY services/nerva ./services/nerva
COPY services/policy ./services/policy
COPY services/shell ./services/shell
COPY services/vault ./services/vault

RUN cd services/brain && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/brain \
    ./cmd/brain

# ---
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/brain /brain

EXPOSE 8080 8090

ENTRYPOINT ["/brain"]

