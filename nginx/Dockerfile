FROM nginx:1.29-alpine

RUN apk add --no-cache gettext

COPY entrypoint.sh /entrypoint.sh
COPY nginx.conf.template /etc/nginx/templates/cats.conf.template

RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
