package util

import (
	"context"
	"testing"
	"time"
)

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