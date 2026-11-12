#!/bin/sh
set -eu
UP="${KADR_API_UPSTREAM:-http://api:8080}"
UP="${UP%/}"
sed "s|__API_UPSTREAM__|${UP}|g" /etc/nginx/templates/default.conf.template \
  > /etc/nginx/conf.d/default.conf
exec nginx -g 'daemon off;'
