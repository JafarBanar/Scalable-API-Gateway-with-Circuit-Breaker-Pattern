package circuitbreaker

import (
	"fmt"
	"sync"
	"time"
)

// CircuitBreakerState represents the current state of the circuit breaker
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu               sync.RWMutex
	state            CircuitBreakerState
	failureCount     int
	failureThreshold int
	resetTimeout     time.Duration
	lastFailureTime  time.Time
	successCount     int
	totalCount       int
	successRate      float64
}

// NewCircuitBreaker creates a new circuit breaker instance
func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration, successRate float64) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		successRate:      successRate,
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// RecordFailure records a failure and updates the circuit breaker state
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.totalCount++
	cb.lastFailureTime = time.Now()

	if cb.state == StateClosed {
		if cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
		}
	} else if cb.state == StateHalfOpen {
		cb.state = StateOpen
	}
}

// RecordSuccess records a success and updates the circuit breaker state
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	fmt.Printf("[DEBUG] RecordSuccess called. State before: %v\n", cb.state)
	cb.successCount++
	cb.totalCount++

	if cb.state == StateHalfOpen {
		fmt.Println("[DEBUG] Transitioning from HALF-OPEN to CLOSED")
		cb.state = StateClosed
		cb.failureCount = 0
		cb.successCount = 0 // Reset success count when transitioning to CLOSED
		cb.totalCount = 0   // Reset total count when transitioning to CLOSED
		return              // Return immediately after transitioning to CLOSED
	}

	// Only check success rate if we're in CLOSED state
	if cb.state == StateClosed && cb.totalCount > 0 {
		currentSuccessRate := float64(cb.successCount) / float64(cb.totalCount)
		fmt.Printf("[DEBUG] Current success rate: %.2f\n", currentSuccessRate)
		if currentSuccessRate < cb.successRate {
			fmt.Println("[DEBUG] Success rate below threshold, transitioning to OPEN")
			cb.state = StateOpen
		}
	}
	fmt.Printf("[DEBUG] RecordSuccess finished. State after: %v\n", cb.state)
}

// AllowRequest determines if a request should be allowed based on the current state
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	fmt.Printf("[DEBUG] AllowRequest called. State: %v\n", cb.state)
	switch cb.state {
	case StateClosed:
		fmt.Println("[DEBUG] State is CLOSED, allowing request.")
		return true
	case StateOpen:
		if time.Since(cb.lastFailureTime) >= cb.resetTimeout {
			fmt.Println("[DEBUG] Timeout passed, transitioning to HALF-OPEN.")
			cb.state = StateHalfOpen
			return true
		}
		fmt.Println("[DEBUG] State is OPEN, request not allowed.")
		return false
	case StateHalfOpen:
		fmt.Println("[DEBUG] State is HALF-OPEN, allowing request.")
		return true
	default:
		fmt.Println("[DEBUG] Unknown state, request not allowed.")
		return false
	}
}
