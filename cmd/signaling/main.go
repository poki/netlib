package main

import (
	"context"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/koenbollen/logging"
	"github.com/poki/netlib/internal"
	"github.com/poki/netlib/internal/cloudflare"
	"github.com/poki/netlib/internal/metrics"
	"github.com/poki/netlib/internal/signaling/stores"
	"github.com/poki/netlib/internal/util"
	"github.com/rs/cors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger := logging.New(ctx, "netlib", "signaling")
	defer logger.Sync() // nolint:errcheck
	logger.Info("init")
	defer logger.Info("fin")
	ctx = logging.WithLogger(ctx, logger)

	store, flushed, err := stores.FromEnv(ctx)
	if err != nil {
		// Don't print a stacktrace here, it's confusing for users.
		logger.WithOptions(zap.AddStacktrace(zapcore.InvalidLevel)).Error("failed to setup store", zap.Error(err))
		return
	}

	if os.Getenv("ENV") == "local" || os.Getenv("ENV") == "test" {
		rand.Seed(0)
	} else {
		rand.Seed(time.Now().UnixNano())
	}

	credentialsClient := cloudflare.NewCredentialsClient(
		os.Getenv("CLOUDFLARE_APP_ID"),
		os.Getenv("CLOUDFLARE_AUTH_KEY"),
		2*time.Hour,
	)

	go credentialsClient.Run(ctx)

	mux, cleanup := internal.Signaling(ctx, store, credentialsClient)

	corsHandler := cors.Default()
	handler := corsHandler.Handler(mux)
	handler = util.NoStoreMiddleware(handler)
	handler = logging.Middleware(handler, logger)

	if metricsURL, ok := os.LookupEnv("METRICS_URL"); ok {
		client := metrics.NewClient(metricsURL)
		handler = metrics.Middleware(handler, client)
	}

	addr := util.Getenv("ADDR", ":8080")
	server := &http.Server{
		Addr:    addr,
		Handler: handler,

		BaseContext: func(net.Listener) context.Context {
			return ctx
		},

		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  650 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("failed to listen and serve", zap.Error(err))
		}
	}()
	logger.Info("listening", zap.String("addr", addr))

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("failed to shutdown server", zap.Error(err))
	}

	cleanup()
	if flushed != nil {
		<-flushed
	}
}
