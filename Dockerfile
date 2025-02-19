# Build stage
FROM golang:1 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main main.go

# Final stage
FROM debian:bookworm-slim

# Install necessary system packages
RUN apt-get update && apt-get install -y ca-certificates tzdata curl && rm -rf /var/lib/apt/lists/*

# Create a non-root user for security
RUN groupadd -r app && useradd -r -g app app

WORKDIR /app
COPY --from=builder /app/main .

# Set proper ownership
RUN chown -R app:app /app

# Switch to non-root user
USER app:app

# Default command
CMD ["./main"]

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:${PORT:-8080}/ || exit 1
