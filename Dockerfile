# Build binary with CGO support for SQLite
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install C compile toolchain required for sqlite3
RUN apk add --no-cache gcc musl-dev

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy application code
COPY . .

# Build statically linked CGO binary (linking musl statically)
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s -extldflags '-static'" \
    -o /app/eko .

# Distroless Minimal Security Image
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/eko /app/eko

# Run as non-root user (distroless default nonroot UID)
USER 65532:65532

EXPOSE 8080

ENTRYPOINT ["/app/eko"]