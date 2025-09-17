#!/bin/bash

# Build release binaries for multiple platforms
# Usage: ./scripts/build-release.sh [version]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get version from argument or git tag
VERSION=${1:-$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")}
COMMIT=$(git rev-parse --short HEAD)
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

echo -e "${GREEN}Building srake ${VERSION}${NC}"
echo "Commit: ${COMMIT}"
echo "Build Date: ${BUILD_DATE}"

# Create dist directory
DIST_DIR="dist"
rm -rf ${DIST_DIR}
mkdir -p ${DIST_DIR}

# Build targets
TARGETS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

# Build function
build() {
    local os=$1
    local arch=$2
    local output="srake-${os}-${arch}"

    if [ "${os}" = "windows" ]; then
        output="${output}.exe"
    fi

    echo -e "${YELLOW}Building ${os}/${arch}...${NC}"

    GOOS=${os} GOARCH=${arch} CGO_ENABLED=0 go build \
        -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildDate=${BUILD_DATE}" \
        -o "${DIST_DIR}/${output}" \
        ./cmd/srake

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Built ${output}${NC}"

        # Create archive
        cd ${DIST_DIR}
        if [ "${os}" = "windows" ]; then
            zip -q "../srake-${VERSION}-${os}-${arch}.zip" ${output} ../README.md ../LICENSE
        else
            tar czf "../srake-${VERSION}-${os}-${arch}.tar.gz" ${output} ../README.md ../LICENSE
        fi
        cd ..

        # Remove binary after archiving
        rm "${DIST_DIR}/${output}"
    else
        echo -e "${RED}✗ Failed to build ${os}/${arch}${NC}"
        exit 1
    fi
}

# Build all targets
for target in "${TARGETS[@]}"; do
    IFS='/' read -r os arch <<< "${target}"
    build ${os} ${arch}
done

# Generate checksums
echo -e "${YELLOW}Generating checksums...${NC}"
sha256sum srake-${VERSION}-*.{tar.gz,zip} > srake-${VERSION}-SHA256SUMS.txt

# Clean up
rm -rf ${DIST_DIR}

echo -e "${GREEN}✓ Release build complete!${NC}"
echo "Archives created:"
ls -lh srake-${VERSION}-*.{tar.gz,zip}
echo ""
echo "Checksums file: srake-${VERSION}-SHA256SUMS.txt"

# Create release notes template
cat > RELEASE_NOTES.md << EOF
# srake ${VERSION}

## What's Changed

### Features
-

### Bug Fixes
-

### Performance Improvements
-

### Documentation
-

## Installation

### Pre-built binaries

Download the appropriate archive for your platform from the releases page.

\`\`\`bash
# Linux/macOS
tar -xzf srake-${VERSION}-<os>-<arch>.tar.gz
sudo mv srake /usr/local/bin/

# Windows
unzip srake-${VERSION}-windows-amd64.zip
\`\`\`

### Docker

\`\`\`bash
docker pull ghcr.io/nishad/srake:${VERSION}
\`\`\`

### Go Install

\`\`\`bash
go install github.com/nishad/srake/cmd/srake@${VERSION}
\`\`\`

## Checksums

See \`srake-${VERSION}-SHA256SUMS.txt\` for file checksums.

## Full Changelog

See [CHANGELOG.md](https://github.com/nishad/srake/blob/${VERSION}/CHANGELOG.md)

EOF

echo ""
echo -e "${GREEN}Release notes template created: RELEASE_NOTES.md${NC}"
echo ""
echo "Next steps:"
echo "1. Edit RELEASE_NOTES.md with actual changes"
echo "2. Create git tag: git tag -a ${VERSION} -m 'Release ${VERSION}'"
echo "3. Push tag: git push origin ${VERSION}"
echo "4. Upload archives to GitHub releases"