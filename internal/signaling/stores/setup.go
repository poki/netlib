package stores

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/koenbollen/logging"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/poki/netlib/migrations"
	"go.uber.org/zap"
)

func FromEnv(ctx context.Context) (Store, chan struct{}, error) {
	logger := logging.GetLogger(ctx)

	if url, ok := os.LookupEnv("DATABASE_URL"); ok {
		db, err := pgxpool.New(ctx, url)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to connect: %w", err)
		}
		if err := migrations.Up(db.Config().ConnConfig); err != nil {
			return nil, nil, fmt.Errorf("failed to migrate: %w", err)
		}
		store, err := NewPostgresStore(ctx, db)
		if err != nil {
			return nil, nil, err
		}
		return store, nil, nil

	} else if _, hasDocker := os.LookupEnv("DOCKER_HOST"); hasDocker {
		pool, err := dockertest.NewPool("")
		if err != nil {
			return nil, nil, err
		}
		resource, err := pool.RunWithOptions(&dockertest.RunOptions{
			Repository: "postgres",
			Tag:        "15-alpine",
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
		databaseUrl := fmt.Sprintf("postgres://test:test@%s/test?sslmode=disable", resource.GetHostPort("5432/tcp"))

		// This log message is used by the test suite to pass the database URL to the testproxy.
		logger.Info("using database", zap.String("url", databaseUrl))

		var db *pgxpool.Pool
		pool.MaxWait = 120 * time.Second
		if err = pool.Retry(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
			defer cancel()
			db, err = pgxpool.New(ctx, databaseUrl)
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

		store, err := NewPostgresStore(ctx, db)
		if err != nil {
			return nil, nil, err
		}
		return store, flushed, nil
	}
	return nil, nil, fmt.Errorf("no database configured export DATABASE_URL or DOCKER_HOST to run locally")
}
