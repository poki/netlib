package internal

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/poki/netlib/internal/cloudflare"
	"github.com/poki/netlib/internal/signaling"
	"github.com/poki/netlib/internal/signaling/stores"
	"github.com/poki/netlib/internal/util"
)

func Signaling(ctx context.Context, store stores.Store, credentialsClient *cloudflare.CredentialsClient) (http.Handler, func()) {
	mux := http.NewServeMux()

	openConnections, signaling := signaling.Handler(ctx, store, credentialsClient)

	cleanup := func() {
		openConnections.Wait()
	}
	mux.Handle("/v0/signaling", signaling)

	hasCredentials := uint32(0)
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadUint32(&hasCredentials) != 0 {
			w.WriteHeader(http.StatusOK)
			return
		}
		creds, _ := credentialsClient.GetCredentials(r.Context())
		if creds != nil {
			atomic.StoreUint32(&hasCredentials, 1)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})

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

	return mux, cleanup
}
