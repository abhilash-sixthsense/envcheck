#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="$ROOT_DIR/localdist"

cd "$ROOT_DIR"

VERSION="${VERSION:-dev}"

echo "========================================"
echo " Building envcheck"
echo " Version: $VERSION"
echo "========================================"
echo

mkdir -p "$DIST_DIR"

echo "Building Linux AMD64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -trimpath \
    -ldflags="-s -w -X main.version=$VERSION" \
    -o "$DIST_DIR/envcheck-linux-amd64" .

echo "Building Linux ARM64..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build \
    -trimpath \
    -ldflags="-s -w -X main.version=$VERSION" \
    -o "$DIST_DIR/envcheck-linux-arm64" .

echo
echo "Build completed successfully."
echo

ls -lh \
    "$DIST_DIR/envcheck-linux-amd64" \
    "$DIST_DIR/envcheck-linux-arm64"

echo
echo "Verify:"
echo "  $DIST_DIR/envcheck-linux-amd64 -v"
echo "  $DIST_DIR/envcheck-linux-arm64 -v"
