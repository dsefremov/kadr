#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/docker-wrap.sh
source "${ROOT}/scripts/docker-wrap.sh"
export COMPOSE_PROFILES="${COMPOSE_PROFILES:-dev}"
ids="$(compose_wrap ps -q)"
if [ -z "$ids" ]; then
  echo "no containers" >&2
  exit 1
fi
printf "service\tidle_rss_mib\n"
while read -r id; do
  [ -z "$id" ] && continue
  name="$(docker_wrap inspect --format '{{.Name}}' "$id" | sed 's#^/##')"
  case "$name" in
    *-mysql-*) svc=mysql ;;
    *-api-*) svc=api ;;
    *-worker-*) svc=worker ;;
    *-proxy-*) svc=proxy ;;
    *-mailpit-*) svc=mailpit ;;
    *-media-runner-*) svc=media-runner ;;
    *) svc="$name" ;;
  esac
  mem="$(docker_wrap stats --no-stream --format '{{.MemUsage}}' "$id" | awk -F'/' '{print $1}' | tr -d ' ' | sed 's/MiB//')"
  printf "%s\t%s\n" "$svc" "$mem"
done <<<"$ids"
