#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${1:-http://127.0.0.1:5300}"
TOTAL="${2:-100}"
CONCURRENCY="${3:-20}"

TMP_FILE="$(mktemp)"
trap 'rm -f "${TMP_FILE}"' EXIT

seq 1 "${TOTAL}" | xargs -n1 -P "${CONCURRENCY}" -I{} bash -c '
  ts=$(date +%s%N)
  code=$(curl -s -o /dev/null -w "%{http_code}" "'"${BASE_URL}"'/healthz")
  te=$(date +%s%N)
  dur=$(( (te-ts)/1000000 ))
  echo "$code $dur"
' >> "${TMP_FILE}"

total_lines=$(wc -l < "${TMP_FILE}" | tr -d ' ')
ok_lines=$(awk '$1=="200"{c++} END{print c+0}' "${TMP_FILE}")
err_lines=$(( total_lines - ok_lines ))
p95=$(awk '{print $2}' "${TMP_FILE}" | sort -n | awk -v n="${total_lines}" 'BEGIN{idx=int(n*0.95); if(idx<1) idx=1} NR==idx{print $1}')

echo "requests=${total_lines}"
echo "ok=${ok_lines}"
echo "error=${err_lines}"
echo "p95_ms=${p95:-0}"
