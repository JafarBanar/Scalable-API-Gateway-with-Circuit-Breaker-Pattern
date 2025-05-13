package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"net"
	"io"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	"github.com/sony/gobreaker"
	"github.com/gorilla/websocket"
)

type Config struct {
	Port            string
	CacheServiceURL string
	APIKey          string
	RateLimit       string
}

type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

var (
	logger = logrus.New()

	// Prometheus metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}
	clients = make(map[*websocket.Conn]bool)
)

var limiterInstance *limiter.Limiter
var cacheBreaker *gobreaker.CircuitBreaker

func init() {
	// Initialize rate limiter
	rate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  100,
	}
	store := memory.NewStore()
	limiterInstance = limiter.New(store, rate)

	cacheBreaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "CacheService",
		MaxRequests: 1, // Allow only one request in half-open state
		Interval:    0,
		Timeout:     10 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			logger.Printf("Circuit breaker '%s' state changed from %v to %v", name, from, to)
			broadcastStateChange(from, to)
		},
	})
}

func main() {
	// Configure logger
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	config := Config{
		Port:            getEnv("PORT", "8080"),
		CacheServiceURL: getEnv("CACHE_SERVICE_URL", "http://localhost:8081"),
		APIKey:          getEnv("API_KEY", "your-secret-key"),
		RateLimit:       getEnv("RATE_LIMIT", "100-M"),
	}

	// Initialize router
	router := setupRouter(config, limiterInstance)

	// Create server with timeouts
	server := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Infof("Starting API Gateway on port %s", config.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for health check, metrics, and cache endpoints
		if r.URL.Path == "/health" || r.URL.Path == "/metrics" || strings.HasPrefix(r.URL.Path, "/api/cache") {
			next.ServeHTTP(w, r)
			return
		}

		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr // fallback
		}
		limiterCtx, err := limiterInstance.Get(r.Context(), ip)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if limiterCtx.Reached {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start).Seconds()

		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, fmt.Sprintf("%d", http.StatusOK)).Inc()
	})
}

