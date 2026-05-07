# syntax=docker/dockerfile:1

FROM node:22-bookworm AS frontend
WORKDIR /build
COPY frontend/package.json frontend/yarn.lock ./
RUN corepack enable && yarn install --frozen-lockfile
COPY frontend/ .
RUN yarn build

FROM golang:1.24-bookworm AS backend
ENV GOFLAGS=-mod=mod
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc libc6-dev \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o /gelift-tracker .

FROM debian:bookworm-slim AS runtime
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates libsqlite3-0 \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=backend /gelift-tracker /usr/local/bin/gelift-tracker
COPY --from=frontend /build/dist ./dist
ENV STATIC_DIR=/app/dist \
    DATABASE=/data/gelift.db \
    PORT=1323
EXPOSE 1323
VOLUME ["/data"]
ENTRYPOINT ["/usr/local/bin/gelift-tracker"]
