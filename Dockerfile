# ========= BUILD FRONTEND =========
FROM --platform=$BUILDPLATFORM node:24-alpine AS frontend-build

WORKDIR /frontend

# Add version for the frontend build
ARG APP_VERSION=dev
ENV VITE_APP_VERSION=$APP_VERSION

COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY frontend/ ./

# Copy .env file (with fallback to .env.production.example)
RUN if [ ! -f .env ]; then \
  if [ -f .env.production.example ]; then \
  cp .env.production.example .env; \
  fi; \
  fi

RUN pnpm build

# ========= BUILD BACKEND =========
# Backend build stage
FROM --platform=$BUILDPLATFORM golang:1.26.3 AS backend-build

# Set working directory
WORKDIR /app

# Install Go dependencies
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Create required directories for embedding
RUN mkdir -p /app/ui/build

# Copy frontend build output for embedding
COPY --from=frontend-build /frontend/dist /app/ui/build

COPY backend/ ./

# Compile the backend
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
RUN CGO_ENABLED=0 \
  GOOS=$TARGETOS \
  GOARCH=$TARGETARCH \
  go build -o /app/main ./cmd


# ========= RUNTIME =========
# In this final image we ship only what the running app actually needs:
#   - Build-only tooling (compilers, codegen, key fetchers) stays in the earlier
#     stages above and never reaches here.
#   - Anything pulled in for a single build step is purged within the same layer.
#   - We keep this tight because every extra binary widens the attack surface and
#     adds another CVE to track.
FROM debian:bookworm-slim

# Add version metadata to runtime image
ARG APP_VERSION=dev
ARG TARGETARCH
LABEL org.opencontainers.image.version=$APP_VERSION
ENV APP_VERSION=$APP_VERSION
ENV CONTAINER_ARCH=$TARGETARCH

# Set production mode for Docker containers
ENV ENV_MODE=production

# ========= Install runtime apt packages in a single layer =========
# The internal store is SQLite embedded in the app binary — no database server.
# ca-certificates for TLS to remote storage, gosu to drop root, rclone for the
# rclone storage backend.
RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
      ca-certificates gosu rclone; \
    rm -rf /var/lib/apt/lists/*

# Create non-root user for the main application process
RUN useradd -r -s /usr/sbin/nologin -u 65532 dockvol

WORKDIR /app

# Copy app binary
COPY --from=backend-build /app/main .

# Expose the binary as the `dockvol` command on PATH (e.g. `dockvol healthcheck`)
RUN ln -s /app/main /usr/local/bin/dockvol

# Copy UI files
COPY --from=backend-build /app/ui/build ./ui/build

# Bake .env.example as /.env so the binary has defaults when no env file is
# mounted. The backend looks for .env at the parent of cwd (= /app), i.e. /.
# Real env vars (-e, compose, k8s) take precedence — godotenv.Load does not
# overwrite already-set variables.
COPY .env.example /.env

# Create startup script
COPY <<EOF /app/start.sh
#!/bin/bash
set -e

# Generate runtime configuration for frontend
echo "Generating runtime configuration..."

# Detect if email is configured (both SMTP_HOST and DOCKVOL_URL must be set)
if [ -n "\${SMTP_HOST:-}" ] && [ -n "\${DOCKVOL_URL:-}" ]; then
  IS_EMAIL_CONFIGURED="true"
else
  IS_EMAIL_CONFIGURED="false"
fi

cat > /app/ui/build/runtime-config.js <<JSEOF
// Runtime configuration injected at container startup
// This file is generated dynamically and should not be edited manually
window.__RUNTIME_CONFIG__ = {
  GITHUB_CLIENT_ID: '\${GITHUB_CLIENT_ID:-}',
  GOOGLE_CLIENT_ID: '\${GOOGLE_CLIENT_ID:-}',
  IS_EMAIL_CONFIGURED: '\$IS_EMAIL_CONFIGURED',
  CLOUDFLARE_TURNSTILE_SITE_KEY: '\${CLOUDFLARE_TURNSTILE_SITE_KEY:-}',
  CONTAINER_ARCH: '\${CONTAINER_ARCH:-unknown}'
};
JSEOF

# The internal store is a SQLite file under /dockvol-data, created by the app on
# first boot. Make the data directory owned by the non-root app user so it can
# create the DB, backups and temp files. Non-recursive so startup stays O(1)
# regardless of how many backups the volume holds; the app creates and owns its
# own subdirectories (backups, temp) as the dockvol user.
echo "Setting up data directory permissions..."
mkdir -p /dockvol-data
chown dockvol:dockvol /dockvol-data

# The app runs as non-root but must read the mounted Docker socket to discover
# containers and spawn the tar-sidecar. The socket's owning group GID varies by
# host, so match it at runtime: reuse the existing group for that GID or create
# one, then add dockvol to it before dropping privileges.
if [ -S /var/run/docker.sock ]; then
  SOCKET_GID=\$(stat -c '%g' /var/run/docker.sock)
  SOCKET_GROUP=\$(getent group "\$SOCKET_GID" | cut -d: -f1)
  if [ -z "\$SOCKET_GROUP" ]; then
    SOCKET_GROUP=dockersock
    groupadd -g "\$SOCKET_GID" "\$SOCKET_GROUP"
  fi
  usermod -aG "\$SOCKET_GROUP" dockvol
  echo "Granted dockvol access to Docker socket (group \$SOCKET_GROUP, gid \$SOCKET_GID)"
else
  echo "Warning: /var/run/docker.sock not found - container discovery will fail. Mount it with -v /var/run/docker.sock:/var/run/docker.sock"
fi

echo "Starting DockVol application..."

exec gosu dockvol ./main
EOF

LABEL org.opencontainers.image.source="https://github.com/mavnezz/dockvol"

RUN chmod +x /app/start.sh

EXPOSE 4005

# Liveness probe: the runtime image ships no wget/curl, so the binary checks
# itself against the dependency-free /system/version endpoint.
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
  CMD ["dockvol", "healthcheck"]

# Persistent data: SQLite metadata DB, backups, temp files
VOLUME ["/dockvol-data"]

ENTRYPOINT ["/app/start.sh"]
CMD []
