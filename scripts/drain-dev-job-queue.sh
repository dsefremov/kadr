#!/usr/bin/env bash
# Dev-only: clear stale worker jobs before Playwright so verify/derive are not blocked
# behind a long queue from prior local runs. Not used in CI workflows.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/docker-wrap.sh
source "${ROOT}/scripts/docker-wrap.sh"
export COMPOSE_PROFILES="${COMPOSE_PROFILES:-dev}"
: "${COMPOSE_FILE:=${ROOT}/docker-compose.yml}"

compose_wrap exec mysql mysql -u kadr -pkadr-dev kadr -e "DELETE FROM jobs WHERE state = 'pending';" >/dev/null
echo "dev job queue drained (pending cleared)"
