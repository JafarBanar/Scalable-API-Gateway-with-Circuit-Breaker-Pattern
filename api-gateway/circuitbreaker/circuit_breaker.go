package circuitbreaker

import (
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

	cb.successCount++
	cb.totalCount++

	if cb.state == StateHalfOpen {
		cb.state = StateClosed
		cb.failureCount = 0
	}

	// Check if we need to transition to OPEN state based on success rate
	if cb.state == StateClosed && cb.totalCount > 0 {
		currentSuccessRate := float64(cb.successCount) / float64(cb.totalCount)
		if currentSuccessRate < cb.successRate {
			cb.state = StateOpen
		}
	}
}

// AllowRequest determines if a request should be allowed based on the current state
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailureTime) >= cb.resetTimeout {
			cb.state = StateHalfOpen
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}
