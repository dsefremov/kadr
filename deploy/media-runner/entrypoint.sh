#!/bin/sh
set -eu
if [ "${1:-}" = "smoke" ]; then
  exec /opt/kadr/smoke.sh
fi
exec sleep infinity
