#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <backup.tar.gz> [target_data_dir]" >&2
  exit 1
fi

ARCHIVE="$1"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET_DIR="${2:-${ROOT_DIR}/data}"

if [[ ! -f "${ARCHIVE}" ]]; then
  echo "archive not found: ${ARCHIVE}" >&2
  exit 1
fi

mkdir -p "${TARGET_DIR}"
tar -xzf "${ARCHIVE}" -C "${TARGET_DIR}"
echo "restore completed to: ${TARGET_DIR}"
