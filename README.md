# Scalable API Gateway with Circuit Breaker Pattern

A robust and scalable API Gateway implementation that demonstrates the Circuit Breaker pattern in a microservices architecture. This project showcases how to build resilient microservices that can gracefully handle failures and prevent cascading issues in distributed systems.

## 🌟 Features

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

## 🚀 Getting Started

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

## 📊 Circuit Breaker States

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

## 🔧 Configuration

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

## 📈 Monitoring

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

## 🧪 Testing

Run the circuit breaker test sequence:
1. Click "Run Circuit Breaker Test" on the dashboard
2. Watch the state transitions:
   - CLOSED → OPEN (after 3 failures)
   - OPEN → HALF-OPEN (after 10 seconds)
   - HALF-OPEN → CLOSED (after successful request)

## 🔍 Architecture

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

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Go Circuit Breaker](https://github.com/sony/gobreaker)
- [Chart.js](https://www.chartjs.org/)
- [Gorilla WebSocket](https://github.com/gorilla/websocket)
- [Prometheus Client](https://github.com/prometheus/client_golang) 