# syntax=docker/dockerfile:1.7

FROM node:24-alpine AS flowgram-editor-builder
WORKDIR /src/flowgram-editor

RUN corepack enable && corepack prepare pnpm@10.30.2 --activate
COPY flowgram-editor/package.json flowgram-editor/pnpm-lock.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
	pnpm install --frozen-lockfile

COPY flowgram-editor/ ./
RUN pnpm build

FROM node:24-alpine AS web-builder
WORKDIR /src/web

RUN corepack enable && corepack prepare pnpm@10.30.2 --activate
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
	pnpm install --frozen-lockfile

COPY web/ ./
COPY --from=flowgram-editor-builder /src/web/public/flowgram-editor ./public/flowgram-editor
RUN pnpm build:sdk && pnpm build

FROM golang:1.26-alpine AS server-builder
WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct

RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
	go mod download

COPY . ./
COPY --from=web-builder /src/web/out ./web/out

ARG TARGETOS=linux
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
	go build -v -trimpath -ldflags="-s -w" -o /out/agent-desk ./cmd/server

FROM golang:1.26-trixie AS server-builder-lancedb
WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct

ARG TARGETOS=linux
ARG TARGETARCH
ARG LANCEDB_VERSION=v0.1.2

RUN apt-get update \
	&& apt-get install -y --no-install-recommends bash binutils build-essential ca-certificates curl file git \
	&& rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
	go mod download

COPY . ./
COPY --from=web-builder /src/web/out ./web/out

RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	set -eux; \
	if [ "${TARGETOS}" != "linux" ]; then \
		echo "LanceDB Docker image supports TARGETOS=linux only, got ${TARGETOS}" >&2; \
		exit 1; \
	fi; \
	arch="${TARGETARCH:-$(go env GOARCH)}"; \
	case "$arch" in \
		amd64) ;; \
		*) echo "LanceDB Docker image currently supports linux/amd64 only, got linux/$arch" >&2; exit 1 ;; \
	esac; \
	mkdir -p /out; \
	curl -fL --retry 3 --retry-delay 2 -o /tmp/lancedb-go-native-binaries.tar.gz \
		"https://github.com/lancedb/lancedb-go/releases/download/${LANCEDB_VERSION}/lancedb-go-native-binaries.tar.gz"; \
	tar -xzf /tmp/lancedb-go-native-binaries.tar.gz -C /src; \
	test -f "/src/include/lancedb.h"; \
	test -f "/src/lib/linux_$arch/liblancedb_go.so"; \
	ldd --version | head -n 1; \
	file "/src/lib/linux_$arch/liblancedb_go.so"; \
	CGO_ENABLED=1 GOOS="${TARGETOS}" GOARCH="$arch" \
	CGO_CFLAGS="-I/src/include" \
	CGO_LDFLAGS="-L/src/lib/linux_$arch -llancedb_go -Wl,-rpath,/usr/local/lib -lm -ldl -lpthread" \
	go build -tags lancedb -v -trimpath -ldflags="-s -w" -o /out/agent-desk ./cmd/server; \
	ldd /out/agent-desk; \
	cp "/src/lib/linux_$arch/liblancedb_go.so" /out/liblancedb_go.so

FROM debian:trixie-slim AS app-lancedb
WORKDIR /app

ENV TZ=Asia/Shanghai
ENV LD_LIBRARY_PATH=/usr/local/lib

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates libgcc-s1 tzdata wget \
	&& mkdir -p /app/config /app/data/storage /app/data/lancedb \
	&& rm -rf /var/lib/apt/lists/*

COPY --from=server-builder-lancedb /out/agent-desk /app/agent-desk
COPY --from=server-builder-lancedb /out/liblancedb_go.so /usr/local/lib/liblancedb_go.so
COPY config/config.example.yaml /app/config/config.example.yaml
COPY config/config.example.yaml /app/config/config.yaml

EXPOSE 8083
VOLUME ["/app/data"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
	CMD wget -qO- http://127.0.0.1:8083/ >/dev/null || exit 1

CMD ["/app/agent-desk", "-config", "/app/config/config.yaml"]

FROM alpine:3.22 AS app
WORKDIR /app

ENV TZ=Asia/Shanghai

RUN apk add --no-cache ca-certificates tzdata wget \
	&& mkdir -p /app/config /app/data/storage

COPY --from=server-builder /out/agent-desk /app/agent-desk
COPY config/config.example.yaml /app/config/config.example.yaml
COPY config/config.example.yaml /app/config/config.yaml

EXPOSE 8083
VOLUME ["/app/data"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
	CMD wget -qO- http://127.0.0.1:8083/ >/dev/null || exit 1

CMD ["/app/agent-desk", "-config", "/app/config/config.yaml"]
