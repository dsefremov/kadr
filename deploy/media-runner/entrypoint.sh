#!/bin/sh
set -eu
if [ "${1:-}" = "smoke" ]; then
  exec /opt/kadr/smoke.sh
fi
if [ "${1:-}" = "derive" ]; then
  exec /opt/kadr/derive.sh "$2" "$3"
fi
if [ "${1:-}" = "watch" ]; then
  exec /opt/kadr/watch.sh
fi
exec sleep infinity
