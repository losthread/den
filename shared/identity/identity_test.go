// File: shared/identity/identity_test.go
// Description: Tests local identity database operations.
// Author: Parth Naik

package identity

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
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

	value, err := GetIdentity(db)
	if err != nil || value.Username != "anonymous" || len(value.PeerID) != 1 || len(value.PublicKey) != 1 {
		t.Fatalf("GetIdentity() = %+v, %v", value, err)
	}
	if err := ChangeUsername(db, "  Alice  "); err != nil {
		t.Fatal(err)
	}
	value, err = GetIdentity(db)
	if err != nil || value.Username != "Alice" {
		t.Fatalf("updated username = %q, %v", value.Username, err)
	}
	if err := ChangeUsername(db, "   "); err == nil {
		t.Fatal("empty username accepted")
	}
}

func identityTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE identity (peer_id BLOB PRIMARY KEY, username TEXT NOT NULL, public_key BLOB NOT NULL, encrypted_private_key BLOB NOT NULL, encryption_key BLOB NOT NULL, salt BLOB NOT NULL, nonce BLOB NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCreateIdentityReplacesLegacyPlaceholder(t *testing.T) {
	db := identityTestDB(t)
	defer db.Close()
	_, err := db.Exec(`INSERT INTO identity (peer_id, username, public_key, encrypted_private_key, encryption_key, salt, nonce) VALUES (x'01', 'anonymous', x'02', x'', x'', x'', x'')`)
	if err != nil {
		t.Fatal(err)
	}
	if err := CreateIdentity(db, nil); err != nil {
		t.Fatal(err)
	}
	var encrypted, key, nonce []byte
	if err := db.QueryRow(`SELECT encrypted_private_key, encryption_key, nonce FROM identity`).Scan(&encrypted, &key, &nonce); err != nil {
		t.Fatal(err)
	}
	if len(encrypted) == 0 || len(key) != 32 || len(nonce) == 0 {
		t.Fatal("legacy placeholder was not replaced")
	}
}

func TestCreateIdentityDerivesAndDecrypts(t *testing.T) {
	db := identityTestDB(t)
	defer db.Close()
	if err := CreateIdentity(db, nil); err != nil {
		t.Fatal(err)
	}
	var peerID, publicKey, encrypted, key, nonce []byte
	if err := db.QueryRow(`SELECT peer_id, public_key, encrypted_private_key, encryption_key, nonce FROM identity`).Scan(&peerID, &publicKey, &encrypted, &key, &nonce); err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256(publicKey)
	if string(peerID) != string(want[:]) {
		t.Fatal("peer ID does not derive from public key")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	privateKey, err := aead.Open(nil, nonce, encrypted, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(privateKey) != ed25519.PrivateKeySize || string(ed25519.PrivateKey(privateKey).Public().(ed25519.PublicKey)) != string(publicKey) {
		t.Fatal("decrypted key mismatch")
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	source := identityTestDB(t)
	defer source.Close()
	if err := CreateIdentity(source, nil); err != nil {
		t.Fatal(err)
	}
	if err := ChangeUsername(source, " Alice "); err != nil {
		t.Fatal(err)
	}
	data, err := Export(source)
	if err != nil {
		t.Fatal(err)
	}
	var exported map[string]any
	if err := json.Unmarshal(data, &exported); err != nil {
		t.Fatal(err)
	}
	target := identityTestDB(t)
	defer target.Close()
	if err := CreateIdentity(target, nil); err != nil {
		t.Fatal(err)
	}
	if err := Import(target, data); err != nil {
		t.Fatal(err)
	}
	got, err := GetIdentity(target)
	if err != nil {
		t.Fatal(err)
	}
	want, err := GetIdentity(source)
	if err != nil {
		t.Fatal(err)
	}
	if got.Username != want.Username || string(got.PeerID) != string(want.PeerID) || string(got.PublicKey) != string(want.PublicKey) {
		t.Fatalf("round trip mismatch: got %+v want %+v", got, want)
	}
}

func TestImportRejectsTamperedIdentity(t *testing.T) {
	db := identityTestDB(t)
	defer db.Close()
	if err := CreateIdentity(db, nil); err != nil {
		t.Fatal(err)
	}
	data, err := Export(db)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	value["peer_id"] = "AQ=="
	data, err = json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := Import(db, data); err == nil {
		t.Fatal("tampered identity accepted")
	}
}
