#!/usr/bin/env bash
# Docker on cloud agents: user may lack group membership; use sudo -n sg docker when needed.
# sg(1) drops unlisted env — forward compose-related variables explicitly.

docker_wrap_env_prefix() {
  local out=()
  local names=(
    COMPOSE_PROFILES
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
  if sg docker -c "$cmd" </dev/null 2>/dev/null; then
    return 0
  fi
  sudo -n sg docker -c "$cmd"
}

docker_wrap() {
  if docker info >/dev/null 2>&1; then
    docker "$@"
    return
  fi
  local prefix
  prefix="$(docker_wrap_env_prefix)"
  local inner="${prefix}docker $(printf '%q ' "$@")"
  _sg_docker "$inner"
}

compose_wrap() {
  docker_wrap compose "$@"
}
