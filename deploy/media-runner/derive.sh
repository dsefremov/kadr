#!/bin/sh
# Process one spool job directory; writes OUT_DIR/result.json
set -eu

JOB_DIR="${1:?job dir}"
OUT_DIR="${2:-$(dirname "$JOB_DIR")/../out/$(basename "$JOB_DIR")}"
mkdir -p "$OUT_DIR"
WORK="$OUT_DIR/work"
rm -rf "$WORK"
mkdir -p "$WORK"

MANIFEST="$JOB_DIR/manifest.json"
INPUT_REL="$(sed -n 's/.*"input_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$MANIFEST" | head -1)"
KIND="$(sed -n 's/.*"kind"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$MANIFEST" | head -1)"
INPUT="$JOB_DIR/$INPUT_REL"

wall_start=$(date +%s)
peak_rss=0
if [ -r /proc/self/status ]; then
  peak_rss=$(awk '/^VmRSS:/{print $2}' /proc/self/status)
fi

dims() {
  vipsheader -f width -f height "$1" 2>/dev/null | tr '\n' ' '
}

sha256_file() {
  sha256sum "$1" | awk '{print $1}'
}

fail() {
  code="$1"
  msg="$2"
  printf '{"ok":false,"error_code":"%s","error_message":"%s"}\n' "$code" "$msg" >"$OUT_DIR/result.json"
  touch "$OUT_DIR/failed"
  exit 0
}

if [ ! -f "$INPUT" ]; then
  fail missing_input "input file missing"
fi

case "$INPUT" in
  *.gif|*.GIF)
    if grep -q NETSCAPE2.0 "$INPUT" 2>/dev/null; then
      fail animated "Анимированные изображения не поддерживаются"
    fi
    ;;
esac
if grep -q ANIM "$INPUT" 2>/dev/null; then
  fail animated "Анимированные изображения не поддерживаются"
fi
if grep -q acTL "$INPUT" 2>/dev/null; then
  fail animated "Анимированные изображения не поддерживаются"
fi

derive_photo() {
  src="$1"
  flat="$WORK/flat.jpg"

  if vipsheader -f bands "$src" 2>/dev/null | grep -q '^4$'; then
    vips flatten "$src" "$flat" --background 255 2>/dev/null || vips copy "$src" "$flat"
    src="$flat"
  fi

  vips thumbnail "$src" "$WORK/thumb.jpg" 3200 --size=both --no-rotate --linear --strip --Q=85
  vips autorot "$WORK/thumb.jpg" "$OUT_DIR/master.jpg" 2>/dev/null || cp "$WORK/thumb.jpg" "$OUT_DIR/master.jpg"
  vips thumbnail "$src" "$OUT_DIR/preview-640.webp" 640 --size=both --no-rotate --strip --Q=80

  read -r mw mh <<EOF
$(dims "$OUT_DIR/master.jpg")
EOF
  read -r pw ph <<EOF
$(dims "$OUT_DIR/preview-640.webp")
EOF
  mb=$(wc -c <"$OUT_DIR/master.jpg")
  pb=$(wc -c <"$OUT_DIR/preview-640.webp")

  wall_end=$(date +%s)
  wall=$((wall_end - wall_start))
  printf '{"ok":true,"width":%s,"height":%s,"variants":[{"variant":"master","file":"master.jpg","mime":"image/jpeg","bytes":%s,"width":%s,"height":%s,"sha256_hex":"%s"},{"variant":"preview-640","file":"preview-640.webp","mime":"image/webp","bytes":%s,"width":%s,"height":%s,"sha256_hex":"%s"}],"metrics":{"wall_seconds":%s,"peak_rss_kb":%s}}\n' \
    "$mw" "$mh" "$mb" "$mw" "$mh" "$(sha256_file "$OUT_DIR/master.jpg")" "$pb" "$pw" "$ph" "$(sha256_file "$OUT_DIR/preview-640.webp")" "$wall" "$peak_rss" >"$OUT_DIR/result.json"
}

derive_video() {
  src="$1"
  dur=$(ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 "$src" 2>/dev/null || echo 0)
  dur_ms=$(awk -v d="$dur" 'BEGIN{printf "%d", d*1000}')
  if [ "$dur_ms" -gt 3600000 ]; then
    fail video_too_long "Видео длиннее 60 минут"
  fi

  transfer=$(ffprobe -v error -select_streams v:0 -show_entries stream=color_transfer -of csv=p=0 "$src" 2>/dev/null | tr -d '\r')
  vf="scale='min(1920,iw)':-2:force_original_aspect_ratio=decrease,setsar=1"
  if echo "$transfer" | grep -qiE 'smpte2084|arib-std-b67'; then
    vf="zscale=transfer=linear,tonemap=tonemap=hable:desat=0,zscale=transfer=bt709:matrix=bt709:primaries=bt709,format=yuv420p,$vf"
  fi

  ffmpeg -nostdin -y -i "$src" \
    -map 0:v:0? -map 0:a:0? \
    -c:v libx264 -crf 23 -preset medium -vf "$vf" -r 30 \
    -c:a aac -b:a 128k -movflags +faststart \
    "$OUT_DIR/video.mp4" 2>"$WORK/ffmpeg.log" || fail transcode "Не удалось обработать видео"

  ffmpeg -nostdin -y -ss 1 -i "$OUT_DIR/video.mp4" -vframes 1 \
    "$WORK/poster.png" 2>/dev/null || ffmpeg -nostdin -y -i "$OUT_DIR/video.mp4" -vframes 1 "$WORK/poster.png"
  vips thumbnail "$WORK/poster.png" "$OUT_DIR/poster.webp" 1280 --size=both --no-rotate --strip --Q=80

  vw=$(ffprobe -v error -select_streams v:0 -show_entries stream=width -of csv=p=0 "$OUT_DIR/video.mp4")
  vh=$(ffprobe -v error -select_streams v:0 -show_entries stream=height -of csv=p=0 "$OUT_DIR/video.mp4")
  vb=$(wc -c <"$OUT_DIR/video.mp4")
  pb=$(wc -c <"$OUT_DIR/poster.webp")
  read -r pw ph <<EOF
$(dims "$OUT_DIR/poster.webp")
EOF

  wall_end=$(date +%s)
  wall=$((wall_end - wall_start))
  printf '{"ok":true,"width":%s,"height":%s,"duration_ms":%s,"variants":[{"variant":"video","file":"video.mp4","mime":"video/mp4","bytes":%s,"width":%s,"height":%s,"sha256_hex":"%s"},{"variant":"poster","file":"poster.webp","mime":"image/webp","bytes":%s,"width":%s,"height":%s,"sha256_hex":"%s"}],"metrics":{"wall_seconds":%s,"peak_rss_kb":%s}}\n' \
    "$vw" "$vh" "$dur_ms" "$vb" "$vw" "$vh" "$(sha256_file "$OUT_DIR/video.mp4")" "$pb" "$pw" "$ph" "$(sha256_file "$OUT_DIR/poster.webp")" "$wall" "$peak_rss" >"$OUT_DIR/result.json"
}

case "$KIND" in
  photo) derive_photo "$INPUT" ;;
  video) derive_video "$INPUT" ;;
  *) fail unsupported "Неизвестный тип медиа" ;;
esac

rm -rf "$WORK"
touch "$OUT_DIR/done"
