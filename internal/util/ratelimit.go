package util

import (
	"context"
	"sync"
	"time"
)

// GetRemoteAddr extracts the remote IP address from context 
// This duplicates the logic from metrics package to avoid circular imports
func GetRemoteAddr(ctx context.Context) string {
	// Use the same context key pattern as metrics package
	type metricsContextKey int
	remoteAddrKey := metricsContextKey(1)
	
	if addr, ok := ctx.Value(remoteAddrKey).(string); ok {
		return addr
	}
	return ""
}

// PasswordRateLimiter provides simple IP-based rate limiting for password attempts
type PasswordRateLimiter struct {
	mu            sync.RWMutex
	attempts      map[string][]time.Time
	maxAttempts   int
	windowSize    time.Duration
	cleanupTicker *time.Ticker
	done          chan bool
}

// NewPasswordRateLimiter creates a new rate limiter for password attempts
// maxAttempts: maximum password attempts allowed per IP in the time window
// windowSize: time window for counting attempts (e.g., 15 minutes)
func NewPasswordRateLimiter(maxAttempts int, windowSize time.Duration) *PasswordRateLimiter {
	rl := &PasswordRateLimiter{
		attempts:    make(map[string][]time.Time),
		maxAttempts: maxAttempts,
		windowSize:  windowSize,
		done:        make(chan bool),
	}

	// Start cleanup routine to prevent memory leaks
	rl.cleanupTicker = time.NewTicker(windowSize)
	go rl.cleanup()

	return rl
}

// IsAllowed checks if a password attempt from the given IP is allowed
// Returns true if attempt is allowed, false if rate limited
func (rl *PasswordRateLimiter) IsAllowed(ctx context.Context, remoteAddr string) bool {
	if remoteAddr == "" {
		return true // Allow if we can't determine IP
	}

	rl.mu.RLock()
	attempts, exists := rl.attempts[remoteAddr]
	rl.mu.RUnlock()

	if !exists {
		return true
	}

	now := time.Now()
	cutoff := now.Add(-rl.windowSize)

	// Count valid attempts within window
	validAttempts := 0
	for _, attemptTime := range attempts {
		if attemptTime.After(cutoff) {
			validAttempts++
		}
	}

	return validAttempts < rl.maxAttempts
}

// RecordFailedAttempt records a failed password attempt for the given IP
func (rl *PasswordRateLimiter) RecordFailedAttempt(ctx context.Context, remoteAddr string) {
	if remoteAddr == "" {
		return // Nothing to record if we can't determine IP
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.windowSize)

	// Clean up old attempts for this IP and add new one
	attempts := rl.attempts[remoteAddr]
	validAttempts := make([]time.Time, 0, len(attempts)+1)

	// Keep only recent attempts
	for _, attemptTime := range attempts {
		if attemptTime.After(cutoff) {
			validAttempts = append(validAttempts, attemptTime)
		}
	}

	// Add current attempt
	validAttempts = append(validAttempts, now)
	rl.attempts[remoteAddr] = validAttempts
}

// cleanup periodically removes old entries to prevent memory leaks
func (rl *PasswordRateLimiter) cleanup() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.mu.Lock()
			now := time.Now()
			cutoff := now.Add(-rl.windowSize)

			for ip, attempts := range rl.attempts {
				validAttempts := make([]time.Time, 0, len(attempts))
				for _, attemptTime := range attempts {
					if attemptTime.After(cutoff) {
						validAttempts = append(validAttempts, attemptTime)
					}
				}

				if len(validAttempts) == 0 {
					delete(rl.attempts, ip)
				} else {
					rl.attempts[ip] = validAttempts
				}
			}
			rl.mu.Unlock()
		case <-rl.done:
			return
		}
	}
}

// Close stops the cleanup routine
func (rl *PasswordRateLimiter) Close() {
	if rl.cleanupTicker != nil {
		rl.cleanupTicker.Stop()
	}
	close(rl.done)
}