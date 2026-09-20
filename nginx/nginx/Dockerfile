FROM nginx:alpine

RUN apk --no-cache add ca-certificates tzdata

RUN rm /etc/nginx/conf.d/default.conf

COPY nginx/nginx.conf /etc/nginx/nginx.conf

EXPOSE ${NGINX_HTTP_PORT} ${NGINX_HTTPS_PORT}

CMD ["nginx", "-g", "daemon off;"]