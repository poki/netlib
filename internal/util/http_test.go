package util

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNoStoreMiddleware(t *testing.T) {
	sampleHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := NoStoreMiddleware(sampleHandler)

	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	expectedCacheControl := "no-store"
	if rr.Header().Get("Cache-Control") != expectedCacheControl {
		t.Errorf("expected Cache-Control header to be %v, got %v",
			expectedCacheControl, rr.Header().Get("Cache-Control"))
	}
}
