# Stage 1: Build
FROM golang:alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first
COPY go.mod go.sum ./
RUN go mod download

# Copy all source code
COPY . .

# CGO_ENABLED=0: Create a static binary (statically linked), independent of external C libraries
# GOOS=linux: Ensure the build file runs on the Linux operating system
# -ldflags="-w -s": Remove debug information to significantly reduce file size
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o joblessyu-bot ./cmd/bot

# Stage 2: Final Image
FROM alpine:latest

# Install SSL/TLS certificates and time zone
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from build stage
COPY --from=builder /app/joblessyu-bot .

# Run the bot
CMD ["./joblessyu-bot"]
