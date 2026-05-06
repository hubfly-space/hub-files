#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 <version> <goos> <goarch>" >&2
  exit 1
fi

VERSION="$1"
GOOS="$2"
GOARCH="$3"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${ROOT_DIR}/dist"
FRONTEND_DIST_DIR="${ROOT_DIR}/frontend/dist"
VERSION_STRIPPED="${VERSION#v}"
ASSET_BASENAME="hubfly-files_${VERSION_STRIPPED}_${GOOS}_${GOARCH}"
PACKAGE_DIR="${OUT_DIR}/${ASSET_BASENAME}"
ARCHIVE_PATH="${OUT_DIR}/${ASSET_BASENAME}.tar.gz"
COMMIT_SHA="${GITHUB_SHA:-$(git -C "${ROOT_DIR}" rev-parse --short HEAD)}"
BUILD_DATE="${BUILD_DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"

if [[ ! -f "${FRONTEND_DIST_DIR}/index.html" ]]; then
  echo "frontend build not found at ${FRONTEND_DIST_DIR}; run the frontend build first" >&2
  exit 1
fi

rm -rf "${PACKAGE_DIR}"
mkdir -p "${PACKAGE_DIR}/bin" "${PACKAGE_DIR}/web" "${PACKAGE_DIR}/systemd" "${PACKAGE_DIR}/config"

CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" \
  go build \
  -trimpath \
  -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT_SHA} -X main.buildDate=${BUILD_DATE}" \
  -o "${PACKAGE_DIR}/bin/hubfly-files" \
  ./cmd/hubfly-files

cp -R "${FRONTEND_DIST_DIR}/." "${PACKAGE_DIR}/web/"
cp "${ROOT_DIR}/packaging/hubfly-files.service" "${PACKAGE_DIR}/systemd/"
cp "${ROOT_DIR}/packaging/hubfly-files.env.example" "${PACKAGE_DIR}/config/"
cp "${ROOT_DIR}/scripts/install.sh" "${PACKAGE_DIR}/"
cp "${ROOT_DIR}/README.md" "${PACKAGE_DIR}/README.md"

tar -C "${OUT_DIR}" -czf "${ARCHIVE_PATH}" "${ASSET_BASENAME}"
(
  cd "${OUT_DIR}"
  sha256sum "${ASSET_BASENAME}.tar.gz" > "${ASSET_BASENAME}.tar.gz.sha256"
)
