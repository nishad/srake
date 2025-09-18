#!/bin/bash

# Setup script for FAISS library required by Bleve vector search
# Supports macOS (Intel & ARM) and Linux

set -e

echo "Setting up FAISS library for Bleve vector search..."

# Detect OS and architecture
OS=$(uname -s)
ARCH=$(uname -m)

echo "Detected: $OS $ARCH"

# Function to install dependencies
install_deps() {
    if [[ "$OS" == "Darwin" ]]; then
        echo "Checking for Homebrew..."
        if ! command -v brew &> /dev/null; then
            echo "Homebrew not found. Please install from https://brew.sh"
            exit 1
        fi
        echo "Installing dependencies via Homebrew..."
        brew install cmake libomp openblas
    elif [[ "$OS" == "Linux" ]]; then
        echo "Installing dependencies..."
        if command -v apt-get &> /dev/null; then
            sudo apt-get update
            sudo apt-get install -y cmake build-essential libomp-dev libopenblas-dev
        elif command -v yum &> /dev/null; then
            sudo yum install -y cmake gcc gcc-c++ openblas-devel
        else
            echo "Unsupported Linux distribution. Please install cmake, gcc, and OpenBLAS manually."
            exit 1
        fi
    fi
}

# Create temporary build directory
BUILD_DIR=$(mktemp -d)
cd "$BUILD_DIR"

echo "Cloning Bleve's FAISS fork..."
git clone https://github.com/blevesearch/faiss.git
cd faiss

# Checkout the required version
git checkout 352484e

# Configure build
echo "Configuring FAISS build..."
cmake -B build . \
    -DCMAKE_BUILD_TYPE=Release \
    -DFAISS_ENABLE_GPU=OFF \
    -DFAISS_ENABLE_PYTHON=OFF \
    -DFAISS_ENABLE_C_API=ON \
    -DBUILD_TESTING=OFF \
    -DBUILD_SHARED_LIBS=ON

# Build FAISS
echo "Building FAISS (this may take a few minutes)..."
cmake --build build --config Release -j$(nproc 2>/dev/null || sysctl -n hw.ncpu)

# Install FAISS
echo "Installing FAISS..."
if [[ "$OS" == "Darwin" ]]; then
    # Install to /usr/local on macOS
    sudo cmake --install build --prefix /usr/local

    # Update library paths
    echo "Updating library paths..."
    sudo update_dyld_shared_cache 2>/dev/null || true
else
    # Install to /usr/local on Linux
    sudo cmake --install build --prefix /usr/local
    sudo ldconfig
fi

# Verify installation
echo "Verifying FAISS installation..."
if [[ "$OS" == "Darwin" ]]; then
    if [ -f /usr/local/lib/libfaiss_c.dylib ]; then
        echo "✓ FAISS C API library found: /usr/local/lib/libfaiss_c.dylib"
    else
        echo "✗ FAISS C API library not found!"
        exit 1
    fi
else
    if [ -f /usr/local/lib/libfaiss_c.so ]; then
        echo "✓ FAISS C API library found: /usr/local/lib/libfaiss_c.so"
    else
        echo "✗ FAISS C API library not found!"
        exit 1
    fi
fi

# Clean up
cd /
rm -rf "$BUILD_DIR"

echo ""
echo "FAISS setup complete!"
echo ""
echo "To build srake with vector support, use:"
echo "  go build -tags vectors ./cmd/srake"
echo ""
echo "Or add to your Makefile:"
echo "  TAGS = -tags vectors"
echo ""