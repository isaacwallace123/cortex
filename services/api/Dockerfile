# syntax=docker/dockerfile:1

# ---
FROM golang:1.25-alpine AS builder

WORKDIR /build

# API's go.mod has: replace github.com/cortexhq/cortex/services/brain => ../brain
# Brain's go.mod has: replace .../inference => ../inference
# All three modules must be present. GOWORK=off: use go.mod replace directives.
COPY pkg/observe/go.mod pkg/observe/go.sum ./pkg/observe/
COPY services/api/go.mod services/api/go.sum ./services/api/
COPY services/arsenal/go.mod services/arsenal/go.sum ./services/arsenal/
COPY services/atlas/go.mod services/atlas/go.sum ./services/atlas/
COPY services/beacon/go.mod services/beacon/go.sum ./services/beacon/
COPY services/brain/go.mod services/brain/go.sum ./services/brain/
COPY services/chat/go.mod services/chat/go.sum ./services/chat/
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
COPY services/sovereign/go.mod services/sovereign/go.sum ./services/sovereign/
COPY services/workspace/go.mod services/workspace/go.sum ./services/workspace/
RUN cd services/api && GOWORK=off go mod download

COPY pkg/observe ./pkg/observe
COPY services/api ./services/api
COPY services/arsenal ./services/arsenal
COPY services/atlas ./services/atlas
COPY services/beacon ./services/beacon
COPY services/brain ./services/brain
COPY services/chat ./services/chat
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
COPY services/sovereign ./services/sovereign
COPY services/workspace ./services/workspace

RUN cd services/api && CGO_ENABLED=0 GOOS=linux GOWORK=off go build \
    -ldflags="-s -w" \
    -o /bin/api \
    ./cmd/api

# ---
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/api /api

EXPOSE 8000

ENTRYPOINT ["/api"]

