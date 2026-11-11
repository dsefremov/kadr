#!/usr/bin/env bash
# Docker on cloud agents: user may lack group membership; use sudo -n sg docker when needed.
# sg(1) drops unlisted env — forward compose-related variables explicitly.

docker_wrap_env_prefix() {
  local out=()
  local names=(
    COMPOSE_PROFILES
    COMPOSE_FILE
    KADR_MYSQL_DSN
    KADR_SMTP_HOST
    KADR_SMTP_PORT
    KADR_API_UPSTREAM
    KADR_PUBLIC_URL
    KADR_HTTP_PORT
    KADR_KEEP_UP
    KADR_WORKER_POLL_INTERVAL
    KADR_VERSION
    KADR_COMMIT
    KADR_BUILD_TIME
  )
  local n
  for n in "${names[@]}"; do
    if [ -n "${!n-}" ]; then
      out+=("${n}=$(printf '%q' "${!n}")")
    fi
  done
  printf '%s ' "${out[@]}"
}

_sg_docker() {
  local cmd=$1
  # Prefer passwordless sudo sg on agents where the user is not in group docker (sg alone prompts).
  if sudo -n sg docker -c "$cmd" </dev/null 2>/dev/null; then
    return 0
  fi
  if sg docker -c "$cmd" </dev/null 2>/dev/null; then
    return 0
  fi
  sudo -n sg docker -c "$cmd"
}

DOCKER_BIN="${DOCKER_BIN:-/usr/bin/docker}"

docker_wrap() {
  if "${DOCKER_BIN}" info >/dev/null 2>&1; then
    "${DOCKER_BIN}" "$@"
    return
  fi
  local prefix
  prefix="$(docker_wrap_env_prefix)"
  local inner="${prefix}${DOCKER_BIN} $(printf '%q ' "$@")"
  _sg_docker "$inner"
}

compose_wrap() {
  docker_wrap compose "$@"
}
