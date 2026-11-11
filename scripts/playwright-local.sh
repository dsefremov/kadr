#!/usr/bin/env bash
# Full Playwright + axe against proxy :8088 (same wiring as docker-integration job).
# Local dev only — does not change GitHub Actions workflows.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
mkdir -p "${ROOT}/tmp"
# shellcheck source=scripts/docker-wrap.sh
source "${ROOT}/scripts/docker-wrap.sh"

export COMPOSE_PROFILES=dev
export KADR_KEEP_UP=1
export KADR_HTTP_PORT="${KADR_HTTP_PORT:-8088}"

# Free proxy port so nginx binds :8088 (vite preview fallback is CI-only).
if PIDS="$(lsof -i :"${KADR_HTTP_PORT}" -t 2>/dev/null || true)"; then
  if [ -n "$PIDS" ]; then
    echo "Stopping listeners on :${KADR_HTTP_PORT}: $PIDS"
    kill -9 $PIDS || true
  fi
fi

"${ROOT}/scripts/ci-integration.sh"

"${ROOT}/scripts/drain-dev-job-queue.sh"

PORT="${KADR_HTTP_PORT}"
if ! curl -fsS "http://127.0.0.1:${PORT}/health" >/dev/null 2>&1; then
  echo "proxy health failed on :${PORT}" >&2
  # shellcheck source=scripts/docker-wrap.sh
  source "${ROOT}/scripts/docker-wrap.sh"
  compose_wrap ps
  exit 1
fi

cd "${ROOT}/web"
export PLAYWRIGHT_BASE_URL="http://127.0.0.1:${PORT}"
export MAILPIT_URL="${MAILPIT_URL:-http://127.0.0.1:8025}"
npm run test:e2e 2>&1 | tee "${ROOT}/tmp/kadr-playwright.log"
