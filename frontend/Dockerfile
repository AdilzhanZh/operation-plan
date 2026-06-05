FROM node:22-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

ARG VITE_API_BASE_URL=/api
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}

RUN npm run build

FROM nginx:1.27-alpine

RUN rm -f /etc/nginx/conf.d/default.conf

COPY nginx.conf /etc/nginx/templates/default.conf.template
COPY 15-resolve-dns.envsh /docker-entrypoint.d/15-resolve-dns.envsh
RUN chmod +x /docker-entrypoint.d/15-resolve-dns.envsh
COPY --from=builder /app/dist /usr/share/nginx/html

ENV NGINX_ENVSUBST_FILTER="BACKEND_URL|RESOLVER_IP"
ENV BACKEND_URL=http://backend:8080
ENV RESOLVER_IP=127.0.0.11

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
