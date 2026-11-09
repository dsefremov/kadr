#!/bin/sh
set -eu

echo "=== media-runner smoke ==="
cat /opt/kadr/versions.lock || true
FFMPEG_PKG="$(dpkg-query -W -f='${Package} ${Version}' ffmpeg 2>/dev/null || true)"
VIPS_PKG="$(dpkg-query -W -f='${Package} ${Version}' libvips-tools 2>/dev/null || true)"
HEIF_PKG="$(dpkg-query -W -f='${Package} ${Version}' libheif1 2>/dev/null || true)"
echo "installed: ${FFMPEG_PKG}"
echo "installed: ${VIPS_PKG}"
echo "installed: ${HEIF_PKG}"
ffmpeg -version | head -1
vips -v 2>&1 | head -1

IN="${FIXTURES_DIR:-/fixtures/in}"
OUT="${OUTPUT_DIR:-/tmp/out}"
mkdir -p "$OUT"

HEIC="$(find "$IN" -name '*.heic' | head -1)"
MOV="$(find "$IN" -name '*.mov' | head -1)"
if [ -z "$HEIC" ] || [ -z "$MOV" ]; then
  echo "missing fixtures in $IN" >&2
  exit 1
fi

MASTER_JPEG="$OUT/master.jpg"
PREVIEW_WEBP="$OUT/preview.webp"
VIDEO_MP4="$OUT/video.mp4"

# HEIC → JPEG q85 (long side policy exercised in stage 4; here prove decode + encode).
vips copy "$HEIC" "$MASTER_JPEG[Q=85,strip]"
# WebP preview q80
vips copy "$HEIC" "$PREVIEW_WEBP[Q=80,strip]"

# MOV → H.264 MP4 (SDR toolchain check)
ffmpeg -nostdin -y -i "$MOV" \
  -map 0:v:0? -map 0:a:0? \
  -c:v libx264 -crf 23 -preset medium -vf "scale='min(1920,iw)':-2" -r 30 \
  -c:a aac -b:a 128k -movflags +faststart \
  "$VIDEO_MP4"

for f in "$MASTER_JPEG" "$PREVIEW_WEBP" "$VIDEO_MP4"; do
  test -s "$f" || { echo "empty output: $f" >&2; exit 1; }
  echo "ok: $f ($(wc -c <"$f") bytes)"
done
echo "=== smoke passed ==="
