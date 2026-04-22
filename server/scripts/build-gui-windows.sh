#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${ROOT_DIR}/dist/windows"
APP_NAME="kefu-server"
BIN_NAME="${APP_NAME}.exe"

mkdir -p "${OUT_DIR}"

echo "[windows] generate tray/file icon resources..."
(
  cd "${ROOT_DIR}"
  GOCACHE=/tmp/go-build-cache go generate -tags gui ./systray
  GOCACHE=/tmp/go-build-cache go run github.com/akavel/rsrc@latest -ico systray/icon.ico -o windows/rsrc.syso
)

echo "[windows] build gui binary..."
(
  cd "${ROOT_DIR}"
  GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
  GOCACHE=/tmp/go-build-cache \
  go build -tags gui -o "${OUT_DIR}/${BIN_NAME}" .
)

echo "[windows] done: ${OUT_DIR}/${BIN_NAME}"
