#!/bin/bash

# Test build script for different platforms
set -e

echo "Testing builds for supported platforms..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Create build directory
mkdir -p build

# Test native build first
echo -e "${YELLOW}Testing native build...${NC}"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Map architecture names
if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
    ARCH="arm64"
fi

# Map OS names
if [ "$OS" = "darwin" ]; then
    OS="darwin"
elif [ "$OS" = "linux" ]; then
    OS="linux"
fi

echo "Building for ${OS}/${ARCH} (native)..."
CGO_ENABLED=1 go build -v -tags "sqlite_fts5,search" -o "build/srake-${OS}-${ARCH}" ./cmd/srake

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Native build successful${NC}"
    ls -la "build/srake-${OS}-${ARCH}"

    # Test the binary
    echo -e "${YELLOW}Testing native binary...${NC}"
    "./build/srake-${OS}-${ARCH}" version || echo "Version command not implemented yet"
else
    echo -e "${RED}✗ Native build failed${NC}"
    exit 1
fi

echo ""
echo -e "${YELLOW}Testing cross-platform builds (CGO disabled)...${NC}"

# Test cross-compilation for other platforms (CGO disabled)
test_cross_build() {
    local GOOS=$1
    local GOARCH=$2
    local OUTPUT="build/srake-${GOOS}-${GOARCH}"

    if [ "${GOOS}" = "windows" ]; then
        OUTPUT="${OUTPUT}.exe"
    fi

    echo "Building for ${GOOS}/${GOARCH}..."

    # Cross-compile without CGO
    GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=0 go build -v -o "${OUTPUT}" ./cmd/srake

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ ${GOOS}/${GOARCH} build successful${NC}"
        ls -la "${OUTPUT}"
    else
        echo -e "${RED}✗ ${GOOS}/${GOARCH} build failed${NC}"
    fi
    echo ""
}

# Test cross-compilation for main platforms
if [ "$OS" != "linux" ]; then
    test_cross_build "linux" "amd64"
fi

if [ "$OS" != "darwin" ]; then
    test_cross_build "darwin" "amd64"
fi

if [ "$OS" != "windows" ]; then
    test_cross_build "windows" "amd64"
fi

echo -e "${GREEN}Build tests complete!${NC}"
echo ""
echo "Binaries created in ./build/:"
ls -la build/