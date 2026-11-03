// File: shared/identity/identity.go
// Description: Stores and updates the user's local identity details.
// Author: Parth Naik
//
// Package identity stores the user's local identity details.
package identity

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// Identity contains the user's local identity details.
type Identity struct {
	Username  string `json:"username"`
	PeerID    []byte `json:"peer_id"`
	PublicKey []byte `json:"public_key"`
}

// Export serializes the identity and encrypted private-key material.
func Export(db *sql.DB) ([]byte, error) {
	var value struct {
		Identity
		EncryptedPrivateKey []byte `json:"encrypted_private_key"`
		EncryptionKey       []byte `json:"encryption_key"`
		Nonce               []byte `json:"nonce"`
	}
	row := db.QueryRow(`SELECT username, peer_id, public_key, encrypted_private_key, encryption_key, nonce FROM identity LIMIT 1`)
	if err := row.Scan(&value.Username, &value.PeerID, &value.PublicKey, &value.EncryptedPrivateKey, &value.EncryptionKey, &value.Nonce); err != nil {
		return nil, err
	}
	if strings.TrimSpace(value.Username) == "" || len(value.PeerID) != sha256.Size || len(value.PublicKey) != ed25519.PublicKeySize || len(value.EncryptionKey) != 32 || len(value.EncryptedPrivateKey) == 0 || len(value.Nonce) == 0 {
		return nil, fmt.Errorf("identity has no valid key material")
	}
	return json.Marshal(value)
}

// Import replaces the local identity with a validated exported identity.
func Import(db *sql.DB, data []byte) error {
	var value struct {
		Identity
		EncryptedPrivateKey []byte `json:"encrypted_private_key"`
		EncryptionKey       []byte `json:"encryption_key"`
		Nonce               []byte `json:"nonce"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if strings.TrimSpace(value.Username) == "" || len(value.PeerID) != sha256.Size || len(value.PublicKey) != ed25519.PublicKeySize || len(value.EncryptionKey) != 32 || len(value.EncryptedPrivateKey) == 0 {
		return fmt.Errorf("invalid identity export")
	}
	block, err := aes.NewCipher(value.EncryptionKey)
	if err != nil {
		return fmt.Errorf("invalid identity export: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil || len(value.Nonce) != aead.NonceSize() {
		return fmt.Errorf("invalid identity export")
	}
	privateKey, err := aead.Open(nil, value.Nonce, value.EncryptedPrivateKey, nil)
	if err != nil || len(privateKey) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid identity export")
	}
	publicKey := ed25519.PrivateKey(privateKey).Public().(ed25519.PublicKey)
	if !equalBytes(publicKey, value.PublicKey) {
		return fmt.Errorf("public key does not match private key")
	}
	peerID := sha256.Sum256(value.PublicKey)
	if !equalBytes(peerID[:], value.PeerID) {
		return fmt.Errorf("peer ID does not match public key")
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`DELETE FROM identity`); err != nil {
		return err
	}
	if _, err = tx.Exec(`INSERT INTO identity (peer_id, username, public_key, encrypted_private_key, encryption_key, salt, nonce) VALUES (?, ?, ?, ?, ?, x'', ?)`, value.PeerID, strings.TrimSpace(value.Username), value.PublicKey, value.EncryptedPrivateKey, value.EncryptionKey, value.Nonce); err != nil {
		return err
	}
	return tx.Commit()
}

func equalBytes(a, b []byte) bool {
	return len(a) == len(b) && string(a) == string(b)
}

// PublicKeyText returns the public key as a stable hexadecimal string.
func PublicKeyText(db *sql.DB) (string, error) {
	value, err := GetIdentity(db)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(value.PublicKey), nil
}

// CreateIdentity creates and stores an Ed25519 identity with its private key encrypted by AES-GCM.
func CreateIdentity(db *sql.DB, password []byte) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM identity").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		var encrypted, key, nonce []byte
		if err := db.QueryRow("SELECT encrypted_private_key, encryption_key, nonce FROM identity LIMIT 1").Scan(&encrypted, &key, &nonce); err != nil {
			return err
		}
		if len(encrypted) > 0 && len(key) == 32 && len(nonce) > 0 {
			return nil
		}
		if _, err := db.Exec("DELETE FROM identity"); err != nil {
			return err
		}
	}
	seed := make([]byte, ed25519.SeedSize)
	if _, err := rand.Read(seed); err != nil {
		return err
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return err
	}
	// block is a object that implements cipher.Block for AES encryption operations
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	// aead is an object to encrypt and decrypt data securely
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	// nonce - number used once, added to text to make same text give random ciphertext
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	encrypted := aead.Seal(nil, nonce, privateKey, nil)
	peerID := sha256.Sum256(publicKey)
	_, err = db.Exec(`
		INSERT INTO identity
		(peer_id, username, public_key, encrypted_private_key, encryption_key, salt, nonce)
		VALUES (?, 'anonymous', ?, ?, ?, x'', ?)`, peerID[:], publicKey, encrypted, key, nonce)
	return err
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
