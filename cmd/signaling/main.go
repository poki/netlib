package main

import (
	"context"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/koenbollen/logging"
	"github.com/poki/netlib/internal"
	"github.com/poki/netlib/internal/turn"
	"github.com/poki/netlib/internal/util"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	logger := logging.New(ctx, "netlib", "signaling")
	defer logger.Sync() // nolint:errcheck
	logger.Info("init")
	ctx = logging.WithLogger(ctx, logger)

	if os.Getenv("ENV") == "local" || os.Getenv("ENV") == "test" {
		rand.Seed(0)
	} else {
		rand.Seed(time.Now().UnixNano())
	}

	mux := internal.Signaling()

	cors := cors.Default()
	handler := logging.Middleware(cors.Handler(mux), logger)

	addr := util.Getenv("ADDR", ":8080")
	server := &http.Server{
		Addr:    addr,
		Handler: handler,

		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  650 * time.Second,
	}

	if os.Getenv("ENV") != "test" {
		go turn.Run(ctx, addr) //nolint:errcheck
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("failed to listen and server", zap.Error(err))
		}
	}()
	logger.Info("listening", zap.String("addr", addr))

	<-ctx.Done()

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("failed to shutdown server", zap.Error(err))
	}
	logger.Info("fin")
}
