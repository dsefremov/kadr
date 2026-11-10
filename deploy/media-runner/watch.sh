#!/bin/sh
# Single-threaded spool watcher (v1 queue concurrency = 1).
set -eu
SPOOL="${KADR_SPOOL_DIR:-/spool}"
LOCK="$SPOOL/runner.lock"

mkdir -p "$SPOOL/in" "$SPOOL/out"

while true; do
  if mkdir "$LOCK" 2>/dev/null; then
  trap 'rmdir "$LOCK" 2>/dev/null || true' EXIT
    for ready in "$SPOOL/in"/*/ready; do
      [ -f "$ready" ] || continue
      job_dir="$(dirname "$ready")"
      job_id="$(basename "$job_dir")"
      out="$SPOOL/out/$job_id"
      mkdir -p "$out"
      if [ -f "$out/done" ] || [ -f "$out/failed" ]; then
        continue
      fi
      /opt/kadr/derive.sh "$job_dir" "$out" || true
      rm -f "$ready"
    done
    rmdir "$LOCK" 2>/dev/null || true
    trap - EXIT
  fi
  sleep 1
done
