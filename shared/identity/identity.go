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

// Identity contains the user's local identity details.
type Identity struct {
	Username  string
	PeerID    []byte
	PublicKey []byte
}

// GetIdentity returns the stored identity.
func GetIdentity(db *sql.DB) (Identity, error) {
	var identity Identity
	err := db.QueryRow(`
		SELECT username, peer_id, public_key
		FROM identity
		LIMIT 1
	`).Scan(&identity.Username, &identity.PeerID, &identity.PublicKey)
	return identity, err
}

// ChangeUsername updates the username when identity is non-empty after trimming.
// ChangeUsername validates and stores a trimmed username.
func ChangeUsername(db *sql.DB, identity string) error {
	identity = strings.TrimSpace(identity)

	if identity == "" {
		return fmt.Errorf("username cannot be empty")
	}

	result, err := db.Exec(`
		UPDATE identity
		SET username = ?
	`, identity)
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
