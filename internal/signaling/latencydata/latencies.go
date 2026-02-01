package latencydata

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	_ "embed"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const DataVersion = 1

//go:embed latencies.csv
var latencyCSV []byte

// EnsureLatencyData ensures that the latency data is present and up to date in the database.
func EnsureLatencyData(ctx context.Context, pool *pgxpool.Pool) error {
	defer func() {
		// Clear the embedded data from memory after use.
		latencyCSV = nil
	}()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire db connection: %w", err)
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		INSERT INTO latency_meta (id, version)
		VALUES (1, 0)
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("ensure latency_meta row: %w", err)
	}

	var currentVersion int
	if err := tx.QueryRow(ctx, `
		SELECT version
		FROM latency_meta
		WHERE id = 1 FOR UPDATE
	`).Scan(&currentVersion); err != nil {
		return fmt.Errorf("read latency_meta version: %w", err)
	}

	if currentVersion == DataVersion {
		return tx.Commit(ctx)
	}

	rows, err := loadRows()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		TRUNCATE TABLE latencies
	`); err != nil {
		return fmt.Errorf("truncate latencies: %w", err)
	}

	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"latencies"}, []string{
		"from_country",
		"from_region",
		"to_country",
		"to_region",
		"latency_ms_p50",
	}, pgx.CopyFromRows(rows)); err != nil {
		return fmt.Errorf("copy latencies: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE latency_meta
		SET version = $1
		WHERE id = 1
	`, DataVersion); err != nil {
		return fmt.Errorf("update latency_meta version: %w", err)
	}

	return tx.Commit(ctx)
}

// loadRows loads the latency data from the embedded CSV file.
func loadRows() ([][]any, error) {
	data := latencyCSV
	if len(data) == 0 {
		return nil, fmt.Errorf("read latencies.csv: no data")
	}

	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1

	if _, err := reader.Read(); err != nil {
		return nil, fmt.Errorf("read latencies.csv header: %w", err)
	}

	rows := make([][]any, 0, 512)
	line := 1
	for {
		line++
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("read latencies.csv line %d: %w", line, err)
		}
		if len(record) < 5 {
			return nil, fmt.Errorf("read latencies.csv line %d: expected 5 columns, got %d", line, len(record))
		}

		fromCountry := strings.TrimSpace(record[0])
		toCountry := strings.TrimSpace(record[2])
		if fromCountry == "" || toCountry == "" {
			return nil, fmt.Errorf("read latencies.csv line %d: missing country", line)
		}

		fromRegionRaw := strings.TrimSpace(record[1])
		toRegionRaw := strings.TrimSpace(record[3])

		var fromRegion any
		if fromRegionRaw != "" {
			fromRegion = fromRegionRaw
		}

		var toRegion any
		if toRegionRaw != "" {
			toRegion = toRegionRaw
		}

		latency, err := strconv.ParseFloat(strings.TrimSpace(record[4]), 64)
		if err != nil {
			return nil, fmt.Errorf("read latencies.csv line %d: parse latency: %w", line, err)
		}

		rows = append(rows, []any{fromCountry, fromRegion, toCountry, toRegion, latency})
	}

	return rows, nil
}
