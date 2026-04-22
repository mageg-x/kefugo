#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SERVER_DIR="${ROOT_DIR}/server"

SIZES="${SIZES:-1000 5000 10000}"
REQUESTS="${REQUESTS:-200}"
CONCURRENCY="${CONCURRENCY:-20}"
BASE_PORT="${BASE_PORT:-5311}"
JWT_SECRET="${JWT_SECRET:-bench-secret}"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASS="${ADMIN_PASS:-12345678}"

SERVER_PID=""
SERVER_BIN=""
WORK_DIRS=()
RESULT_FILE="$(mktemp)"

cleanup() {
  if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" 2>/dev/null; then
    kill "${SERVER_PID}" >/dev/null 2>&1 || true
    wait "${SERVER_PID}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${SERVER_BIN}" ]]; then
    rm -f "${SERVER_BIN}" >/dev/null 2>&1 || true
  fi
  rm -f "${RESULT_FILE}" >/dev/null 2>&1 || true
  for dir in "${WORK_DIRS[@]}"; do
    rm -rf "${dir}" >/dev/null 2>&1 || true
  done
}
trap cleanup EXIT

wait_http_ok() {
  local url="$1"
  local tries=0
  until curl -sS "${url}" >/dev/null 2>&1; do
    tries=$((tries + 1))
    if (( tries > 400 )); then
      return 1
    fi
    sleep 0.2
  done
}

build_server_binary() {
  SERVER_BIN="$(mktemp /tmp/kefu-server-bench-bin-XXXXXX)"
  (
    cd "${SERVER_DIR}"
    CGO_ENABLED=0 go build -o "${SERVER_BIN}" ./main.go
  )
}

start_server() {
  local data_dir="$1"
  local port="$2"
  local log_file="$3"

  "${SERVER_BIN}" \
    -addr "127.0.0.1:${port}" \
    -data "${data_dir}" \
    -jwt-secret "${JWT_SECRET}" \
    -log-level error >"${log_file}" 2>&1 &
  SERVER_PID=$!

  if ! wait_http_ok "http://127.0.0.1:${port}/healthz"; then
    if [[ -f "${log_file}" ]]; then
      tail -n 80 "${log_file}" >&2 || true
    fi
    echo "server startup timeout, log=${log_file}" >&2
    return 1
  fi
}

stop_server() {
  if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" 2>/dev/null; then
    kill "${SERVER_PID}" >/dev/null 2>&1 || true
    wait "${SERVER_PID}" >/dev/null 2>&1 || true
  fi
  SERVER_PID=""
}

seed_dataset() {
  local db_path="$1"
  local rows="$2"
  (
    cd "${SERVER_DIR}"
    go run ./scripts/seed_session_index.go -db "${db_path}" -n "${rows}"
  )
}

login_token() {
  local base_url="$1"
  local payload
  payload="$(printf '{"username":"%s","password":"%s"}' "${ADMIN_USER}" "${ADMIN_PASS}")"
  local body
  body="$(curl -sS -X POST "${base_url}/api/v1/login" -H "Content-Type: application/json" -d "${payload}")"
  printf "%s" "${body}" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p'
}

