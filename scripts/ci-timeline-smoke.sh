#!/usr/bin/env bash
set -euo pipefail
API="${KADR_API_URL:-http://127.0.0.1:8080}"
MAILPIT="${MAILPIT_URL:-http://127.0.0.1:8025}"
COOKIE=/tmp/kadr-ci-cookies.txt
EMAIL="ci-tl-$(date +%s)@example.test"
PASS=longpassword99

curl -fsS -X POST "$API/api/auth/register" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASS\"}" >/dev/null

TOKEN=""
for _ in $(seq 1 45); do
  MID=$(curl -fsS "$MAILPIT/api/v1/messages" | jq -r --arg e "$EMAIL" '
    .messages[] | select(.Subject | test("подтвердите")) | select(.To[]?.Address == $e) | .ID' | head -1)
  if [ -n "$MID" ] && [ "$MID" != "null" ]; then
    TOKEN=$(curl -fsS "$MAILPIT/api/v1/message/$MID" | jq -r '.Text' | sed -n 's/.*token=\([A-Za-z0-9_-]*\).*/\1/p' | head -1)
    [ -n "$TOKEN" ] && break
  fi
  sleep 1
done
[ -n "$TOKEN" ] || { echo "no verify token for $EMAIL"; exit 1; }

curl -fsS -X POST "$API/api/auth/verify" -H 'Content-Type: application/json' -d "{\"token\":\"$TOKEN\"}" >/dev/null
curl -fsS -c "$COOKIE" -X POST "$API/api/auth/login" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASS\"}" >/dev/null

FEED=$(curl -fsS -b "$COOKIE" "$API/api/me" | jq -r '.feed_id')
[ -n "$FEED" ] && [ "$FEED" != "null" ] || { echo "no feed"; exit 1; }

FILE=$(wc -c <fixtures/in/sample.png | tr -d ' ')
HEADERS=$(mktemp)
curl -fsS -b "$COOKIE" -X POST "$API/api/uploads" \
  -H "Upload-Length: $FILE" \
  -H 'Tus-Resumable: 1.0.0' \
  -H 'Upload-Metadata: filetype cGhvdG8=' \
  -D "$HEADERS" -o /dev/null
LOC=$(grep -i '^location:' "$HEADERS" | awk '{print $2}' | tr -d '\r')
ID="${LOC##*/}"
rm -f "$HEADERS"
[ -n "$ID" ] || { echo "no upload id"; exit 1; }

curl -fsS -b "$COOKIE" -X PATCH "$API/api/uploads/$ID" \
  -H "Upload-Offset: 0" -H "Tus-Resumable: 1.0.0" \
  --data-binary @fixtures/in/sample.png >/dev/null

for _ in $(seq 1 90); do
  N=$(curl -fsS -b "$COOKIE" "$API/api/feeds/$FEED/timeline" | jq '.items | length')
  if [ "$N" -ge 1 ]; then
    echo "timeline smoke ok items=$N"
    exit 0
  fi
  sleep 2
done
echo "timeline smoke timeout"
docker compose logs --tail=120 worker media-runner 2>&1 | tail -80
exit 1
