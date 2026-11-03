// File: cmd/tui/main.go
// Description: Initializes Den's database and starts the TUI.
// Author: Parth Naik
//
// Package main is the entrypoint to DEN's TUI.
package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"den/tui"
	_ "modernc.org/sqlite"
)

// isDuplicateColumn reports whether a migration column already exists.
func isDuplicateColumn(err error) bool {
	return strings.Contains(err.Error(), "duplicate column name")
}

// main opens the database, initializes identity, and starts the TUI.
func main() {
	if err := os.MkdirAll("data", 0o700); err != nil {
		fmt.Fprintln(os.Stderr, "database:", err)
		os.Exit(1)
	}
	db, err := sql.Open("sqlite", "data/den.db")
	if err != nil {
		fmt.Fprintln(os.Stderr, "database:", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		fmt.Fprintln(os.Stderr, "database:", err)
		os.Exit(1)
	}
	schema, err := os.ReadFile("shared/db/schema.sql")
	if err != nil {
		fmt.Fprintln(os.Stderr, "database schema:", err)
		os.Exit(1)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		fmt.Fprintln(os.Stderr, "database schema:", err)
		os.Exit(1)
	}
	if _, err := db.Exec("ALTER TABLE settings ADD COLUMN notifications INTEGER NOT NULL DEFAULT 1 CHECK (notifications IN (0, 1))"); err != nil && !isDuplicateColumn(err) {
		fmt.Fprintln(os.Stderr, "database migration:", err)
		os.Exit(1)
	}
	for _, column := range []string{"encrypted_private_key", "salt", "nonce"} {
		if _, err := db.Exec("ALTER TABLE identity ADD COLUMN " + column + " BLOB NOT NULL DEFAULT x''"); err != nil && !isDuplicateColumn(err) {
			fmt.Fprintln(os.Stderr, "database migration:", err)
			os.Exit(1)
		}
	}
	if _, err := db.Exec(`
		INSERT INTO identity (peer_id, username, public_key, encrypted_private_key, salt, nonce)
		SELECT randomblob(16), 'anonymous', randomblob(32), x'', x'', x''
		WHERE NOT EXISTS (SELECT 1 FROM identity)
	`); err != nil {
		fmt.Fprintln(os.Stderr, "database identity:", err)
		os.Exit(1)
	}
	if _, err := db.Exec(`
		INSERT INTO settings (id)
		SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM settings WHERE id = 1)
	`); err != nil {
		fmt.Fprintln(os.Stderr, "database settings:", err)
		os.Exit(1)
	}
	tui.SetDB(db)

	if err := tui.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "TUI:", err)
		os.Exit(1)
	}
}
