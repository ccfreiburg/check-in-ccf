# ── Stage 1: Build frontend ────────────────────────────────────────────────
FROM node:22-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
# Disable dev-only HTTPS plugin; the container is behind a TLS-terminating proxy.
RUN VITE_HTTPS=false npm run build

# ── Stage 2: Build backend ─────────────────────────────────────────────────
FROM golang:1.24-alpine AS backend-builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
# glebarez/sqlite is pure-Go — no CGO needed.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server

# ── Stage 3: Runtime ───────────────────────────────────────────────────────
FROM alpine:3.21
# ca-certificates: required for outbound HTTPS (push notifications to FCM/APNs)
# tzdata: required for correct local date in check-in records
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=backend-builder /app/server .
COPY --from=frontend-builder /app/dist ./static

# Pre-create volume subdirectories with correct ownership.
RUN mkdir -p /data/db /data/logs

# ── Runtime environment defaults ───────────────────────────────────────────
# All sensitive values (CT_API_TOKEN, JWT_SECRET, ADMIN_PASSWORD, VAPID_*, …)
# must be injected at runtime via --env / --env-file / Docker secrets.
ENV FRONTEND_DIST_PATH=/app/static \
    DB_PATH=/data/db/checkin.db \
    LOG_FILE=/data/logs/app.log \
    BACKEND_API_PORT=8080

# /data is the single persistent volume:
#   /data/db/   → SQLite database
#   /data/logs/ → application log file
VOLUME /data

EXPOSE 8080
CMD ["/app/server"]
