# Stage 1: Build Go Binary
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o joblessyu-bot ./cmd/bot

# Stage 2: Final Hybrid Container (Go + Python JobSpy Runtime)
FROM python:3.11-alpine

WORKDIR /app

# Install SSL certificates, timezone, and C build toolchain for Python native extensions
RUN apk add --no-cache ca-certificates tzdata build-base libffi-dev

# Install Python scraper dependencies
COPY scraper-python/requirements.txt ./scraper-python/requirements.txt
RUN pip install --no-cache-dir -r ./scraper-python/requirements.txt

# Copy Go binary, Python scraper script, and migrations
COPY --from=builder /app/joblessyu-bot .
COPY scraper-python/ ./scraper-python/
COPY migrations/ ./migrations/

# Expose lightweight HTTP healthcheck port for Cloud Run / K8s readiness probes
EXPOSE 8080

CMD ["./joblessyu-bot"]