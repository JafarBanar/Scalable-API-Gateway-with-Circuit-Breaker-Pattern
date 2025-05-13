# Build stage
FROM rust:1.82-slim as builder

WORKDIR /app

# Copy Cargo.toml and Cargo.lock
COPY Cargo.toml ./

# Create a dummy main.rs to build dependencies
RUN mkdir src && \
    echo "fn main() {}" > src/main.rs && \
    cargo build --release

# Copy source code
COPY . .

# Build the application
RUN cargo build --release

# Runtime stage
FROM debian:bookworm-slim

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/target/release/cache-service .

# Expose port
EXPOSE 8081

# Run the application
CMD ["./cache-service"] 