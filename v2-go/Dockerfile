# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files first (cache-friendly layer ordering)
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build binary with version info and stripped symbols
ARG VERSION=2.0.0
ARG BUILD_TIME
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -o kudig ./cmd/kudig

# Build kubectl plugin
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -o kubectl-kudig ./cmd/kubectl-kudig

# Prepare rules
RUN mkdir -p /app/rules-out && cp /app/rules/*.yaml /app/rules-out/ 2>/dev/null || true

# Runtime stage — minimal distroless-style image
FROM alpine:3.24

WORKDIR /app

# Install runtime dependencies and create non-root user
RUN apk add --no-cache \
    ca-certificates \
    bash \
    curl \
    jq && \
    addgroup -S kudig && adduser -S kudig -G kudig && \
    mkdir -p /home/kudig/.kudig && chown -R kudig:kudig /home/kudig

# Copy binaries from builder
COPY --from=builder /app/kudig /usr/local/bin/kudig
COPY --from=builder /app/kubectl-kudig /usr/local/bin/kubectl-kudig

# Copy default rules
COPY --from=builder /app/rules-out/ /app/rules/

# Set environment
ENV PATH="/app:${PATH}" \
    KUDIG_HOME="/home/kudig/.kudig"

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD kudig --version || exit 1

# Run as non-root user
USER kudig

# Default command
ENTRYPOINT ["kudig"]
CMD ["--help"]
