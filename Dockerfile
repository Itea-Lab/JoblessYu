# Stage 1: Build Go binary
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o joblessyu-bot ./cmd/bot

# Stage 2: Final Runtime Image (Go + Python + JobSpy)
FROM alpine:latest

# Install certificates, timezone, python3, and py3-pip
RUN apk add --no-cache ca-certificates tzdata python3 py3-pip

WORKDIR /app

# Copy python dependencies & set up venv
COPY scraper-python/ /app/scraper-python/
RUN python3 -m venv /app/scraper-python/.venv && \
    /app/scraper-python/.venv/bin/pip install --no-cache-dir -r /app/scraper-python/requirements.txt

# Copy static Go binary from builder
COPY --from=builder /app/joblessyu-bot .

# Run the bot
CMD ["./joblessyu-bot"]