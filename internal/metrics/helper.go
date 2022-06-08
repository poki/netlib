package metrics

import (
	"context"
	"net/http"
	"strings"
)

type metricsContextKey int

var clientKey = metricsContextKey(0)
var remoteAddrKey = metricsContextKey(1)
var userAgentKey = metricsContextKey(2)

func Middleware(next http.Handler, client *Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, clientKey, client)

		remoteAddr := r.RemoteAddr
		if r.Header.Get("X-Forwarded-For") != "" {
			remoteAddr = strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
		}
		ctx = context.WithValue(ctx, remoteAddrKey, remoteAddr)
		ctx = context.WithValue(ctx, userAgentKey, r.UserAgent())

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Record(ctx context.Context, category, action, game, peerID, lobbyID string, data ...string) {
	if client, ok := ctx.Value(clientKey).(*Client); ok {
		client.Record(ctx, category, action, game, peerID, lobbyID, data...)
	}
}

func RecordEvent(ctx context.Context, params EventParams) {
	if client, ok := ctx.Value(clientKey).(*Client); ok {
		client.RecordEvent(ctx, params)
	}
}
