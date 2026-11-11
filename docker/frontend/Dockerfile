# Build from the repository root:
#   docker build -f docker/frontend/Dockerfile .
FROM docker.io/library/node:22-alpine AS build

WORKDIR /web

RUN corepack enable

COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY web/ ./

# Leave unset to call the API on the app's own origin at /api.
ARG VITE_API_URL
RUN pnpm build

FROM docker.io/nginxinc/nginx-unprivileged:alpine

COPY docker/frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /web/dist /usr/share/nginx/html

EXPOSE 8080
