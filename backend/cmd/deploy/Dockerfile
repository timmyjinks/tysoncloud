# syntax=docker/dockerfile:1.7
FROM golang:1.26 AS builder
RUN apt-get update && apt-get install -y --no-install-recommends git && rm -rf /var/lib/apt/lists/*
WORKDIR /src

# Leverage BuildKit cache mounts for Go modules + build cache
# This keeps /go/pkg/mod and /root/.cache/go-build across builds
COPY --link go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY --link . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-w -s" -o /bin/deploy ./cmd/deploy

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates curl docker.io bash && rm -rf /var/lib/apt/lists/* \
    && groupadd -f docker && useradd -u 65532 -m -G docker nonroot \
    && curl -fsSL https://railpack.com/install.sh | bash -s -- --yes || echo "railpack install failed, continuing with docker fallback"
COPY --from=builder /bin/deploy /bin/deploy
USER nonroot

EXPOSE 8080

CMD ["/bin/deploy"]
