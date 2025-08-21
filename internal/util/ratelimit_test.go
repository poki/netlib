package util

import (
	"context"
	"testing"
	"time"
)

func TestPasswordRateLimiter_IsAllowed(t *testing.T) {
	tests := []struct {
		name           string
		maxAttempts    int
		windowSize     time.Duration
		attempts       int
		timeBetween    time.Duration
		expectedResult bool
	}{
		{
			name:           "first attempt should be allowed",
			maxAttempts:    3,
			windowSize:     time.Minute,
			attempts:       1,
			timeBetween:    0,
			expectedResult: true,
		},
		{
			name:           "attempts within limit should be allowed",
			maxAttempts:    3,
			windowSize:     time.Minute,
			attempts:       2,
			timeBetween:    time.Second,
			expectedResult: true,
		},
		{
			name:           "attempts at limit should be blocked",
			maxAttempts:    3,
			windowSize:     time.Minute,
			attempts:       3,
			timeBetween:    time.Second,
			expectedResult: false,
		},
		{
			name:           "attempts beyond limit should be blocked",
			maxAttempts:    3,
			windowSize:     time.Minute,
			attempts:       5,
			timeBetween:    time.Second,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewPasswordRateLimiter(tt.maxAttempts, tt.windowSize)
			defer rl.Close()

			ctx := context.Background()
			ip := "192.168.1.1"

			// Record the specified number of attempts
			for i := 0; i < tt.attempts; i++ {
				if i > 0 && tt.timeBetween > 0 {
					time.Sleep(tt.timeBetween)
				}
				rl.RecordFailedAttempt(ctx, ip)
			}

			// Check if next attempt is allowed
			result := rl.IsAllowed(ctx, ip)
			if result != tt.expectedResult {
				t.Errorf("IsAllowed() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestPasswordRateLimiter_WindowExpiry(t *testing.T) {
	maxAttempts := 2
	windowSize := 100 * time.Millisecond
	rl := NewPasswordRateLimiter(maxAttempts, windowSize)
	defer rl.Close()

	ctx := context.Background()
	ip := "192.168.1.1"

	// Record max attempts
	rl.RecordFailedAttempt(ctx, ip)
	rl.RecordFailedAttempt(ctx, ip)

	// Should be blocked now
	if rl.IsAllowed(ctx, ip) {
		t.Error("Expected to be rate limited")
	}

	// Wait for window to expire
	time.Sleep(windowSize + 10*time.Millisecond)

	// Should be allowed again
	if !rl.IsAllowed(ctx, ip) {
		t.Error("Expected to be allowed after window expiry")
	}
}

func TestPasswordRateLimiter_MultipleIPs(t *testing.T) {
	rl := NewPasswordRateLimiter(2, time.Minute)
	defer rl.Close()

	ctx := context.Background()
	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	// Record max attempts for IP1
	rl.RecordFailedAttempt(ctx, ip1)
	rl.RecordFailedAttempt(ctx, ip1)

	// IP1 should be blocked
	if rl.IsAllowed(ctx, ip1) {
		t.Error("Expected IP1 to be rate limited")
	}

	// IP2 should still be allowed
	if !rl.IsAllowed(ctx, ip2) {
		t.Error("Expected IP2 to be allowed")
	}
}

func TestPasswordRateLimiter_EmptyIP(t *testing.T) {
	rl := NewPasswordRateLimiter(1, time.Minute)
	defer rl.Close()

	ctx := context.Background()

	// Empty IP should always be allowed
	if !rl.IsAllowed(ctx, "") {
		t.Error("Expected empty IP to be allowed")
	}

	// Recording failed attempt with empty IP should not panic
	rl.RecordFailedAttempt(ctx, "")

	// Should still be allowed
	if !rl.IsAllowed(ctx, "") {
		t.Error("Expected empty IP to still be allowed")
	}
}

func TestPasswordRateLimiter_Configuration(t *testing.T) {
	// Test that different configurations work
	tests := []struct {
		name        string
		maxAttempts int
		windowSize  time.Duration
		attempts    int
		shouldBlock bool
	}{
		{
			name:        "strict limit - 2 attempts per minute",
			maxAttempts: 2,
			windowSize:  time.Minute,
			attempts:    2,
			shouldBlock: true,
		},
		{
			name:        "lenient limit - 10 attempts per minute",
			maxAttempts: 10,
			windowSize:  time.Minute,
			attempts:    5,
			shouldBlock: false,
		},
		{
			name:        "very strict - 1 attempt per minute",
			maxAttempts: 1,
			windowSize:  time.Minute,
			attempts:    1,
			shouldBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewPasswordRateLimiter(tt.maxAttempts, tt.windowSize)
			defer rl.Close()

			ctx := context.Background()
			ip := "192.168.1.100"

			// Record the specified number of attempts
			for i := 0; i < tt.attempts; i++ {
				rl.RecordFailedAttempt(ctx, ip)
			}

			// Check if next attempt should be blocked
			allowed := rl.IsAllowed(ctx, ip)
			blocked := !allowed

			if blocked != tt.shouldBlock {
				t.Errorf("Expected blocked=%v but got blocked=%v", tt.shouldBlock, blocked)
			}
		})
	}
}