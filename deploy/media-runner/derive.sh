#!/bin/sh
# Process one spool job directory; writes OUT_DIR/result.json
set -eu

. /opt/kadr/peak-metrics.sh

export VIPS_CONCURRENCY=1
export VIPS_CACHE_MAX="${VIPS_CACHE_MAX:-64}"
export VIPS_DISC_THRESHOLD="${VIPS_DISC_THRESHOLD:-64m}"
export OMP_NUM_THREADS=1
export X264_THREADS=1
export FFMPEG_THREAD_QUEUE_SIZE="${FFMPEG_THREAD_QUEUE_SIZE:-8}"

JOB_DIR="${1:?job dir}"
OUT_DIR="${2:-$(dirname "$JOB_DIR")/../out/$(basename "$JOB_DIR")}"
mkdir -p "$OUT_DIR"
WORK="/tmp/kadr-derive-$$"
rm -rf "$WORK"
mkdir -p "$WORK"

fail() {
  code="$1"
  msg="$2"
  printf '{"ok":false,"error_code":"%s","error_message":"%s"}\n' "$code" "$msg" >"$OUT_DIR/result.json"
  touch "$OUT_DIR/failed"
  rm -rf "$WORK"
  exit 0
}

on_exit() {
  rc=$?
  rm -rf "$WORK" 2>/dev/null || true
  if [ ! -f "$OUT_DIR/result.json" ]; then
    fail internal "derive aborted (exit $rc)"
  fi
}
trap on_exit EXIT

MANIFEST="$JOB_DIR/manifest.json"
INPUT_REL="$(sed -n 's/.*"input_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$MANIFEST" | head -1)"
KIND="$(sed -n 's/.*"kind"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$MANIFEST" | head -1)"
INPUT="$JOB_DIR/$INPUT_REL"

wall_start=$(date +%s)
cgroup_peak_start=0
if cgroup_kb="$(peak_kb_from_cgroup 2>/dev/null)"; then
  cgroup_peak_start="$cgroup_kb"
fi
proc_peak_start="$(peak_kb_from_proc 2>/dev/null || echo 0)"
proc_peak_start="${proc_peak_start:-0}"

finalize_metrics() {
  wall_end=$(date +%s)
  wall=$((wall_end - wall_start))
  proc_peak="$(peak_kb_from_proc 2>/dev/null || echo 0)"
  proc_peak="${proc_peak:-0}"
  cgroup_peak="$cgroup_peak_start"
  if cgroup_kb="$(peak_kb_from_cgroup 2>/dev/null)"; then
    cgroup_peak="$cgroup_kb"
  fi
  peak_rss_kb="$(max_kb "$proc_peak" "$cgroup_peak")"
  peak_rss_kb="$(max_kb "$peak_rss_kb" "$proc_peak_start")"
  export DERIVE_WALL="$wall"
  export DERIVE_PEAK_RSS_KB="$peak_rss_kb"
}

dims() {
  # vipsheader accepts only one -f at a time; multiple -f keeps the last field.
  w="$(vipsheader -f width "$1" 2>/dev/null || echo 0)"
  h="$(vipsheader -f height "$1" 2>/dev/null || echo 0)"
  echo "$w $h"
}

