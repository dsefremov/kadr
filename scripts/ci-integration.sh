#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
# shellcheck source=scripts/docker-wrap.sh
source "${ROOT}/scripts/docker-wrap.sh"

export COMPOSE_PROFILES=dev

compose_wrap build

# Start DB + Mailpit (dev profile); apply bridge gateway env when hostname routing is broken.
compose_wrap up -d mysql mailpit
compose_wrap up -d --wait mysql
# shellcheck source=scripts/compose-dev-bridge-env.sh
source "${ROOT}/scripts/compose-dev-bridge-env.sh"

compose_wrap run --rm migrate
compose_wrap up -d api worker media-runner proxy
# api/worker run as uid 10001; fresh named volumes are root-owned.
compose_wrap exec -T -u root api sh -c 'chown -R 10001:10001 /data'
compose_wrap exec -T -u root worker sh -c 'chmod -R a+rwX /spool && ls -la /spool'
compose_wrap exec -T -u root media-runner sh -c 'chmod -R a+rwX /spool' 2>/dev/null || true

API_PORT=8080
PORT="${KADR_HTTP_PORT:-8088}"
echo "Waiting for API readiness on :${API_PORT} ..."
ready=0
for _ in $(seq 1 60); do
  if body="$(curl -fsS "http://127.0.0.1:${API_PORT}/ready" 2>/dev/null)" && echo "$body" | grep -q '"ready"'; then
    ready=1
    echo "$body"
    break
  fi
  sleep 2
done
if [ "$ready" -ne 1 ]; then
  echo "API not ready" >&2
  compose_wrap logs api mysql
  exit 1
fi

mkdir -p "${ROOT}/tmp"
curl -fsS "http://127.0.0.1:${API_PORT}/api/health" | tee "${ROOT}/tmp/kadr-api-health.json"

echo "Auth register smoke..."
curl -fsS -X POST "http://127.0.0.1:${API_PORT}/api/auth/register" \
  -H 'Content-Type: application/json' \
  -d '{"email":"ci-smoke@example.test","password":"longpassword99"}' | tee "${ROOT}/tmp/kadr-register.json"

if curl -fsS "http://127.0.0.1:${PORT}/health" >/dev/null 2>&1; then
  echo "reverse proxy ok on :${PORT}"
else
  echo "reverse proxy not reachable on :${PORT} (continuing; e2e may use dev preview)"
fi

compose_wrap run --rm --no-deps -e FIXTURES_DIR=/fixtures/in media-runner smoke
./scripts/ci-media-pipeline.sh
./scripts/ci-media-benchmarks.sh
./scripts/ci-timeline-smoke.sh

./scripts/measure-memory.sh | tee "${ROOT}/tmp/kadr-memory.txt"

if [ "${KADR_KEEP_UP:-}" != "1" ]; then
  compose_wrap down -v
fi
