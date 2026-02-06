package stores

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/koenbollen/logging"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/poki/netlib/internal/signaling/latencydata"
	"github.com/poki/netlib/migrations"
	"go.uber.org/zap"
)

func getConfig(url string) (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	return cfg, nil
}

func FromEnv(ctx context.Context) (Store, chan struct{}, error) {
	logger := logging.GetLogger(ctx)

	if url, ok := os.LookupEnv("DATABASE_URL"); ok {
		cfg, err := getConfig(url)
		if err != nil {
			return nil, nil, err
		}
		db, err := pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to connect: %w", err)
		}
		if err := migrations.Up(db.Config().ConnConfig); err != nil {
			return nil, nil, fmt.Errorf("failed to migrate: %w", err)
		}
		if err := latencydata.EnsureLatencyData(ctx, db); err != nil {
			return nil, nil, fmt.Errorf("failed to load latency data: %w", err)
		}
		store, err := NewPostgresStore(ctx, db)
		if err != nil {
			return nil, nil, err
		}
		return store, nil, nil

	} else if os.Getenv("ENV") == "local" || os.Getenv("ENV") == "test" {
		pool, err := dockertest.NewPool("")
		if err != nil {
			return nil, nil, err
		}
		resource, err := pool.RunWithOptions(&dockertest.RunOptions{
			Repository: "pgvector/pgvector",
			Tag:        "pg15",
			Env: []string{
				"POSTGRES_PASSWORD=test",
				"POSTGRES_USER=test",
				"POSTGRES_DB=test",
				"listen_addresses='*'",
				"fsync='off'",
				"full_page_writes='off'",
			},
		}, func(config *docker.HostConfig) {
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		})
		if err != nil {
			return nil, nil, err
		}
		flushed := make(chan struct{})
		go func() {
			<-ctx.Done()
			pool.Purge(resource) // nolint:errcheck
			close(flushed)
		}()
		if os.Getenv("ENV") == "test" {
			// Automatically expire the container after 120 seconds in tests.
			if err := resource.Expire(120); err != nil {
				return nil, nil, err
			}
		}

		hostPort := resource.GetHostPort("5432/tcp")

		// If we are running in Docker, we need to replace localhost with host.docker.internal.
		// This allows us to connect to the postgres dockertest just started inside our container.
		// localhost will not work in this case as it refers to the container itself.
		if runningInDocker() {
			hostPort = strings.ReplaceAll(hostPort, "localhost", "host.docker.internal")
		}

		databaseUrl := fmt.Sprintf("postgres://test:test@%s/test?sslmode=disable", hostPort)

		// This log message is used by the test suite to pass the database URL to the testproxy.
		logger.Info("using database", zap.String("url", databaseUrl))

		cfg, err := getConfig(databaseUrl)
		if err != nil {
			return nil, nil, err
		}

		var db *pgxpool.Pool
		pool.MaxWait = 120 * time.Second
		if err = pool.Retry(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
			defer cancel()
			db, err = pgxpool.NewWithConfig(ctx, cfg)
			if err != nil {
				return err
			}
			return db.Ping(ctx)
		}); err != nil {
			return nil, nil, err
		}

		if err := migrations.Up(db.Config().ConnConfig); err != nil {
			return nil, nil, fmt.Errorf("failed to migrate: %w", err)
		}
		if err := latencydata.EnsureLatencyData(ctx, db); err != nil {
			return nil, nil, fmt.Errorf("failed to load latency data: %w", err)
		}

		store, err := NewPostgresStore(ctx, db)
		if err != nil {
			return nil, nil, err
		}
		return store, flushed, nil
	}
	return nil, nil, fmt.Errorf("no database configured, set DATABASE_URL in production, or ENV=local to automatically start a temporary database")
}

// runningInDocker returns true if the code is running inside a Docker container.
func runningInDocker() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	if b, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		return bytes.Contains(b, []byte("docker")) ||
			bytes.Contains(b, []byte("kubepods")) ||
			bytes.Contains(b, []byte("containerd"))
	}

	_, err := net.LookupHost("host.docker.internal")
	return err == nil
}
