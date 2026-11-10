#!/bin/sh
# Single-threaded spool watcher (v1: one job at a time, no lock file).
set -eu
SPOOL="${KADR_SPOOL_DIR:-/spool}"
mkdir -p "$SPOOL/in" "$SPOOL/out"

busy=0
while true; do
  if [ "$busy" -eq 0 ]; then
    ready="$(find "$SPOOL/in" -mindepth 2 -maxdepth 2 -name ready -type f 2>/dev/null | head -n 1)"
    if [ -n "$ready" ] && [ -f "$ready" ]; then
      job_dir=$(dirname "$ready")
      job_id=$(basename "$job_dir")
      out="$SPOOL/out/$job_id"
      mkdir -p "$out"
      chmod -R a+rwX "$job_dir" "$out" 2>/dev/null || true
      if [ -f "$out/done" ] || [ -f "$out/failed" ] || [ -f "$out/result.json" ]; then
        rm -f "$ready"
      else
        busy=1
        echo "media-runner: derive job_id=$job_id" >&2
        /opt/kadr/derive.sh "$job_dir" "$out" || true
        rm -f "$ready"
        busy=0
      fi
    fi
  fi
  sleep 1
done
