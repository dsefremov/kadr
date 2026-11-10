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
for _ in $(seq 1 30); do
  TOKEN=$(curl -fsS "$MAILPIT/api/v1/messages" | python3 -c "
import json,sys
m=json.load(sys.stdin)
for msg in m.get('messages',[]):
  if 'подтвердите' in msg.get('Subject','') and '$EMAIL' in str(msg.get('To','')):
    import re
    body=curl_get(msg['ID']) if False else ''
" 2>/dev/null || true)
  MSG=$(curl -fsS "$MAILPIT/api/v1/messages" | python3 -c "
import json,sys,re,urllib.request
m=json.load(sys.stdin)
for msg in m.get('messages',[]):
  if 'подтвердите' not in msg.get('Subject',''):
    continue
  to=msg.get('To') or []
  if not any('$EMAIL' in (t.get('Address') or '') for t in to):
    continue
  mid=msg['ID']
  full=json.load(urllib.request.urlopen('$MAILPIT/api/v1/message/'+mid))
  text=full.get('Text','')
  mt=re.search(r'token=([A-Za-z0-9_-]+)', text)
  if mt:
    print(mt.group(1))
    break
")
  TOKEN="$MSG"
  [ -n "$TOKEN" ] && break
  sleep 1
done
[ -n "$TOKEN" ] || { echo "no verify token"; exit 1; }

curl -fsS -X POST "$API/api/auth/verify" -H 'Content-Type: application/json' -d "{\"token\":\"$TOKEN\"}" >/dev/null
curl -fsS -c "$COOKIE" -X POST "$API/api/auth/login" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASS\"}" >/dev/null

FEED=$(curl -fsS -b "$COOKIE" "$API/api/me" | python3 -c "import json,sys; print(json.load(sys.stdin).get('feed_id',''))")
[ -n "$FEED" ] || { echo "no feed"; exit 1; }

FILE=$(wc -c <fixtures/in/sample.png | tr -d ' ')
LOC=$(curl -fsS -b "$COOKIE" -X POST "$API/api/uploads" \
  -H 'Upload-Length: '"$FILE" \
  -H 'Tus-Resumable: 1.0.0' \
  -H 'Upload-Metadata: filetype cGhvdG8=' \
  -D - -o /dev/null | awk '$1 ~ /(?i)^location:/ {print $2}' | tr -d '\r')
ID="${LOC##*/}"
curl -fsS -b "$COOKIE" -X PATCH "$API/api/uploads/$ID" \
  -H "Upload-Offset: 0" -H "Tus-Resumable: 1.0.0" \
  --data-binary @fixtures/in/sample.png >/dev/null

for _ in $(seq 1 90); do
  N=$(curl -fsS -b "$COOKIE" "$API/api/feeds/$FEED/timeline" | python3 -c "import json,sys; print(len(json.load(sys.stdin).get('items',[])))")
  if [ "$N" -ge 1 ]; then
    echo "timeline smoke ok items=$N"
    exit 0
  fi
  sleep 2
done
echo "timeline smoke timeout"
docker compose logs --tail=80 worker media-runner
exit 1
