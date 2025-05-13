package circuitbreaker

import (
	"testing"
	"time"
)

func TestCircuitBreakerStateTransitions(t *testing.T) {
	cb := NewCircuitBreaker(3, 5*time.Second, 0.5)

	// Test initial state
	if cb.GetState() != StateClosed {
		t.Errorf("Expected initial state to be CLOSED, got %v", cb.GetState())
	}

	// Test transition to OPEN after failures
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}
	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be OPEN after 3 failures, got %v", cb.GetState())
	}

	// Test half-open state after timeout
	time.Sleep(5 * time.Second)
	if cb.GetState() != StateHalfOpen {
		t.Errorf("Expected state to be HALF-OPEN after timeout, got %v", cb.GetState())
	}

	// Test successful recovery
	cb.RecordSuccess()
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be CLOSED after success, got %v", cb.GetState())
	}
}

func TestCircuitBreakerFailureThreshold(t *testing.T) {
	cb := NewCircuitBreaker(2, 5*time.Second, 0.5)

	// Test failure threshold
	cb.RecordFailure()
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be CLOSED after 1 failure, got %v", cb.GetState())
	}

	cb.RecordFailure()
	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be OPEN after 2 failures, got %v", cb.GetState())
	}
}

func TestCircuitBreakerSuccessRate(t *testing.T) {
	cb := NewCircuitBreaker(3, 5*time.Second, 0.6)

	// Test success rate threshold
	cb.RecordSuccess()
	cb.RecordSuccess()
	cb.RecordFailure()
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be CLOSED with 66%% success rate, got %v", cb.GetState())
	}

	cb.RecordFailure()
	cb.RecordFailure()
	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be OPEN with 40%% success rate, got %v", cb.GetState())
	}
}
