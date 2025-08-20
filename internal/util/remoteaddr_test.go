package util

import (
	"context"
	"testing"
)

func TestGetRemoteAddr(t *testing.T) {
	// Test context key matching with metrics package
	type metricsContextKey int
	remoteAddrKey := metricsContextKey(1)
	
	testAddr := "192.168.1.1:12345"
	ctx := context.WithValue(context.Background(), remoteAddrKey, testAddr)
	
	result := GetRemoteAddr(ctx)
	if result != testAddr {
		t.Errorf("GetRemoteAddr() = %v, want %v", result, testAddr)
	}
}

func TestGetRemoteAddr_Empty(t *testing.T) {
	// Test with empty context
	ctx := context.Background()
	
	result := GetRemoteAddr(ctx)
	if result != "" {
		t.Errorf("GetRemoteAddr() = %v, want empty string", result)
	}
}