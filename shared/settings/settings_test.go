// File: shared/settings/settings_test.go
// Description: Tests local settings database operations.
// Author: Parth Naik

package settings

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// TestGetSettings verifies settings are read into a struct.
func TestGetSettings(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE settings (id INTEGER PRIMARY KEY, bluetooth INTEGER, webrtc INTEGER, mesh_relaying INTEGER, max_relay_hops INTEGER, peer_discovery INTEGER, notifications INTEGER, incoming_requests TEXT, timestamp TEXT); INSERT INTO settings VALUES (1, 1, 0, 1, 7, 1, 0, 'ASK', '12H')`)
	if err != nil {
		t.Fatal(err)
	}
	value, err := GetSettings(db)
	if err != nil || !value.Bluetooth || value.WebRTC || value.MaxRelayHops != 7 || value.Notifications || value.Timestamp != "12H" {
		t.Fatalf("GetSettings() = %+v, %v", value, err)
	}
}