func authMiddleware(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for health check, metrics, monitoring, WebSocket, favicon, test endpoints, and root path
			if r.URL.Path == "/health" || r.URL.Path == "/metrics" || 
			   r.URL.Path == "/monitor" || r.URL.Path == "/ws" || 
			   r.URL.Path == "/favicon.ico" || r.URL.Path == "/test-circuit" ||
			   r.URL.Path == "/" {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondWithError(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				respondWithError(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			if parts[1] != apiKey {
				respondWithError(w, "Invalid API key", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func respondWithError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(Response{
		Status:  "error",
		Message: message,
	})
}

func respondWithJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(Response{
		Status: "success",
		Data:   data,
	})
}

func broadcastStateChange(from, to gobreaker.State) {
	logger.Printf("Broadcasting state change from %v to %v", from, to)
	message := map[string]interface{}{
		"type": "state_change",
		"from": from.String(),
		"to":   to.String(),
		"time": time.Now().Format(time.RFC3339),
	}
	
	logger.Printf("Number of connected clients: %d", len(clients))
	for client := range clients {
		logger.Printf("Sending state change to client")
		err := client.WriteJSON(message)
		if err != nil {
			logger.Printf("Failed to send state change to client: %v", err)
			client.Close()
			delete(clients, client)
		}
	}
}

func setupRouter(config Config, limiter *limiter.Limiter) http.Handler {
	r := mux.NewRouter()

	// Root path redirects to monitor dashboard
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/monitor", http.StatusMovedPermanently)
	}).Methods("GET")

	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		respondWithJSON(w, map[string]string{"status": "healthy"}, http.StatusOK)
	}).Methods("GET")

	// Test endpoint for circuit breaker demonstration
	r.HandleFunc("/test-circuit", func(w http.ResponseWriter, r *http.Request) {
		// Simulate a sequence of state changes
		go func() {
			// Initial state is CLOSED
			logger.Printf("Starting circuit breaker test sequence")
			time.Sleep(5 * time.Second) // 5 seconds before step 2

			// Simulate failures to trigger OPEN state
			for i := 0; i < 3; i++ {
				_, err := cacheBreaker.Execute(func() (interface{}, error) {
					logger.Printf("Simulated failure %d", i+1)
					return nil, fmt.Errorf("simulated failure")
				})
				if err != nil {
					logger.Printf("Simulated failure %d", i+1)
				}
				time.Sleep(1 * time.Second)
			}

			// Wait for HALF-OPEN state
			logger.Printf("Waiting for circuit breaker timeout...")
			time.Sleep(10 * time.Second) // 10 seconds before step 3

			// Additional delay after HALF-OPEN state
			logger.Printf("Circuit breaker is now in HALF-OPEN state, waiting before attempting success...")
			time.Sleep(2 * time.Second) // 2 seconds before step 4

			// Simulate successful request to return to CLOSED
			logger.Printf("Attempting successful request...")
			success := false
			for i := 0; i < 5; i++ { // Try up to 5 times
				result, err := cacheBreaker.Execute(func() (interface{}, error) {
					logger.Printf("Executing success simulation attempt %d", i+1)
					return "success", nil
				})
				if err == nil {
					logger.Printf("Successfully simulated recovery: %v", result)
					success = true
					break
				}
				logger.Printf("Failed to simulate success (attempt %d): %v", i+1, err)
				time.Sleep(1 * time.Second)
			}

			if !success {
				logger.Printf("Failed to simulate success after all attempts")
			}

			// Verify final state
			time.Sleep(2 * time.Second)
			finalState := cacheBreaker.State()
			logger.Printf("Final circuit breaker state: %v", finalState)

			// Force a state update to ensure the UI reflects the final state
			if finalState == gobreaker.StateClosed {
				broadcastStateChange(gobreaker.StateHalfOpen, gobreaker.StateClosed)
			}
		}()

		respondWithJSON(w, map[string]string{
			"status": "success",
			"message": "Circuit breaker test sequence started. Watch the monitor dashboard for state changes.",
		}, http.StatusOK)
	}).Methods("GET")

	// Favicon endpoint (returns 404)
	r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	// Metrics endpoint
	r.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// API routes
	api := r.PathPrefix("/api").Subrouter()
	
	// Example protected endpoint
	api.HandleFunc("/example", func(w http.ResponseWriter, r *http.Request) {
		respondWithJSON(w, map[string]string{"message": "This is a protected endpoint"}, http.StatusOK)
	}).Methods("GET")

	// Cache endpoints
	api.HandleFunc("/cache/{key}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		key := vars["key"]

		// Forward request to cache service
		cacheURL := config.CacheServiceURL + "/cache/" + key
		req, err := http.NewRequest("GET", cacheURL, nil)
		if err != nil {
			respondWithError(w, "Failed to create request", http.StatusInternalServerError)
			return
		}
		result, err := cacheBreaker.Execute(func() (interface{}, error) {
			return http.DefaultClient.Do(req)
		})
		if err != nil {
			respondWithError(w, "Failed to get cache", http.StatusInternalServerError)
			return
		}
		resp := result.(*http.Response)
		defer resp.Body.Close()

		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}).Methods("GET")

	api.HandleFunc("/cache", func(w http.ResponseWriter, r *http.Request) {
		// Forward request to cache service
		cacheURL := config.CacheServiceURL + "/cache"
		result, err := cacheBreaker.Execute(func() (interface{}, error) {
			return http.Post(cacheURL, "application/json", r.Body)
		})
		if err != nil {
			respondWithError(w, "Failed to set cache", http.StatusInternalServerError)
			return
		}
		resp := result.(*http.Response)
		defer resp.Body.Close()

		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}).Methods("POST")

	// Add WebSocket endpoint for monitoring
	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("New WebSocket connection request")
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Printf("Failed to upgrade connection: %v", err)
			return
		}
		defer conn.Close()

		logger.Printf("WebSocket connection established")
		clients[conn] = true
		defer delete(clients, conn)

		// Send initial state
		currentState := cacheBreaker.State().String()
		logger.Printf("Sending initial state: %s", currentState)
		err = conn.WriteJSON(map[string]interface{}{
			"type": "initial_state",
			"state": currentState,
		})
		if err != nil {
			logger.Printf("Failed to send initial state: %v", err)
			return
		}

		// Keep connection alive
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				logger.Printf("WebSocket connection closed: %v", err)
				break
			}
		}
	})

	// Add monitoring dashboard
	r.HandleFunc("/monitor", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "monitor.html")
	})

	// Apply middleware in order
	handler := metricsMiddleware(r)
	handler = rateLimitMiddleware(handler)
	handler = authMiddleware(config.APIKey)(handler)

	return handler
} 