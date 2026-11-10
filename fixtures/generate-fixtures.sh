#!/usr/bin/env bash
# Regenerate project-owned test fixtures (CC0). Run from repo root.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/fixtures/in"
mkdir -p "$OUT"

CC0="CC0-1.0 — Public Domain Dedication (project-generated)"

echo "==> sample.mov (SDR, short smoke)"
ffmpeg -nostdin -y \
  -f lavfi -i "testsrc2=size=320x240:rate=30" \
  -f lavfi -i "sine=frequency=440:duration=2" \
  -t 2 \
  -c:v libx264 -pix_fmt yuv420p -crf 23 -preset veryfast \
  -c:a aac -b:a 128k \
  -movflags +faststart \
  "$OUT/sample.mov"

echo "==> benchmark-hdr-hlg.mov (HLG BT.2020, 1080p30, 30s — CC0)"
ffmpeg -nostdin -y \
  -f lavfi -i "testsrc2=size=1920x1080:rate=30" \
  -f lavfi -i "sine=frequency=220:duration=30" \
  -t 30 \
  -c:v libx264 -pix_fmt yuv420p10le -profile:v high422 \
  -colorspace bt2020nc -color_primaries bt2020 -color_trc arib-std-b67 \
  -crf 18 -preset veryfast \
  -c:a aac -b:a 128k \
  -movflags +faststart \
  "$OUT/benchmark-hdr-hlg.mov"

echo "==> photo-24mp.jpg (6000x4000 — CC0)"
ffmpeg -nostdin -y \
  -f lavfi -i "testsrc2=size=6000x4000:rate=1" \
  -frames:v 1 \
  -q:v 2 \
  "$OUT/photo-24mp.jpg"

echo "==> sample.png (minimal)"
printf '\x89PNG\r\n\x1a\n' >"$OUT/sample.png.tmp"
# keep existing tiny png if present; otherwise 1x1 via ffmpeg
if [ ! -s "$OUT/sample.png" ] || [ "$(wc -c <"$OUT/sample.png")" -lt 100 ]; then
  ffmpeg -nostdin -y -f lavfi -i "color=c=blue:s=64x64" -frames:v 1 -update 1 "$OUT/sample.png"
fi
rm -f "$OUT/sample.png.tmp"

echo "==> sample.heic (CC0 via media-runner heif-enc)"
if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
  docker compose -f "$ROOT/docker-compose.yml" build media-runner >/dev/null
  TMPJ="$OUT/.gen-heic-src.jpg"
  ffmpeg -nostdin -y -f lavfi -i "testsrc=size=640x480:rate=1" -frames:v 1 -q:v 3 "$TMPJ"
  docker compose -f "$ROOT/docker-compose.yml" run --rm --no-deps \
    --entrypoint /usr/bin/heif-enc \
    -v "$OUT:/out" -v "$TMPJ:/in.jpg:ro" \
    media-runner -o /out/sample.heic /in.jpg || true
  rm -f "$TMPJ"
fi
if [ ! -s "$OUT/sample.heic" ]; then
  echo "warn: sample.heic not regenerated (docker/heif-enc unavailable); keep checked-in file" >&2
fi

echo "Done. License: $CC0 for generated MOV/JPG/PNG; see fixtures/README.md for HEIC."
