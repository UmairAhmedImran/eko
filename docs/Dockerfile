# ==============================================================================
# Stage 1: Build Docusaurus v3 Static Site
# ==============================================================================
FROM node:20-alpine AS builder

WORKDIR /app

# Copy dependency manifests
COPY package*.json ./

# Install dependencies cleanly
RUN npm ci

# Copy Docusaurus source code
COPY . .

# Build static website into /app/build
RUN npm run build

# ==============================================================================
# Stage 2: Production Nginx Server (WSO2 Choreo Compatible)
# ==============================================================================
FROM nginx:alpine AS runner

# Create non-root user matching WSO2 Choreo UID constraints (10000 - 20000 range)
ARG USER_ID=10014
ARG GROUP_ID=10014

RUN addgroup -g ${GROUP_ID} choreo && \
    adduser -u ${USER_ID} -G choreo -s /bin/sh -D choreo

# Copy custom Nginx configuration
COPY nginx.conf /etc/nginx/nginx.conf

# Copy static build output from builder stage
COPY --from=builder /app/build /usr/share/nginx/html

# Grant required permissions to non-root user for temp directories & static files
RUN mkdir -p /tmp/client_temp /tmp/proxy_temp_path /tmp/fastcgi_temp /tmp/uwsgi_temp /tmp/scgi_temp && \
    chown -R choreo:choreo /usr/share/nginx/html && \
    chown -R choreo:choreo /var/cache/nginx && \
    chown -R choreo:choreo /var/log/nginx && \
    chown -R choreo:choreo /etc/nginx && \
    chown -R choreo:choreo /tmp

# Switch to non-root user for security compliance
USER 10014

# Expose HTTP port for WSO2 Choreo router
EXPOSE 8080

# Health check probe
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/healthz || exit 1

# Start Nginx in foreground
CMD ["nginx", "-g", "daemon off;"]
