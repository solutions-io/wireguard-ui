#!/usr/bin/env bash
# Build the wireguard-ui binary for the VPN server into dist/. Always build with this script:
# prepare_assets.sh copies custom/js into assets/, and the binary embeds assets/.
# Without a local Go installation, the Go build runs in the golang container.
set -euo pipefail

cd "$(dirname "$0")"

GO_IMAGE="golang:1.27"
GOARCH="${GOARCH:-arm64}"
VERSION="${VERSION:-$(git describe --tags --always --dirty)}"
COMMIT="$(git rev-parse --short HEAD)"
REF="$(git rev-parse --abbrev-ref HEAD)"
BUILD_TIME="$(date -u +'%Y-%m-%d %H:%M:%S UTC')"
OUT="dist/wireguard-ui-linux-${GOARCH}"
LDFLAGS="-s -w -X main.appVersion=${VERSION} -X main.gitCommit=${COMMIT} -X main.gitRef=${REF} -X 'main.buildTime=${BUILD_TIME}'"

bash prepare_assets.sh
mkdir -p dist

if command -v go >/dev/null 2>&1; then
  CGO_ENABLED=0 GOOS=linux GOARCH="${GOARCH}" go build -trimpath -ldflags "${LDFLAGS}" -o "${OUT}" .
else
  # pwd -W gives the Windows path in Git Bash; MSYS_NO_PATHCONV stops Git Bash from rewriting the paths.
  SRC="$(pwd -W 2>/dev/null || pwd)"
  MSYS_NO_PATHCONV=1 docker run --rm -v "${SRC}:/src" -w /src \
    -e CGO_ENABLED=0 -e GOOS=linux -e GOARCH="${GOARCH}" \
    "${GO_IMAGE}" go build -buildvcs=false -trimpath -ldflags "${LDFLAGS}" -o "${OUT}" .
fi

echo "Built ${OUT} (${VERSION})"
sha256sum "${OUT}"
