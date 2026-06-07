FROM nginx:1.27.0-alpine

COPY index.html /usr/share/nginx/html/
COPY style.css /usr/share/nginx/html/
COPY Script1.js /usr/share/nginx/html/
COPY images /usr/share/nginx/html/images/
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80