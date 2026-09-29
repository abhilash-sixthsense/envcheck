#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT_DIR"

if [ "$#" -ne 1 ]; then
    echo "Usage:"
    echo "  ./scripts/release.sh v1.0.0"
    exit 1
fi

VERSION="$1"

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: invalid version: $VERSION"
    echo
    echo "Version must look like:"
    echo "  v1.0.0"
    echo "  v1.2.3"
    exit 1
fi

echo "========================================"
echo " Preparing release $VERSION"
echo "========================================"
echo

# ------------------------------------------------------------
# Check required commands
# ------------------------------------------------------------

if ! command -v go >/dev/null 2>&1; then
    echo "Error: Go is not installed."
    exit 1
fi

if ! command -v git >/dev/null 2>&1; then
    echo "Error: Git is not installed."
    exit 1
fi

# ------------------------------------------------------------
# Check Git working tree
# ------------------------------------------------------------

echo "Checking Git working tree..."

if [ -n "$(git status --porcelain)" ]; then
    echo
    echo "Error: Git working tree is not clean."
    echo
    git status --short
    echo
    echo "Commit or stash your changes before creating a release."
    exit 1
fi

# ------------------------------------------------------------
# Check current branch
# ------------------------------------------------------------

BRANCH="$(git branch --show-current)"

if [ "$BRANCH" != "master" ]; then
    echo
    echo "Warning: you are currently on branch '$BRANCH'."
    echo "Releases are normally created from 'master'."
    echo
    read -r -p "Continue? [y/N] " ANSWER

    case "$ANSWER" in
        y|Y)
            ;;
        *)
            echo "Release cancelled."
            exit 1
            ;;
    esac
fi

# ------------------------------------------------------------
# Check tag
# ------------------------------------------------------------

if git rev-parse "$VERSION" >/dev/null 2>&1; then
    echo
    echo "Error: Git tag $VERSION already exists."
    exit 1
fi

# ------------------------------------------------------------
# Run tests
# ------------------------------------------------------------

echo
echo "Running tests..."
go test ./...

# ------------------------------------------------------------
# Check formatting
# ------------------------------------------------------------

echo
echo "Checking formatting..."

UNFORMATTED="$(gofmt -l .)"

if [ -n "$UNFORMATTED" ]; then
    echo
    echo "Error: the following files are not formatted:"
    echo
    echo "$UNFORMATTED"
    echo
    echo "Run:"
    echo "  gofmt -w ."
    exit 1
fi

echo "Formatting OK."

# ------------------------------------------------------------
# Build release binaries
# ------------------------------------------------------------

echo
echo "Building release binaries..."

VERSION="$VERSION" ./scripts/build.sh


# ------------------------------------------------------------
# Create checksums
# ------------------------------------------------------------

echo
echo "Creating checksums..."

cd "$ROOT_DIR/dist"

sha256sum \
    envcheck-linux-amd64 \
    envcheck-linux-arm64 \
    > checksums.txt

cd "$ROOT_DIR"

cat "$ROOT_DIR/dist/checksums.txt"

# ------------------------------------------------------------
# Create Git tag
# ------------------------------------------------------------

echo
echo "========================================"
echo " Everything looks good"
echo "========================================"
echo

echo "Version: $VERSION"
echo
echo "The following files were created:"
echo
ls -lh \
    "$ROOT_DIR/dist/envcheck-linux-amd64" \
    "$ROOT_DIR/dist/envcheck-linux-arm64" \
    "$ROOT_DIR/dist/checksums.txt"

echo
read -r -p "Create Git tag $VERSION? [y/N] " ANSWER

case "$ANSWER" in
    y|Y)
        ;;
    *)
        echo
        echo "Release cancelled."
        echo "Build files are still available under:"
        echo "  dist/"
        exit 0
        ;;
esac

git tag -a "$VERSION" -m "Release $VERSION"

echo
echo "Git tag created successfully:"
echo "  $VERSION"

echo
echo "Next step:"
echo
echo "  git push origin $VERSION"
echo
echo "GitHub Actions will build and publish the GitHub Release."