#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

export COMPOSE_PROFILES=dev

mysql_dsn_default='kadr:kadr-dev@tcp(mysql:3306)/kadr?parseTime=true&charset=utf8mb4&loc=UTC&timeout=5s&multiStatements=true'
export KADR_MYSQL_DSN="${KADR_MYSQL_DSN:-$mysql_dsn_default}"

docker compose build

# Start DB first; some cloud VMs block container→container TCP — fall back to bridge gateway + published port.
docker compose up -d mysql mailpit
docker compose up -d --wait mysql

GW=""
if ! docker run --rm --network kadr_internal alpine:3.20 nc -zv -w 3 mysql 3306 >/dev/null 2>&1; then
  GW="$(docker network inspect kadr_internal -f '{{(index .IPAM.Config 0).Gateway}}')"
  echo "mysql hostname unreachable from network; using bridge gateway ${GW}:3306"
  export KADR_MYSQL_DSN="kadr:kadr-dev@tcp(${GW}:3306)/kadr?parseTime=true&charset=utf8mb4&loc=UTC&timeout=5s&multiStatements=true"
fi

if [ -n "${GW}" ] || ! docker run --rm --network kadr_internal alpine:3.20 nc -zv -w 3 mailpit 1025 >/dev/null 2>&1; then
  GW="${GW:-$(docker network inspect kadr_internal -f '{{(index .IPAM.Config 0).Gateway}}')}"
  echo "mailpit hostname unreachable from network; using bridge gateway ${GW}:1025 for SMTP"
  export KADR_SMTP_HOST="${GW}"
  export KADR_SMTP_PORT=1025
fi

docker compose run --rm migrate
docker compose up -d api worker proxy
# api/worker run as uid 10001; fresh named volumes are root-owned.
docker compose exec -T -u root api sh -c 'chown -R 10001:10001 /data'

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
  docker compose logs api mysql
  exit 1
fi

curl -fsS "http://127.0.0.1:${API_PORT}/api/health" | tee /tmp/kadr-api-health.json

echo "Auth register smoke..."
curl -fsS -X POST "http://127.0.0.1:${API_PORT}/api/auth/register" \
  -H 'Content-Type: application/json' \
  -d '{"email":"ci-smoke@example.test","password":"longpassword99"}' | tee /tmp/kadr-register.json

if curl -fsS "http://127.0.0.1:${PORT}/health" >/dev/null 2>&1; then
  echo "reverse proxy ok on :${PORT}"
else
  echo "reverse proxy not reachable on :${PORT} (continuing; e2e may use dev preview)"
fi

docker compose run --rm --no-deps -e FIXTURES_DIR=/fixtures/in media-runner smoke

./scripts/measure-memory.sh | tee /tmp/kadr-memory.txt

if [ "${KADR_KEEP_UP:-}" != "1" ]; then
  docker compose down -v
fi
