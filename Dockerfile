# ── Stage 1: builder ──────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Download dependencies first so Docker can cache this layer independently of
# source changes.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source tree.
COPY . .

# Build a statically-linked binary.  CGO_ENABLED=0 is safe because we are
# using the pure-Go modernc.org/sqlite driver.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /clandar ./cmd/server

# ── Stage 2: runtime ──────────────────────────────────────────────────────────
# Pin to a specific Alpine release for reproducible builds.
FROM alpine:3.20

# ca-certificates  — outbound HTTPS calls to holiday APIs
# tzdata           — correct timezone handling in date calculations
# wget             — used by Docker/Render health checks
RUN apk add --no-cache ca-certificates tzdata wget

# Run as a non-root user for security.
RUN addgroup -S clandar && adduser -S -G clandar clandar

WORKDIR /app

# Copy the compiled binary from the builder stage.
COPY --from=builder /clandar       /app/clandar

# Copy static assets and the database schema.
COPY --from=builder /build/web     /app/web
COPY --from=builder /build/db      /app/db

# The SQLite file lives in /data.
# • Local Docker: backed by a named volume (docker-compose.yml)
# • Render:       backed by a Persistent Disk (render.yaml)
RUN mkdir -p /data && chown -R clandar:clandar /data /app

USER clandar

# Render injects $PORT at runtime; the app reads it via config.Load().
# EXPOSE is documentation only — it does not pin the port.
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=25s --retries=3 \
  CMD wget -qO- http://localhost:${PORT:-8080}/health || exit 1

CMD ["/app/clandar"]
