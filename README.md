# Scalable API Gateway with Circuit Breaker Pattern

A robust and scalable API Gateway implementation that demonstrates the Circuit Breaker pattern in a microservices architecture. This project showcases how to build resilient microservices that can gracefully handle failures and prevent cascading issues in distributed systems.

## Features

- **Circuit Breaker Pattern Implementation**
  - CLOSED state: Normal operation, requests flow through
  - OPEN state: Fails fast when service is down
  - HALF-OPEN state: Tests service recovery
  - Automatic state transitions based on failure thresholds

- **Real-time Monitoring Dashboard**
  - Live state visualization with Chart.js
  - WebSocket-based real-time updates
  - State change timeline
  - Color-coded status indicators

- **Security & Performance**
  - Rate limiting to prevent abuse
  - API key authentication
  - Prometheus metrics integration
  - Request/response logging

- **Technology Stack**
  - Go (API Gateway)
  - Rust (Cache Service)
  - Redis (Caching)
  - WebSocket (Real-time Updates)
  - Chart.js (Visualization)
  - Docker (Containerization)

## Getting Started

### Prerequisites

- Docker and Docker Compose
- Go 1.21 or later
- Rust 1.82 or later

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/scalable-api-gateway.git
   cd scalable-api-gateway
   ```

2. Build and run with Docker Compose:
   ```bash
   docker-compose up --build
   ```

3. Access the monitoring dashboard:
   ```
   http://localhost:8080/monitor
   ```

## Circuit Breaker States

### CLOSED (Green)
- Normal operation
- Requests are allowed through to the service
- Failures are counted
- After 3 consecutive failures, transitions to OPEN state

### OPEN (Red)
- Service is failing
- Requests are immediately rejected
- Prevents cascading failures
- After 10 seconds, transitions to HALF-OPEN state

### HALF-OPEN (Yellow)
- Testing service recovery
- Limited requests allowed through
- Success returns to CLOSED state
- Failure returns to OPEN state

## Configuration

### Environment Variables

```env
PORT=8080                    # API Gateway port
CACHE_SERVICE_URL=http://localhost:8081  # Cache service URL
API_KEY=your-secret-key      # API key for authentication
RATE_LIMIT=100-M            # Rate limit (100 requests per minute)
```

### Circuit Breaker Settings

```go
MaxRequests: 1              // Allow only one request in half-open state
Timeout: 10 * time.Second   // Time before transitioning to half-open
ConsecutiveFailures: 3      // Number of failures before opening circuit
```

## Monitoring

### Dashboard Features
- Real-time state visualization
- State change timeline
- Color-coded status indicators
- Interactive chart with state history

### Metrics
- HTTP request counts
- Request duration
- Circuit breaker state changes
- Error rates

## Testing

Run the circuit breaker test sequence:
1. Click "Run Circuit Breaker Test" on the dashboard
2. Watch the state transitions:
   - CLOSED → OPEN (after 3 failures)
   - OPEN → HALF-OPEN (after 10 seconds)
   - HALF-OPEN → CLOSED (after successful request)

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│                 │     │                 │     │                 │
│  API Gateway    │────▶│  Cache Service  │────▶│     Redis       │
│    (Go)         │     │    (Rust)       │     │                 │
│                 │     │                 │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │
        │
        ▼
┌─────────────────┐
│                 │
│  Monitoring     │
│  Dashboard      │
│                 │
└─────────────────┘
```

## Technology Stack Deep Dive

### 1. Go (Golang) - API Gateway
**Why We Used Go:**
- High Concurrency: Go's lightweight Goroutines allow efficient handling of thousands of concurrent requests
- Simplicity: Go's clean, minimal syntax makes it easy to build and maintain scalable APIs
- Fast Compilation: Go is a compiled language, which means the API Gateway starts up instantly and runs efficiently
- Standard Library: Go's net/http package provides a powerful and flexible HTTP server out of the box
- Scalability: Goroutines and Channels make it easy to build scalable, concurrent applications

**How We Used Go:**
- Built the API Gateway that receives client requests
- Implemented the Circuit Breaker Pattern to protect the backend (Cache Service)
- Added API Key Authentication for secure access
- Enabled Rate Limiting to prevent abuse (100 requests per minute)
- Used http.Client for efficient HTTP request forwarding to the Cache Service (Rust)
- Configured HTTPS/TLS for secure communication

### 2. Rust - Cache Service
**Why We Used Rust:**
- High Performance: Rust provides low-level memory control without a Garbage Collector (GC), making it ideal for a high-performance cache
- Memory Safety: Rust's Ownership and Borrowing model ensures memory safety without runtime overhead
- Concurrency without Errors: Rust's async/await model allows for non-blocking, safe concurrency
- Error Handling: Rust's Result<T, E> model ensures that all errors are handled explicitly
- Security: Rust's strict compile-time checks prevent memory-related vulnerabilities

