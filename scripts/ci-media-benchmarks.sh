#!/usr/bin/env bash
# Peak memory + wall time for representative derive jobs (768 MiB cap in compose).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
FIX="$ROOT/fixtures/in"
OUT_ROOT="$ROOT/.tmp-bench-out"
RESULTS="$OUT_ROOT/results.tsv"
mkdir -p "$OUT_ROOT"
printf "asset\tkind\twall_seconds\tpeak_rss_kb\tpeak_mib\n" >"$RESULTS"

run_derive() {
  local label="$1" kind="$2" src="$3"
  local jobdir="$OUT_ROOT/job-$label"
  local outdir="$OUT_ROOT/out-$label"
  rm -rf "$jobdir" "$outdir"
  mkdir -p "$jobdir" "$outdir"
  ext="${src##*.}"
  cp "$src" "$jobdir/input.$ext"
  printf '{"job_id":"%s","kind":"%s","input_name":"input.%s"}\n' "$label" "$kind" "$ext" >"$jobdir/manifest.json"

  docker compose run --rm --no-deps \
    -v "$jobdir:/job:ro" \
    -v "$outdir:/out" \
    --entrypoint /opt/kadr/derive.sh \
    media-runner /job /out

  test -s "$outdir/result.json"
  grep -q '"ok":true' "$outdir/result.json"
  wall="$(jq -r '.metrics.wall_seconds // 0' "$outdir/result.json")"
  peak_kb="$(jq -r '.metrics.peak_rss_kb // 0' "$outdir/result.json")"
  peak_mib="$(awk -v k="$peak_kb" 'BEGIN{printf "%.1f", k/1024}')"
  printf "%s\t%s\t%s\t%s\t%s\n" "$label" "$kind" "$wall" "$peak_kb" "$peak_mib" | tee -a "$RESULTS"
  echo "benchmark ok: $label peak=${peak_mib}MiB wall=${wall}s"
}

test -f "$FIX/sample.heic"
test -f "$FIX/photo-24mp.jpg"
test -f "$FIX/benchmark-hdr-hlg.mov"

run_derive "heic" photo "$FIX/sample.heic"
run_derive "photo-24mp" photo "$FIX/photo-24mp.jpg"
run_derive "hdr-hlg-1080p30" video "$FIX/benchmark-hdr-hlg.mov"

# Fail CI if peak exceeds 768 MiB cap (with small headroom for metrics noise).
limit_kb=$((768 * 1024))
while IFS=$'\t' read -r label _ _ pk _; do
  [ "$label" = asset ] && continue
  [ "$pk" -le "$limit_kb" ] || { echo "peak ${pk}KiB exceeds 768MiB cap" >&2; exit 1; }
done <"$RESULTS"

cp "$RESULTS" /tmp/kadr-benchmarks.tsv
cat /tmp/kadr-benchmarks.tsv
