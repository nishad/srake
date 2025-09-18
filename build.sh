#!/bin/bash

# Build script for srake with FTS5 support
# This script builds srake with SQLite FTS5 enabled

set -e

# Get version information
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# Build with FTS5 support
echo "Building srake with FTS5 support..."
echo "Version: $VERSION"
echo "Commit: $COMMIT"
echo "Date: $BUILD_DATE"

go build -tags "sqlite_fts5" \
  -ldflags="-s -w -X main.version=$VERSION -X main.commit=$COMMIT -X main.date=$BUILD_DATE" \
  -o srake \
  ./cmd/srake

echo "Build complete: ./srake"
echo ""
echo "Note: FTS5 support is enabled for full-text search functionality."
echo "You can now use both FTS3 and FTS5 with the export command:"
echo "  ./srake db export -o output.sqlite --fts-version 5  # Use FTS5 (default)"
echo "  ./srake db export -o output.sqlite --fts-version 3  # Use FTS3 for compatibility"