**How We Used Rust:**
- Developed the Cache Service, which provides high-speed caching for frequently accessed data
- Used an in-memory HashMap for fast data access
- Integrated with Redis for persistent caching
- Implemented asynchronous request handling with Tokio for non-blocking performance
- Exposed a RESTful API (using Actix-Web) for the Go API Gateway to interact with
- Added error handling using Rust's Result<T, E> for safe, predictable operations

### 3. Redis - Persistent Cache Storage
**Why We Used Redis:**
- High-Speed Caching: Redis is an in-memory data store that provides extremely fast read/write speeds
- Persistence: Redis can be configured to persist data to disk, ensuring that cached data is not lost on service restart
- TTL Support: Time-to-Live (TTL) allows us to automatically expire outdated cache entries
- Scalability: Redis can be used in a distributed setup for large-scale caching

**How We Used Redis:**
- Integrated with the Rust Cache Service for persistent caching
- Stored frequently requested data to reduce the load on the backend
- Provided fast data retrieval with an in-memory store

### 4. Circuit Breaker Pattern (Go)
**Why We Used Circuit Breaker Pattern:**
- Resilience: Prevents cascading failures across the system when the Cache Service (Rust) is down
- Automatic Recovery: Automatically tests service recovery with HALF-OPEN state
- Fast Failure: Avoids wasting resources on repeated failed requests

**How We Used Circuit Breaker:**
- Implemented the Circuit Breaker Pattern in the Go API Gateway
- Configured three states:
  - CLOSED: Normal operation
  - OPEN: Fast failure when the service is down
  - HALF-OPEN: Limited test requests for recovery testing
- Monitored the number of failed requests to transition between states

### 5. WebSocket + Chart.js - Real-Time Monitoring Dashboard
**Why We Used WebSocket + Chart.js:**
- Real-Time Updates: WebSocket provides instant updates without reloading the page
- Visual Insights: Chart.js offers a clear, interactive view of the Circuit Breaker states
- User-Friendly: Color-coded states (Green for CLOSED, Red for OPEN, Yellow for HALF-OPEN) make it intuitive

**How We Used WebSocket + Chart.js:**
- WebSocket:
  - Sent real-time state updates (CLOSED, OPEN, HALF-OPEN) to the dashboard
  - Used a WebSocket server in Go to broadcast state changes
- Chart.js:
  - Visualized state changes over time
  - Displayed state transition history with a clear timeline
  - Provided color-coded state indicators

### 6. Docker & Docker Compose - Containerization
**Why We Used Docker:**
- Portability: Runs consistently across all environments (development, testing, production)
- Isolation: Each service (Go API Gateway, Rust Cache Service, Redis) is isolated in its own container
- Scalability: Easily scales by launching multiple instances of the API Gateway
- Multi-Service Management: Docker Compose simplifies managing all services together

**How We Used Docker:**
- Created Docker images for Go API Gateway and Rust Cache Service
- Configured Docker Compose for multi-container setup:
  - API Gateway (Go)
  - Cache Service (Rust)
  - Redis (Persistent caching)
- Set up volume mapping for Redis persistence

### 7. Prometheus - Metrics Monitoring (Optional)
**Purpose:** Monitors API performance and Circuit Breaker health.

**Why We Used Prometheus:**
- Provides detailed metrics for API requests, response times, and errors
- Supports alerting for high error rates or Circuit Breaker failures

**How We Used Prometheus:**
- Configured Prometheus to collect metrics from the Go API Gateway
- Exposed a /metrics endpoint in the API Gateway for Prometheus scraping
- Visualized metrics in Grafana (optional)

### 8. API Key Authentication (Go)
**Purpose:** Secures the API Gateway by allowing only authorized clients.

**Why We Used API Key Authentication:**
- Simple but effective security mechanism
- Prevents unauthorized access to the API Gateway

**How We Used It:**
- Each request must include a valid API key in the header
- Requests without a valid API key are rejected with a 401 Unauthorized error

### 9. Rate Limiting (Go)
**Purpose:** Prevents API abuse by limiting the number of requests per client.

**Why We Used Rate Limiting:**
- Protects the API Gateway from excessive traffic
- Ensures fair usage for all clients

**How We Used It:**
- Configured a rate limit of 100 requests per minute (configurable)
- Requests exceeding this limit are rejected with a 429 Too Many Requests error

### 10. HTTPS/TLS (Secure Communication)
**Purpose:** Ensures secure communication between clients and the API Gateway.

**Why We Used HTTPS/TLS:**
- Encrypts client-server communication, protecting data in transit
- Prevents eavesdropping and man-in-the-middle attacks

**How We Used It:**
- Configured HTTPS in the Go API Gateway with TLS certificates
- Ensured all client requests use HTTPS for security

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Go Circuit Breaker](https://github.com/sony/gobreaker)
- [Chart.js](https://www.chartjs.org/)
- [Gorilla WebSocket](https://github.com/gorilla/websocket)
- [Prometheus Client](https://github.com/prometheus/client_golang) 