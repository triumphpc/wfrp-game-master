# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o wfrp-bot main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/wfrp-bot .

# Copy non-essential files
COPY --from=builder /app/rules ./rules
COPY --from=builder /app/prompts ./prompts

# Expose ports (no actual ports needed for bot)
# EXPOSE 8080

# Set environment
ENV PATH="/app:${PATH}"

# Run the bot
CMD ["./wfrp-bot"]
