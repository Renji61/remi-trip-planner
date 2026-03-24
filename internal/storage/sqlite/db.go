package sqlite

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func OpenAndMigrate(dbPath, migrationFile string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	if _, err = db.Exec(`PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;`); err != nil {
		return nil, err
	}
	migrationSQL, err := os.ReadFile(migrationFile)
	if err != nil {
		return nil, err
	}
	if _, err = db.Exec(string(migrationSQL)); err != nil {
		return nil, err
	}
	return db, nil
}
