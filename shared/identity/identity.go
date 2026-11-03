// File: shared/identity/identity.go
// Description: Stores and updates the user's local identity details.
// Author: Parth Naik
//
// Package identity stores the user's local identity details.
package identity

import (
	"database/sql"
	"fmt"
	"strings"
)

// GetIdentity returns the stored username, peer ID, and public key.
func GetIdentity(db *sql.DB) (string, []byte, []byte, error) {
	var username string
	var peerID []byte
	var publicKey []byte

	err := db.QueryRow(`
		SELECT username, peer_id, public_key
		FROM identity
		LIMIT 1
	`).Scan(&username, &peerID, &publicKey)

	return username, peerID, publicKey, err
}

// ChangeUsername updates the username when value is non-empty after trimming.
// ChangeUsername validates and stores a trimmed username.
func ChangeUsername(db *sql.DB, value string) error {
	value = strings.TrimSpace(value)

	if value == "" {
		return fmt.Errorf("username cannot be empty")
	}

	result, err := db.Exec(`
		UPDATE identity
		SET username = ?
	`, value)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("identity not found")
	}
	return nil
}
