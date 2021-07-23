package internal

import (
	"net/http"

	"github.com/poki/netlib/internal/signaling"
	"github.com/poki/netlib/internal/signaling/stores"
	"github.com/poki/netlib/internal/util"
)

func Signaling() http.Handler {
	mux := http.NewServeMux()

	store := stores.NewMemoryStore()
	mux.Handle("/v0/signaling", signaling.Handler(store))

	healthCheck := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/health" {
			util.ErrorAndAbort(w, r, http.StatusNotFound, "not-found")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{\"healthy\":true}\n")) //nolint:errcheck
	}
	mux.HandleFunc("/health", healthCheck)
	mux.HandleFunc("/", healthCheck)

	return mux
}
