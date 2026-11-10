#!/bin/sh
set -eu
if [ "${1:-}" = "smoke" ]; then
  exec /opt/kadr/smoke.sh
fi
if [ "${1:-}" = "derive" ]; then
  exec /opt/kadr/derive.sh "$2" "$3"
fi
if [ "${1:-}" = "watch" ]; then
  SPOOL="${KADR_SPOOL_DIR:-/spool}"
  mkdir -p "$SPOOL/in" "$SPOOL/out"
  chmod -R a+rwX "$SPOOL" 2>/dev/null || true
  exec /opt/kadr/watch.sh
fi
exec sleep infinity
