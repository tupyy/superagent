package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	_ "github.com/marcboeker/go-duckdb"
)

//go:embed schema.sql
var schema string

func NewDB(path string) (*sql.DB, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("opening duckdb at %q: %w", path, err)
	}
	db.SetMaxOpenConns(1)

	if _, err := db.ExecContext(context.Background(), schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("applying schema: %w", err)
	}
	return db, nil
}
