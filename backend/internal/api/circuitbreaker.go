package api

import (
	"context"
	"log"
	"sync"
	"time"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	// StateClosed means the circuit is closed (allowing requests)
	StateClosed CircuitState = iota
	// StateOpen means the circuit is open (blocking requests)
	StateOpen
	// StateHalfOpen means the circuit is half-open (testing if service is recovered)
	StateHalfOpen
)

// CircuitBreakerConfig holds configuration for the circuit breaker
type CircuitBreakerConfig struct {
	// MaxFailures is the number of consecutive failures before opening the circuit
	MaxFailures int
	// Timeout is the duration to wait before attempting to close the circuit
	Timeout time.Duration
	// HalfOpenMaxRequests is the number of requests allowed in half-open state
	HalfOpenMaxRequests int
}

// CircuitBreaker implements the circuit breaker pattern for resilience
type CircuitBreaker struct {
	mu                sync.RWMutex
	state              CircuitState
	failures           int
	successCount        int
	lastFailureTime     time.Time
	config             CircuitBreakerConfig
	halfOpenSuccessChan chan struct{}
}

// NewCircuitBreaker creates a new circuit breaker with default configuration
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		state:  StateClosed,
		config: CircuitBreakerConfig{
			MaxFailures:        5,              // Open after 5 consecutive failures
			Timeout:            30 * time.Second, // Wait 30s before attempting recovery
			HalfOpenMaxRequests: 3,               // Allow 3 requests in half-open state
		},
		halfOpenSuccessChan: make(chan struct{}, 1),
	}
}

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	// Check if we can proceed
	if err := cb.canProceed(); err != nil {
		log.Printf("[CircuitBreaker] Circuit state: %v, blocking request", cb.getStateName())
		return err
	}

	// Execute the function
	err := fn()
	cb.recordResult(err)
	return err
}

// canProceed checks if the circuit breaker allows requests
func (cb *CircuitBreaker) canProceed() error {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return nil
	case StateOpen:
		// Check if timeout has elapsed
		if time.Since(cb.lastFailureTime) >= cb.config.Timeout {
			return nil // Allow request to test if service is recovered
		}
		return &CircuitBreakerError{
			State: cb.state,
			Message: "Circuit breaker is open, request blocked",
		}
	case StateHalfOpen:
		if cb.successCount >= cb.config.HalfOpenMaxRequests {
			return &CircuitBreakerError{
				State: cb.state,
				Message: "Circuit breaker is half-open, maximum test requests reached",
			}
		}
		return nil
	}
	return nil
}

// recordResult records the result of an operation and updates circuit state
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.handleFailure()
	} else {
		cb.handleSuccess()
	}
}

// handleFailure handles a failed operation
func (cb *CircuitBreaker) handleFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()
	cb.successCount = 0

	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.config.MaxFailures {
			cb.state = StateOpen
			log.Printf("[CircuitBreaker] Circuit opened after %d consecutive failures", cb.failures)
		}
	case StateHalfOpen:
		cb.state = StateOpen
		log.Printf("[CircuitBreaker] Circuit re-opened during half-open test")
	case StateOpen:
		// Already open, just update failure time
		log.Printf("[CircuitBreaker] Failure while circuit is open")
	}
}

// handleSuccess handles a successful operation
func (cb *CircuitBreaker) handleSuccess() {
	cb.failures = 0

	switch cb.state {
	case StateClosed:
		// Reset failure count
		cb.failures = 0
	case StateOpen:
		// Transition to half-open
		cb.state = StateHalfOpen
		cb.successCount = 1
		log.Printf("[CircuitBreaker] Circuit transitioned to half-open state")
	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.config.HalfOpenMaxRequests {
			cb.state = StateClosed
			cb.successCount = 0
			log.Printf("[CircuitBreaker] Circuit closed after successful recovery test")
		}
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// getStateName returns a string representation of the circuit state
func (cb *CircuitBreaker) getStateName() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	switch cb.state {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// GetStats returns statistics about the circuit breaker
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	timeUntilRetry := time.Duration(0)
	if cb.state == StateOpen {
		timeUntilRetry = cb.config.Timeout - time.Since(cb.lastFailureTime)
		if timeUntilRetry < 0 {
			timeUntilRetry = 0
		}
	}

	return CircuitBreakerStats{
		State:             cb.getStateName(),
		Failures:          cb.failures,
		SuccessCount:       cb.successCount,
		TimeUntilNextRetry:  timeUntilRetry,
	}
}

// CircuitBreakerError is returned when the circuit breaker blocks a request
type CircuitBreakerError struct {
	State   CircuitState
	Message string
}

func (e *CircuitBreakerError) Error() string {
	return e.Message
}

// CircuitBreakerStats provides statistics about the circuit breaker
type CircuitBreakerStats struct {
	State             string        `json:"state"`
	Failures          int           `json:"failures"`
	SuccessCount       int           `json:"successCount"`
	TimeUntilNextRetry  time.Duration `json:"timeUntilNextRetry"`
}

// Global circuit breaker instance for MPD operations
var MPDCircuitBreaker = NewCircuitBreaker()