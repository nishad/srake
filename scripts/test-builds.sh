#!/bin/bash

# Test build script for different platforms
set -e

echo "Testing builds for multiple platforms..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to test a build
test_build() {
    local GOOS=$1
    local GOARCH=$2
    local CGO_ENABLED=$3
    local OUTPUT="build/srake-${GOOS}-${GOARCH}"

    if [ "${GOOS}" = "windows" ]; then
        OUTPUT="${OUTPUT}.exe"
    fi

    echo -e "${YELLOW}Building for ${GOOS}/${GOARCH} (CGO=${CGO_ENABLED})...${NC}"

    # Set environment variables
    export GOOS=${GOOS}
    export GOARCH=${GOARCH}
    export CGO_ENABLED=${CGO_ENABLED}

    # Build command
    if [ "${CGO_ENABLED}" = "1" ]; then
        if go build -v -tags "sqlite_fts5,search" -o "${OUTPUT}" ./cmd/srake 2>/dev/null; then
            echo -e "${GREEN}✓ ${GOOS}/${GOARCH} build successful${NC}"
            ls -la "${OUTPUT}"
        else
            echo -e "${RED}✗ ${GOOS}/${GOARCH} build failed (CGO required, may need native compilation)${NC}"
        fi
    else
        if go build -v -o "${OUTPUT}" ./cmd/srake 2>/dev/null; then
            echo -e "${GREEN}✓ ${GOOS}/${GOARCH} build successful${NC}"
            ls -la "${OUTPUT}"
        else
            echo -e "${RED}✗ ${GOOS}/${GOARCH} build failed${NC}"
        fi
    fi

    echo ""
}

# Create build directory
mkdir -p build

# Test native build first
echo -e "${YELLOW}Testing native build...${NC}"
unset GOOS GOARCH
CGO_ENABLED=1 go build -v -tags "sqlite_fts5,search" -o build/srake-native ./cmd/srake
echo -e "${GREEN}✓ Native build successful${NC}"
echo ""

# Test cross-compilation
echo -e "${YELLOW}Testing cross-compilation...${NC}"

# Linux builds
test_build "linux" "amd64" "0"
test_build "linux" "arm64" "0"

# macOS builds (CGO disabled for cross-compilation)
test_build "darwin" "amd64" "0"
test_build "darwin" "arm64" "0"

# Windows builds (always CGO disabled)
test_build "windows" "amd64" "0"

echo -e "${GREEN}Build tests complete!${NC}"
echo ""
echo "Binaries created in ./build/:"
ls -la build/