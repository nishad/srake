# syntax=docker/dockerfile:1

# ── Stage 1: build the SvelteKit SPA ────────────────────────────────────────
# The frontend is embedded into the Go binary, so it must be built first and
# placed into internal/web/build/ before the Go build runs.
FROM node:20-bookworm-slim AS web-builder
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ── Stage 2: build the Go binary (with the embedded web UI) ──────────────────
# Debian (glibc) is used rather than Alpine (musl) so the resulting binary is
# compatible with Microsoft's prebuilt onnxruntime, which is a glibc build.
FROM golang:1.25-bookworm AS go-builder
RUN apt-get update && apt-get install -y --no-install-recommends \
        gcc libc6-dev libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Embed the freshly built SPA. The repo ships internal/web/build/ empty
# (only .gitkeep), so the compiled web assets are copied in from the web stage
# before the Go build runs //go:embed all:build.
COPY --from=web-builder /web/build/ /web-build/
RUN rm -rf internal/web/build/* && cp -r /web-build/. internal/web/build/

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# CGO is required for go-sqlite3; FTS5 + search features are enabled via tags.
RUN CGO_ENABLED=1 GOOS=linux go build \
    -tags "sqlite_fts5,search" \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildDate=${BUILD_DATE}" \
    -o srake ./cmd/srake

# ── Stage 3: fetch the onnxruntime shared library ────────────────────────────
# Microsoft publishes glibc Linux builds for x64 and aarch64. TARGETARCH is
# provided automatically by BuildKit (amd64 / arm64).
FROM debian:bookworm-slim AS ort-fetch
ARG TARGETARCH
ARG ONNXRUNTIME_VERSION=1.26.0
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates \
    && rm -rf /var/lib/apt/lists/*
RUN set -eux; \
    case "${TARGETARCH}" in \
        amd64) ort_arch="x64" ;; \
        arm64) ort_arch="aarch64" ;; \
        *) echo "unsupported TARGETARCH: ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    url="https://github.com/microsoft/onnxruntime/releases/download/v${ONNXRUNTIME_VERSION}/onnxruntime-linux-${ort_arch}-${ONNXRUNTIME_VERSION}.tgz"; \
    curl -fsSL "$url" -o /tmp/ort.tgz; \
    mkdir -p /opt/onnxruntime; \
    tar -xzf /tmp/ort.tgz -C /opt/onnxruntime --strip-components=1; \
    rm /tmp/ort.tgz

# ── Stage 4: minimal glibc runtime ───────────────────────────────────────────
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates libsqlite3-0 wget \
    && rm -rf /var/lib/apt/lists/*

COPY --from=go-builder /build/srake /usr/local/bin/srake

# Install the onnxruntime shared library so semantic (vector) search works.
# The Go embedder finds it via SRAKE_ONNXRUNTIME_LIB.
COPY --from=ort-fetch /opt/onnxruntime/lib/libonnxruntime.so* /usr/local/lib/
RUN ldconfig
ENV SRAKE_ONNXRUNTIME_LIB=/usr/local/lib/libonnxruntime.so

# SRAKE follows the XDG spec; point its data dir at the mounted volume so the
# database at /data/srake.db is picked up automatically.
ENV SRAKE_DATA_DIR=/data
RUN mkdir -p /data
VOLUME ["/data"]
WORKDIR /data

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

ENTRYPOINT ["srake"]
# Default: serve the API + embedded web UI on 0.0.0.0:8080 using /data/srake.db.
CMD ["server", "--host", "0.0.0.0", "--port", "8080", "--db", "/data/srake.db"]
