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

	tests := []struct {
		method               string
		expectedCacheControl string
	}{
		{"GET", "no-store"},
		{"POST", ""},
	}

	for _, tt := range tests {
		req, err := http.NewRequest(tt.method, "/test", nil)
		if err != nil {
			t.Fatalf("Could not create %v request: %v", tt.method, err)
		}

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		actualCacheControl := recorder.Header().Get("Cache-Control")
		if actualCacheControl != tt.expectedCacheControl {
			t.Errorf("expected Cache-Control header to be %q for %v request, got %q",
				tt.expectedCacheControl, tt.method, actualCacheControl)
		}
	}
}
