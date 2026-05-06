#!/usr/bin/env bash

set -euo pipefail

REPO="hubfly-space/hub-files"
VERSION="latest"
BIN_DIR="/usr/local/bin"
INSTALL_ROOT="/opt/hubfly-files"
CONFIG_DIR="/etc/hubfly-files"
SERVICE_NAME="hubfly-files"
SKIP_SERVICE=0

usage() {
  cat <<'EOF'
usage: install.sh [options]

options:
  --version <tag>      install a specific release tag such as v1.2.3
  --bin-dir <path>     install the binary into this directory
  --install-root <path> install web assets and support files into this directory
  --config-dir <path>  install the environment file into this directory
  --repo <owner/name>  override the GitHub repository
  --skip-service       install files without enabling or restarting systemd
  -h, --help           show this help text
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      VERSION="${2:?missing value for --version}"
      shift 2
      ;;
    --bin-dir)
      BIN_DIR="${2:?missing value for --bin-dir}"
      shift 2
      ;;
    --install-root)
      INSTALL_ROOT="${2:?missing value for --install-root}"
      shift 2
      ;;
    --config-dir)
      CONFIG_DIR="${2:?missing value for --config-dir}"
      shift 2
      ;;
    --repo)
      REPO="${2:?missing value for --repo}"
      shift 2
      ;;
    --skip-service)
      SKIP_SERVICE=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ "${VERSION}" != latest && "${VERSION}" != v* ]]; then
  VERSION="v${VERSION}"
fi

if [[ ${EUID} -eq 0 ]]; then
  SUDO=""
else
  if ! command -v sudo >/dev/null 2>&1; then
    echo "this installer needs root privileges; rerun as root or install sudo" >&2
    exit 1
  fi
  SUDO="sudo"
fi

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "required command not found: $1" >&2
    exit 1
  fi
}

need_cmd curl
need_cmd tar
need_cmd sha256sum
need_cmd install

map_arch() {
  case "$(uname -m)" in
    x86_64|amd64)
      echo "amd64"
      ;;
    aarch64|arm64)
      echo "arm64"
      ;;
    *)
      echo "unsupported architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

resolve_version() {
  if [[ "${VERSION}" != latest ]]; then
    echo "${VERSION}"
    return
  fi

  curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
    | head -n 1
}

ARCH="$(map_arch)"
TAG="$(resolve_version)"

if [[ -z "${TAG}" ]]; then
  echo "failed to resolve a release tag for ${REPO}" >&2
  exit 1
fi

VERSION_STRIPPED="${TAG#v}"
ASSET_BASENAME="hubfly-files_${VERSION_STRIPPED}_linux_${ARCH}"
ARCHIVE_NAME="${ASSET_BASENAME}.tar.gz"
CHECKSUMS_NAME="hubfly-files_${VERSION_STRIPPED}_checksums.txt"
BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

curl -fsSL "${BASE_URL}/${ARCHIVE_NAME}" -o "${TMP_DIR}/${ARCHIVE_NAME}"
curl -fsSL "${BASE_URL}/${CHECKSUMS_NAME}" -o "${TMP_DIR}/${CHECKSUMS_NAME}"

(
  cd "${TMP_DIR}"
  grep " ${ARCHIVE_NAME}$" "${CHECKSUMS_NAME}" | sha256sum -c -
)

tar -xzf "${TMP_DIR}/${ARCHIVE_NAME}" -C "${TMP_DIR}"
PACKAGE_DIR="${TMP_DIR}/${ASSET_BASENAME}"

if [[ ! -d "${PACKAGE_DIR}" ]]; then
  echo "release archive did not contain ${ASSET_BASENAME}" >&2
  exit 1
fi

${SUDO} install -d "${BIN_DIR}" "${INSTALL_ROOT}" "${CONFIG_DIR}" "${INSTALL_ROOT}/demo"
${SUDO} install -m 0755 "${PACKAGE_DIR}/bin/hubfly-files" "${BIN_DIR}/hubfly-files"

${SUDO} rm -rf "${INSTALL_ROOT}/web.tmp"
${SUDO} mkdir -p "${INSTALL_ROOT}/web.tmp"
${SUDO} cp -R "${PACKAGE_DIR}/web/." "${INSTALL_ROOT}/web.tmp/"
${SUDO} rm -rf "${INSTALL_ROOT}/web"
${SUDO} mv "${INSTALL_ROOT}/web.tmp" "${INSTALL_ROOT}/web"

ENV_FILE="${CONFIG_DIR}/hubfly-files.env"
if [[ ! -f "${ENV_FILE}" ]]; then
  ${SUDO} install -m 0644 "${PACKAGE_DIR}/config/hubfly-files.env.example" "${ENV_FILE}"
else
  if ! ${SUDO} grep -q '^HUBFLY_UI_DIR=' "${ENV_FILE}"; then
    echo "HUBFLY_UI_DIR=${INSTALL_ROOT}/web" | ${SUDO} tee -a "${ENV_FILE}" >/dev/null
  fi
  if ! ${SUDO} grep -q '^HUBFLY_DEMO_DIR=' "${ENV_FILE}"; then
    echo "HUBFLY_DEMO_DIR=${INSTALL_ROOT}/demo" | ${SUDO} tee -a "${ENV_FILE}" >/dev/null
  fi
fi

SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
if [[ ${SKIP_SERVICE} -eq 0 && -d /run/systemd/system && -x "$(command -v systemctl || true)" ]]; then
  ${SUDO} install -m 0644 "${PACKAGE_DIR}/systemd/hubfly-files.service" "${SERVICE_FILE}"
  ${SUDO} systemctl daemon-reload
  if ${SUDO} systemctl is-enabled --quiet "${SERVICE_NAME}" || ${SUDO} systemctl is-active --quiet "${SERVICE_NAME}"; then
    ${SUDO} systemctl restart "${SERVICE_NAME}"
  else
    ${SUDO} systemctl enable --now "${SERVICE_NAME}"
  fi
else
  echo "systemd service setup skipped"
fi

echo "installed HubFly Files ${TAG}"
echo "binary: ${BIN_DIR}/hubfly-files"
echo "web assets: ${INSTALL_ROOT}/web"
echo "config: ${ENV_FILE}"
echo "ui: http://localhost:10015"
echo "management: http://localhost:10014"
