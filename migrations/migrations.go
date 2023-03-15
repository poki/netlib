package migrations

import (
	"embed"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	driver "github.com/golang-migrate/migrate/v4/database/pgx/v5"
)

//go:embed *.sql
var migrations embed.FS

func Up(config *pgx.ConnConfig) error {
	d, err := iofs.New(migrations, ".")
	if err != nil {
		return err
	}

	db := stdlib.OpenDB(*config)
	driver, err := driver.WithInstance(db, &driver.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", d, "pgx", driver)
	if err != nil {
		return err
	}
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}
