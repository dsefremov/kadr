#!/bin/sh
set -eu
for f in /fixtures/in/photo-24mp.jpg /fixtures/in/sample.heic /fixtures/in/sample.mov /fixtures/in/benchmark-hdr-hlg.mov; do
  echo "===== $f"
  if [ ! -f "$f" ]; then
    echo missing
    continue
  fi
  echo "-- vipsheader"
  vipsheader -a "$f" 2>/dev/null | grep -i date | head -20 || true
  echo "-- ffprobe"
  ffprobe -v error -show_entries format_tags:stream_tags -of flat "$f" 2>/dev/null | grep -iE 'date|time|creat' | head -20 || true
done
