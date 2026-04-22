#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${ROOT_DIR}/dist/macos"
APP_NAME="KefuServer"
APP_BUNDLE="${OUT_DIR}/${APP_NAME}.app"
MACOS_DIR="${APP_BUNDLE}/Contents/MacOS"
RES_DIR="${APP_BUNDLE}/Contents/Resources"

mkdir -p "${OUT_DIR}" "${MACOS_DIR}" "${RES_DIR}"

echo "[macOS] build gui binary..."
(
  cd "${ROOT_DIR}"
  GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 \
  GOCACHE=/tmp/go-build-cache \
  go build -tags gui -o "${MACOS_DIR}/${APP_NAME}" .
)

echo "[macOS] generate app icon (.icns)..."
ICONSET_DIR="$(mktemp -d /tmp/kefu.iconset.XXXXXX)"
cleanup() {
  rm -rf "${ICONSET_DIR}"
}
trap cleanup EXIT

cp "${ROOT_DIR}/systray/icon.png" "${ICONSET_DIR}/icon_512x512.png"
sips -z 16 16     "${ROOT_DIR}/systray/icon.png" --out "${ICONSET_DIR}/icon_16x16.png" >/dev/null
sips -z 32 32     "${ROOT_DIR}/systray/icon.png" --out "${ICONSET_DIR}/icon_16x16@2x.png" >/dev/null
sips -z 32 32     "${ROOT_DIR}/systray/icon.png" --out "${ICONSET_DIR}/icon_32x32.png" >/dev/null
sips -z 64 64     "${ROOT_DIR}/systray/icon.png" --out "${ICONSET_DIR}/icon_32x32@2x.png" >/dev/null
sips -z 128 128   "${ROOT_DIR}/systray/icon.png" --out "${ICONSET_DIR}/icon_128x128.png" >/dev/null
sips -z 256 256   "${ROOT_DIR}/systray/icon.png" --out "${ICONSET_DIR}/icon_128x128@2x.png" >/dev/null
sips -z 256 256   "${ROOT_DIR}/systray/icon.png" --out "${ICONSET_DIR}/icon_256x256.png" >/dev/null
sips -z 512 512   "${ROOT_DIR}/systray/icon.png" --out "${ICONSET_DIR}/icon_256x256@2x.png" >/dev/null
sips -z 512 512   "${ROOT_DIR}/systray/icon.png" --out "${ICONSET_DIR}/icon_512x512@2x.png" >/dev/null

iconutil -c icns "${ICONSET_DIR}" -o "${RES_DIR}/AppIcon.icns"

cat >"${APP_BUNDLE}/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key>
  <string>KefuServer</string>
  <key>CFBundleDisplayName</key>
  <string>KefuServer</string>
  <key>CFBundleIdentifier</key>
  <string>cn.lingdian.kefu.server</string>
  <key>CFBundleVersion</key>
  <string>1.0.0</string>
  <key>CFBundleShortVersionString</key>
  <string>1.0.0</string>
  <key>CFBundleExecutable</key>
  <string>KefuServer</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleIconFile</key>
  <string>AppIcon</string>
  <key>LSMinimumSystemVersion</key>
  <string>11.0</string>
</dict>
</plist>
PLIST

echo "[macOS] done: ${APP_BUNDLE}"
