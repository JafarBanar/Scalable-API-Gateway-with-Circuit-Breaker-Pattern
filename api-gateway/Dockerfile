# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code and monitor.html
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o api-gateway

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary and monitor.html from builder
COPY --from=builder /app/api-gateway .
COPY --from=builder /app/monitor.html .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./api-gateway"] 