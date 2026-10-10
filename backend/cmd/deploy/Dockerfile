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
# Build toolchain is pinned and checksum-verified (no curl | bash). Bump the
# versions and sha256s together from each project's GitHub release assets.
ARG RAILPACK_VERSION=0.40.1
ARG RAILPACK_SHA256_AMD64=2842de93e68713af9037e0bc0a398d7da78f3b96aa4804303a638db2bc69bd30
ARG RAILPACK_SHA256_ARM64=c24a064b586b8f4f8c2fab44dd5ef19253e4c6cc4e1df793b3ae19cd87f7a5d4
ARG BUILDKIT_VERSION=0.23.2
ARG BUILDKIT_SHA256_AMD64=2771c3403e3a1f75a83cde387a05365794d3b900c355e864772a36c3ce541f82
ARG BUILDKIT_SHA256_ARM64=6385ff70b2fb4134b50ac3183eea3a0b06c6f6129173940d73178ae0477368f1
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates curl docker.io bash && rm -rf /var/lib/apt/lists/* \
    && groupadd -f docker && useradd -u 65532 -m -G docker nonroot \
    && ARCH="$(dpkg --print-architecture)" \
    && case "$ARCH" in \
         amd64) RP_ARCH="x86_64"; RP_SHA="$RAILPACK_SHA256_AMD64"; BK_SHA="$BUILDKIT_SHA256_AMD64";; \
         arm64) RP_ARCH="arm64"; RP_SHA="$RAILPACK_SHA256_ARM64"; BK_SHA="$BUILDKIT_SHA256_ARM64";; \
         *) echo "unsupported architecture: $ARCH" >&2; exit 1;; \
       esac \
    && curl -fsSL -o /tmp/railpack.tar.gz "https://github.com/railwayapp/railpack/releases/download/v${RAILPACK_VERSION}/railpack-v${RAILPACK_VERSION}-${RP_ARCH}-unknown-linux-musl.tar.gz" \
    && echo "${RP_SHA}  /tmp/railpack.tar.gz" | sha256sum -c - \
    && tar -xzf /tmp/railpack.tar.gz -C /usr/local/bin railpack \
    && curl -fsSL -o /tmp/buildkit.tar.gz "https://github.com/moby/buildkit/releases/download/v${BUILDKIT_VERSION}/buildkit-v${BUILDKIT_VERSION}.linux-${ARCH}.tar.gz" \
    && echo "${BK_SHA}  /tmp/buildkit.tar.gz" | sha256sum -c - \
    && tar -xzf /tmp/buildkit.tar.gz -C /usr/local bin/buildctl \
    && ln -sf /usr/local/bin/buildctl /usr/bin/buildctl \
    && rm -f /tmp/railpack.tar.gz /tmp/buildkit.tar.gz \
    && railpack --version && buildctl --version
COPY --from=builder /bin/deploy /bin/deploy
USER nonroot

EXPOSE 8080

CMD ["/bin/deploy"]
