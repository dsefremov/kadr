#!/usr/bin/env bash
# Local dev only (COMPOSE_PROFILES=dev): when container→container TCP is blocked,
# reach mysql/mailpit via the compose bridge gateway + published host ports.
# Sourced by ci-integration.sh and playwright-local.sh — not used in CI workflows.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/docker-wrap.sh
source "${ROOT}/scripts/docker-wrap.sh"

: "${COMPOSE_PROFILES:=dev}"
export COMPOSE_PROFILES

mysql_dsn_default='kadr:kadr-dev@tcp(mysql:3306)/kadr?parseTime=true&charset=utf8mb4&loc=UTC&timeout=5s&multiStatements=true'
export KADR_MYSQL_DSN="${KADR_MYSQL_DSN:-$mysql_dsn_default}"

# mysql + mailpit must be up on kadr_internal before probing.
GW=""
if ! docker_wrap run --rm --network kadr_internal alpine:3.20 nc -zv -w 3 mysql 3306 >/dev/null 2>&1; then
  GW="$(docker_wrap network inspect kadr_internal -f '{{(index .IPAM.Config 0).Gateway}}')"
  echo "mysql hostname unreachable from network; using bridge gateway ${GW}:3306"
  export KADR_MYSQL_DSN="kadr:kadr-dev@tcp(${GW}:3306)/kadr?parseTime=true&charset=utf8mb4&loc=UTC&timeout=5s&multiStatements=true"
fi

if [ -n "${GW}" ] || ! docker_wrap run --rm --network kadr_internal alpine:3.20 nc -zv -w 3 mailpit 1025 >/dev/null 2>&1; then
  GW="${GW:-$(docker_wrap network inspect kadr_internal -f '{{(index .IPAM.Config 0).Gateway}}')}"
  echo "mailpit hostname unreachable from network; using bridge gateway ${GW}:1025 for SMTP"
  export KADR_SMTP_HOST="${GW}"
  export KADR_SMTP_PORT=1025
fi

if [ -n "${GW}" ] || ! docker_wrap run --rm --network kadr_internal alpine:3.20 nc -zv -w 3 api 8080 >/dev/null 2>&1; then
  GW="${GW:-$(docker_wrap network inspect kadr_internal -f '{{(index .IPAM.Config 0).Gateway}}')}"
  echo "api hostname unreachable from network; proxy upstream http://${GW}:8080"
  export KADR_API_UPSTREAM="http://${GW}:8080"
fi
export KADR_API_UPSTREAM="${KADR_API_UPSTREAM:-http://api:8080}"
# Local e2e generates many derive jobs; poll faster only for dev scripts (not CI workflow).
export KADR_WORKER_POLL_INTERVAL="${KADR_WORKER_POLL_INTERVAL:-1s}"
