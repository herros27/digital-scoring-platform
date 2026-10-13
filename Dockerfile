# ==========================================
# Stage 1: Build binary
# ==========================================
FROM golang:alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Cache go modules download
COPY go.mod go.sum ./
RUN go mod download

# Copy source files
COPY . .

# Compile optimized static binary
# -ldflags="-s -w" removes symbol table and debug information for ~30% smaller binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/server ./cmd/server

# ==========================================
# Stage 2: Minimal runtime image (~20MB)
# ==========================================
FROM alpine:3.21

# Install CA certificates and create unprivileged user
RUN apk --no-cache add ca-certificates tzdata \
    && addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy compiled binary from builder
COPY --from=builder /app/bin/server /app/server

# Copy runtime configuration & static assets
COPY --from=builder /app/rulesets /app/rulesets
COPY --from=builder /app/docs /app/docs

# Security: run as non-root user
USER appuser

EXPOSE 8080

# Container healthcheck
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/docs || exit 1

ENTRYPOINT ["/app/server"]
