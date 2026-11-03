// File: shared/identity/identity_test.go
// Description: Tests local identity database operations.
// Author: Parth Naik

package identity

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// TestIdentityDatabaseOperations verifies identity reads and updates.
func TestIdentityDatabaseOperations(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE identity (peer_id BLOB PRIMARY KEY, username TEXT NOT NULL, public_key BLOB NOT NULL, encrypted_private_key BLOB NOT NULL, salt BLOB NOT NULL, nonce BLOB NOT NULL); INSERT INTO identity (peer_id, username, public_key, encrypted_private_key, salt, nonce) VALUES (x'01', 'anonymous', x'02', x'03', x'04', x'05')`)
	if err != nil {
		t.Fatal(err)
	}

	username, peerID, publicKey, err := GetIdentity(db)
	if err != nil || username != "anonymous" || len(peerID) != 1 || len(publicKey) != 1 {
		t.Fatalf("GetIdentity() = %q, %x, %x, %v", username, peerID, publicKey, err)
	}
	if err := ChangeUsername(db, "  Alice  "); err != nil {
		t.Fatal(err)
	}
	username, _, _, err = GetIdentity(db)
	if err != nil || username != "Alice" {
		t.Fatalf("updated username = %q, %v", username, err)
	}
	if err := ChangeUsername(db, "   "); err == nil {
		t.Fatal("empty username accepted")
	}
}
