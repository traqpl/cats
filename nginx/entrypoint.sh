#!/bin/sh
set -eu

mkdir -p /etc/nginx/conf.d
cp /etc/nginx/templates/cats.conf.template /etc/nginx/conf.d/default.conf

exec nginx -g 'daemon off;'
