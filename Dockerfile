# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o pr-reviewer ./cmd/app

# Runtime stage
FROM alpine:3.18

RUN apk update && \
    apk add --no-cache ca-certificates 

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/pr-reviewer .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./pr-reviewer"]
