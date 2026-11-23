FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/deploy ./cmd/deploy

FROM debian:bookworm-slim AS tools
ARG RAILPACK_VERSION=0.40.1
ARG RAILPACK_SHA256=2842de93e68713af9037e0bc0a398d7da78f3b96aa4804303a638db2bc69bd30
ADD --checksum=sha256:${RAILPACK_SHA256} \
    https://github.com/railwayapp/railpack/releases/download/v${RAILPACK_VERSION}/railpack-v${RAILPACK_VERSION}-x86_64-unknown-linux-musl.tar.gz \
    /tmp/railpack.tar.gz
RUN tar -xzf /tmp/railpack.tar.gz -C /usr/local/bin railpack

FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        git \
    && rm -rf /var/lib/apt/lists/*
RUN groupadd -r appuser && useradd -r -g appuser appuser
COPY --from=tools /usr/local/bin/railpack /usr/local/bin/railpack
COPY --from=moby/buildkit:v0.23.2@sha256:ddd1ca44b21eda906e81ab14a3d467fa6c39cd73b9a39df1196210edcb8db59e /usr/bin/buildctl /usr/local/bin/buildctl
COPY --from=build /out/deploy /usr/local/bin/deploy
USER appuser
EXPOSE 8080
ENTRYPOINT ["deploy"]
