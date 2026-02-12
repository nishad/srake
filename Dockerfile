# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# Build the binary
RUN CGO_ENABLED=1 GOOS=linux go build \
    -tags "sqlite_fts5,search" \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildDate=${BUILD_DATE}" \
    -o srake ./cmd/srake

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates sqlite

# Create data directory
RUN mkdir -p /data

# Copy binary from builder
COPY --from=builder /build/srake /usr/local/bin/srake

# Set data volume
VOLUME ["/data"]

# Set working directory
WORKDIR /data

# Expose API port
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["srake"]

# Default command
CMD ["--help"]