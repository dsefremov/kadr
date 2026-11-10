#!/bin/sh
# Single-threaded spool watcher (v1: one job at a time, no lock file).
set -eu
SPOOL="${KADR_SPOOL_DIR:-/spool}"
mkdir -p "$SPOOL/in" "$SPOOL/out"

busy=0
while true; do
  if [ "$busy" -eq 0 ]; then
    for ready in "$SPOOL/in"/*/ready; do
      [ -f "$ready" ] || continue
      job_dir=$(dirname "$ready")
      job_id=$(basename "$job_dir")
      out="$SPOOL/out/$job_id"
      mkdir -p "$out"
      if [ -f "$out/done" ] || [ -f "$out/failed" ]; then
        rm -f "$ready"
        continue
      fi
      busy=1
      /opt/kadr/derive.sh "$job_dir" "$out" || true
      rm -f "$ready"
      busy=0
      break
    done
  fi
  sleep 1
done
