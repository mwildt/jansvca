# syntax=docker/dockerfile:1
#
# Image for the web app (/app, the BFF). It builds the Lit SPA (Vite + TS) and
# the Go BFF, then serves the SPA and proxies /api/* to the service container.

# ---- Frontend build stage ------------------------------------------------
FROM node:22-alpine AS frontend
WORKDIR /fe
COPY app/frontend/package.json app/frontend/package-lock.json* ./
RUN npm ci --no-audit --no-fund || npm install --no-audit --no-fund
COPY app/frontend/ ./
RUN npm run build

# ---- Go build stage ------------------------------------------------------
FROM golang:1.27 AS builder
WORKDIR /src
COPY app/go.mod ./app/go.mod
RUN cd app && go mod download
COPY app/ ./app/
RUN cd app && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" \
    -o /out/jansvca-app ./cmd/jansvca-app

# ---- Runtime stage -------------------------------------------------------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
COPY --from=builder /out/jansvca-app /usr/local/bin/jansvca-app
COPY --from=frontend /dist /app/dist

ENV JANSVCA_ADDR=:8080 \
    JANSVCA_SPA_DIR=/app/dist
EXPOSE 8080
USER app
ENTRYPOINT ["/usr/local/bin/jansvca-app"]
