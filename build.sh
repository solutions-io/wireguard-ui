#!/usr/bin/env bash
# Build the wireguard-ui binary for the VPN server into dist/. Always build with this script:
# prepare_assets.sh copies custom/js into assets/, and the binary embeds assets/.
set -euo pipefail

cd "$(dirname "$0")"

GOARCH="${GOARCH:-arm64}"
VERSION="${VERSION:-$(git describe --tags --always --dirty)}"
COMMIT="$(git rev-parse --short HEAD)"
REF="$(git rev-parse --abbrev-ref HEAD)"
BUILD_TIME="$(date -u +'%Y-%m-%d %H:%M:%S UTC')"
OUT="dist/wireguard-ui-linux-${GOARCH}"

bash prepare_assets.sh
mkdir -p dist
CGO_ENABLED=0 GOOS=linux GOARCH="${GOARCH}" go build -trimpath \
  -ldflags "-s -w -X main.appVersion=${VERSION} -X main.gitCommit=${COMMIT} -X main.gitRef=${REF} -X 'main.buildTime=${BUILD_TIME}'" \
  -o "${OUT}" .

echo "Built ${OUT} (${VERSION})"
sha256sum "${OUT}"