sha256_file() {
  sha256sum "$1" | awk '{print $1}'
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

  vips thumbnail "$src" "$WORK/thumb.v" 3200 --size=both --no-rotate
  vips copy "$WORK/thumb.v" "$OUT_DIR/master.jpg[Q=85,strip]"
  vips thumbnail "$src" "$WORK/p320.v" 320 --size=both --no-rotate
  vips copy "$WORK/p320.v" "$OUT_DIR/preview-320.webp[Q=80,strip]"
  vips thumbnail "$src" "$WORK/p640.v" 640 --size=both --no-rotate
  vips copy "$WORK/p640.v" "$OUT_DIR/preview-640.webp[Q=80,strip]"
  vips thumbnail "$src" "$WORK/p1280.v" 1280 --size=both --no-rotate
  vips copy "$WORK/p1280.v" "$OUT_DIR/preview-1280.webp[Q=80,strip]"

  dim_line="$(dims "$OUT_DIR/master.jpg")"
  mw="${dim_line%% *}"
  mh="${dim_line#* }"
  mw="${mw:-0}"
  mh="${mh:-0}"
  p320_line="$(dims "$OUT_DIR/preview-320.webp")"
  p320w="${p320_line%% *}"; p320h="${p320_line#* }"
  p320w="${p320w:-0}"; p320h="${p320h:-0}"
  p640_line="$(dims "$OUT_DIR/preview-640.webp")"
  p640w="${p640_line%% *}"; p640h="${p640_line#* }"
  p640w="${p640w:-0}"; p640h="${p640h:-0}"
  p1280_line="$(dims "$OUT_DIR/preview-1280.webp")"
  p1280w="${p1280_line%% *}"; p1280h="${p1280_line#* }"
  p1280w="${p1280w:-0}"; p1280h="${p1280h:-0}"
  mb=$(wc -c <"$OUT_DIR/master.jpg")
  b320=$(wc -c <"$OUT_DIR/preview-320.webp")
  b640=$(wc -c <"$OUT_DIR/preview-640.webp")
  b1280=$(wc -c <"$OUT_DIR/preview-1280.webp")

  finalize_metrics
  printf '%s' \
    "{\"ok\":true,\"width\":$mw,\"height\":$mh,\"variants\":[" \
    "{\"variant\":\"master\",\"file\":\"master.jpg\",\"mime\":\"image/jpeg\",\"bytes\":$mb,\"width\":$mw,\"height\":$mh,\"sha256_hex\":\"$(sha256_file "$OUT_DIR/master.jpg")\"}," \
    "{\"variant\":\"preview-320\",\"file\":\"preview-320.webp\",\"mime\":\"image/webp\",\"bytes\":$b320,\"width\":$p320w,\"height\":$p320h,\"sha256_hex\":\"$(sha256_file "$OUT_DIR/preview-320.webp")\"}," \
    "{\"variant\":\"preview-640\",\"file\":\"preview-640.webp\",\"mime\":\"image/webp\",\"bytes\":$b640,\"width\":$p640w,\"height\":$p640h,\"sha256_hex\":\"$(sha256_file "$OUT_DIR/preview-640.webp")\"}," \
    "{\"variant\":\"preview-1280\",\"file\":\"preview-1280.webp\",\"mime\":\"image/webp\",\"bytes\":$b1280,\"width\":$p1280w,\"height\":$p1280h,\"sha256_hex\":\"$(sha256_file "$OUT_DIR/preview-1280.webp")\"}" \
    "],\"metrics\":{\"wall_seconds\":$DERIVE_WALL,\"peak_rss_kb\":$DERIVE_PEAK_RSS_KB}}" \
    >"$OUT_DIR/result.json"
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
  x264_params="threads=1:lookahead-threads=1:sliced-threads=0:rc-lookahead=10"

  ffmpeg -nostdin -y -threads 1 -filter_threads 1 -i "$src" \
    -map 0:v:0? -map 0:a:0? \
    -c:v libx264 -crf 23 -preset veryfast -threads 1 -x264-params "$x264_params" -vf "$vf" -r 30 \
    -c:a aac -b:a 128k -movflags +faststart \
    "$OUT_DIR/video.mp4" 2>"$WORK/ffmpeg.log" || fail transcode "Не удалось обработать видео"

  ffmpeg -nostdin -y -threads 1 -ss 1 -i "$OUT_DIR/video.mp4" -vframes 1 \
    "$WORK/poster.png" 2>/dev/null || ffmpeg -nostdin -y -i "$OUT_DIR/video.mp4" -vframes 1 "$WORK/poster.png"
  vips thumbnail "$WORK/poster.png" "$WORK/poster.v" 1280 --size=both --no-rotate
  vips copy "$WORK/poster.v" "$OUT_DIR/poster.webp[Q=80,strip]"

  vw=$(ffprobe -v error -select_streams v:0 -show_entries stream=width -of csv=p=0 "$OUT_DIR/video.mp4")
  vh=$(ffprobe -v error -select_streams v:0 -show_entries stream=height -of csv=p=0 "$OUT_DIR/video.mp4")
  vb=$(wc -c <"$OUT_DIR/video.mp4")
  pb=$(wc -c <"$OUT_DIR/poster.webp")
  poster_line="$(dims "$OUT_DIR/poster.webp")"
  pw="${poster_line%% *}"
  ph="${poster_line#* }"
  pw="${pw:-0}"
  ph="${ph:-0}"

  finalize_metrics
  printf '{"ok":true,"width":%s,"height":%s,"duration_ms":%s,"variants":[{"variant":"video","file":"video.mp4","mime":"video/mp4","bytes":%s,"width":%s,"height":%s,"sha256_hex":"%s"},{"variant":"poster","file":"poster.webp","mime":"image/webp","bytes":%s,"width":%s,"height":%s,"sha256_hex":"%s"}],"metrics":{"wall_seconds":%s,"peak_rss_kb":%s}}\n' \
    "$vw" "$vh" "$dur_ms" "$vb" "$vw" "$vh" "$(sha256_file "$OUT_DIR/video.mp4")" "$pb" "$pw" "$ph" "$(sha256_file "$OUT_DIR/poster.webp")" "$DERIVE_WALL" "$DERIVE_PEAK_RSS_KB" >"$OUT_DIR/result.json"
}

case "$KIND" in
  photo) derive_photo "$INPUT" ;;
  video) derive_video "$INPUT" ;;
  *) fail unsupported "Неизвестный тип медиа" ;;
esac

touch "$OUT_DIR/done"
