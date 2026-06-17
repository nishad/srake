# syntax=docker/dockerfile:1

# ── Stage 1: build the SvelteKit SPA ────────────────────────────────────────
# The frontend is embedded into the Go binary, so it must be built first and
# placed into internal/web/build/ before the Go build runs.
FROM node:20-alpine AS web-builder
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ── Stage 2: build the Go binary (with the embedded web UI) ──────────────────
FROM golang:1.25-alpine AS go-builder
RUN apk add --no-cache git gcc musl-dev sqlite-dev
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

# ── Stage 3: minimal runtime ─────────────────────────────────────────────────
FROM alpine:latest
RUN apk add --no-cache ca-certificates sqlite wget

COPY --from=go-builder /build/srake /usr/local/bin/srake

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
