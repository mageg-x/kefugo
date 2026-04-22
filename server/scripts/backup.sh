#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATA_DIR="${1:-${ROOT_DIR}/data}"
OUT_DIR="${2:-${ROOT_DIR}/backups}"

if [[ ! -d "${DATA_DIR}" ]]; then
  echo "data directory not found: ${DATA_DIR}" >&2
  exit 1
fi

mkdir -p "${OUT_DIR}"
STAMP="$(date +%Y%m%d_%H%M%S)"
ARCHIVE="${OUT_DIR}/kefu_backup_${STAMP}.tar.gz"

tar -czf "${ARCHIVE}" -C "${DATA_DIR}" .
echo "backup created: ${ARCHIVE}"
