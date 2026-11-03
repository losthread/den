// File: shared/settings/settings.go
// Description: Stores and updates the user's local settings details.
// Author: Parth Naik
//
// Package identity stores the user's local settings details.
package settings

import (
	"database/sql"
	"fmt"
)

// Settings contains the user's configurable preferences.
type Settings struct {
	Bluetooth        bool
	WebRTC           bool
	MeshRelaying     bool
	MaxRelayHops     int
	PeerDiscovery    bool
	Notifications    bool
	IncomingRequests string
	Timestamp        string
}

// GetSettings returns the stored user settings.
func GetSettings(db *sql.DB) (Settings, error) {
	var setting Settings

	err := db.QueryRow(`
		SELECT bluetooth, webrtc, mesh_relaying, max_relay_hops,
			peer_discovery, notifications, incoming_requests, timestamp
		FROM settings WHERE id = 1
	`).Scan(
		&setting.Bluetooth,
		&setting.WebRTC,
		&setting.MeshRelaying,
		&setting.MaxRelayHops,
		&setting.PeerDiscovery,
		&setting.Notifications,
		&setting.IncomingRequests,
		&setting.Timestamp,
	)

	return setting, err
}

// UpdateSettings validates and persists the user's settings.
func UpdateSettings(db *sql.DB, setting Settings) error {
	if setting.MaxRelayHops < 1 || setting.MaxRelayHops > 10 {
		return fmt.Errorf("max relay hops must be between 1 and 10")
	}
	if setting.IncomingRequests != "ASK" && setting.IncomingRequests != "ACCEPT" {
		return fmt.Errorf("invalid incoming requests setting")
	}
	if setting.Timestamp != "24H" && setting.Timestamp != "12H" {
		return fmt.Errorf("invalid timestamp setting")
	}

	_, err := db.Exec(`
		UPDATE settings SET bluetooth = ?, webrtc = ?, mesh_relaying = ?,
			max_relay_hops = ?, peer_discovery = ?, notifications = ?,
			incoming_requests = ?, timestamp = ?
		WHERE id = 1
	`, setting.Bluetooth, setting.WebRTC, setting.MeshRelaying,
		setting.MaxRelayHops, setting.PeerDiscovery, setting.Notifications,
		setting.IncomingRequests, setting.Timestamp)
	return err
}