run_case() {
  local name="$1"
  local url="$2"
  local token="$3"
  local data_size="$4"

  local tmp_file
  tmp_file="$(mktemp)"
  local running=0

  for ((i=1; i<=REQUESTS; i++)); do
    (
      local ts te dur code
      ts="$(date +%s%N)"
      code="$(curl -sS -o /dev/null -w "%{http_code}" -H "Authorization: Bearer ${token}" "${url}")"
      te="$(date +%s%N)"
      dur=$(( (te - ts) / 1000000 ))
      printf "%s %s\n" "${code}" "${dur}"
    ) >>"${tmp_file}" &

    running=$((running + 1))
    if (( running >= CONCURRENCY )); then
      wait -n
      running=$((running - 1))
    fi
  done
  wait

  local total ok err avg p50 p95 max
  total="$(wc -l < "${tmp_file}" | tr -d ' ')"
  ok="$(awk '$1=="200"{c++} END{print c+0}' "${tmp_file}")"
  err=$((total - ok))
  avg="$(awk '$1=="200"{s+=$2;c++} END{if(c>0) printf "%.2f", s/c; else print "0"}' "${tmp_file}")"
  p50="$(awk '$1=="200"{print $2}' "${tmp_file}" | sort -n | awk '
    {a[NR]=$1}
    END{
      if (NR==0) {print 0; exit}
      idx=int((NR+1)*0.50)
      if (idx < 1) idx=1
      if (idx > NR) idx=NR
      print a[idx]
    }'
  )"
  p95="$(awk '$1=="200"{print $2}' "${tmp_file}" | sort -n | awk '
    {a[NR]=$1}
    END{
      if (NR==0) {print 0; exit}
      idx=int((NR*95 + 99)/100)
      if (idx < 1) idx=1
      if (idx > NR) idx=NR
      print a[idx]
    }'
  )"
  max="$(awk '$1=="200"{if($2>m)m=$2} END{print m+0}' "${tmp_file}")"

  printf "%s|%s|%s|%s|%s|%s|%s|%s\n" \
    "${data_size}" "${name}" "${total}" "${ok}" "${err}" "${avg}" "${p50}" "${p95}" >> "${RESULT_FILE}"
  printf "size=%s case=%s total=%s ok=%s err=%s avg_ms=%s p50_ms=%s p95_ms=%s max_ms=%s\n" \
    "${data_size}" "${name}" "${total}" "${ok}" "${err}" "${avg}" "${p50}" "${p95}" "${max}"

  rm -f "${tmp_file}"
}

print_summary() {
  echo
  echo "==== summary ===="
  printf "%-8s %-18s %-8s %-8s %-8s %-10s %-8s %-8s\n" "size" "case" "total" "ok" "err" "avg_ms" "p50" "p95"
  while IFS='|' read -r size case total ok err avg p50 p95; do
    printf "%-8s %-18s %-8s %-8s %-8s %-10s %-8s %-8s\n" "${size}" "${case}" "${total}" "${ok}" "${err}" "${avg}" "${p50}" "${p95}"
  done < "${RESULT_FILE}"
}

main() {
  build_server_binary
  local idx=0
  for size in ${SIZES}; do
    idx=$((idx + 1))
    local port=$((BASE_PORT + idx))
    local work_dir
    work_dir="$(mktemp -d "/tmp/kefu-bench-${size}-XXXX")"
    WORK_DIRS+=("${work_dir}")
    local data_dir="${work_dir}/data"
    local log_file="${work_dir}/server.log"
    mkdir -p "${data_dir}"

    echo "prepare dataset size=${size} port=${port} data_dir=${data_dir}"
    start_server "${data_dir}" "${port}" "${log_file}"
    stop_server
    seed_dataset "${data_dir}/kefu.db" "${size}"
    start_server "${data_dir}" "${port}" "${log_file}"

    local base_url="http://127.0.0.1:${port}"
    local token
    token="$(login_token "${base_url}")"
    if [[ -z "${token}" ]]; then
      echo "login failed for dataset size=${size}, log=${log_file}" >&2
      return 1
    fi

    local now start
    now="$(date +%s)"
    start=$((now - 7200))

    run_case "all" "${base_url}/api/v1/sessions/list?page=1&page_size=20&no_cache=1" "${token}" "${size}"
    run_case "app_id" "${base_url}/api/v1/sessions/list?page=1&page_size=20&app_id=app-01&no_cache=1" "${token}" "${size}"
    run_case "status_unread" "${base_url}/api/v1/sessions/list?page=1&page_size=20&status=unread&no_cache=1" "${token}" "${size}"
    run_case "assigned_mine" "${base_url}/api/v1/sessions/list?page=1&page_size=20&assigned=mine&no_cache=1" "${token}" "${size}"
    run_case "time_range" "${base_url}/api/v1/sessions/list?page=1&page_size=20&start_time=${start}&end_time=${now}&no_cache=1" "${token}" "${size}"

    stop_server
  done

  print_summary
}

main "$@"
