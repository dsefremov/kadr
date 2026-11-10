#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

JOB="ci-$(date +%s)"
IN="$ROOT/.tmp-spool-in/$JOB"
OUT="$ROOT/.tmp-spool-out/$JOB"
mkdir -p "$IN" "$OUT"
chmod -R a+rwX "$IN" "$OUT"
cp fixtures/in/sample.png "$IN/input.png"
printf '{"job_id":"%s","kind":"photo","input_name":"input.png"}\n' "$JOB" >"$IN/manifest.json"
touch "$IN/ready"

docker compose run --rm --no-deps \
  -v "$IN:/job:ro" \
  -v "$OUT:/out" \
  media-runner derive /job /out

test -s "$OUT/master.jpg"
test -s "$OUT/preview-640.webp"
test -s "$OUT/result.json"
grep -q '"ok":true' "$OUT/result.json"
echo "media derive pipeline ok"

rm -rf "$ROOT/.tmp-spool-in" "$ROOT/.tmp-spool-out"
