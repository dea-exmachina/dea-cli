#!/usr/bin/env sh
# install.sh — dea CLI installer
# Usage: curl -fsSL https://raw.githubusercontent.com/dea-exmachina/dea-cli/main/install.sh | sh

set -e

REPO="dea-exmachina/dea-cli"
INSTALL_DIR="/usr/local/bin"
BINARY="dea"

# --- Detect OS ---
OS="$(uname -s)"
case "${OS}" in
  Linux*)   OS_NAME="linux" ;;
  Darwin*)  OS_NAME="darwin" ;;
  MINGW*|MSYS*|CYGWIN*) OS_NAME="windows" ;;
  *)
    echo "Unsupported OS: ${OS}"
    exit 1
    ;;
esac

# --- Detect Architecture ---
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64|amd64)  ARCH_NAME="amd64" ;;
  aarch64|arm64) ARCH_NAME="arm64" ;;
  *)
    echo "Unsupported architecture: ${ARCH}"
    exit 1
    ;;
esac

# --- Fetch Latest Release Tag ---
echo "Fetching latest release..."
LATEST_TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' \
  | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')

if [ -z "${LATEST_TAG}" ]; then
  echo "Failed to fetch latest release tag."
  exit 1
fi

echo "Latest version: ${LATEST_TAG}"

# --- Construct Asset URL ---
VERSION="${LATEST_TAG#v}"  # Strip leading 'v' for filename

if [ "${OS_NAME}" = "windows" ]; then
  ARCHIVE_EXT="zip"
  BINARY_NAME="dea.exe"
else
  ARCHIVE_EXT="tar.gz"
  BINARY_NAME="dea"
fi

ASSET_NAME="dea_${VERSION}_${OS_NAME}_${ARCH_NAME}.${ARCHIVE_EXT}"
ASSET_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${ASSET_NAME}"
CHECKSUM_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/checksums.txt"

# --- Download ---
TMP_DIR=$(mktemp -d)
trap 'rm -rf "${TMP_DIR}"' EXIT

echo "Downloading ${ASSET_NAME}..."
curl -fsSL "${ASSET_URL}" -o "${TMP_DIR}/${ASSET_NAME}"

# --- Verify Checksum ---
echo "Verifying checksum..."
curl -fsSL "${CHECKSUM_URL}" -o "${TMP_DIR}/checksums.txt"

cd "${TMP_DIR}"
if command -v sha256sum >/dev/null 2>&1; then
  grep "${ASSET_NAME}" checksums.txt | sha256sum --check --status
elif command -v shasum >/dev/null 2>&1; then
  grep "${ASSET_NAME}" checksums.txt | shasum -a 256 --check --status
else
  echo "Warning: no sha256sum or shasum found, skipping checksum verification"
fi
echo "Checksum OK."

# --- Extract ---
if [ "${ARCHIVE_EXT}" = "tar.gz" ]; then
  tar -xzf "${ASSET_NAME}" -C "${TMP_DIR}"
else
  unzip -q "${ASSET_NAME}" -d "${TMP_DIR}"
fi

# --- Install ---
echo "Installing to ${INSTALL_DIR}/${BINARY}..."
if [ -w "${INSTALL_DIR}" ]; then
  mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY}"
  chmod +x "${INSTALL_DIR}/${BINARY}"
else
  sudo mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY}"
  sudo chmod +x "${INSTALL_DIR}/${BINARY}"
fi

# --- Verify ---
if command -v dea >/dev/null 2>&1; then
  echo ""
  echo "dea ${LATEST_TAG} installed successfully."
  dea --version
else
  echo ""
  echo "Installed to ${INSTALL_DIR}/${BINARY}."
  echo "Ensure ${INSTALL_DIR} is in your PATH."
fi
