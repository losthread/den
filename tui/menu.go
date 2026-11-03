// File: tui/menu.go
// Description: Renders and controls Den's terminal interface.
// Author: Parth Naik
//
// Package tui renders Den's terminal interface.
package tui

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"den/shared/identity"
)

const banner = `
  ██████╗  ███████╗ ███╗   ██╗
  ██╔══██╗ ██╔════╝ ████╗  ██║
  ██║  ██║ █████╗   ██╔██╗ ██║
  ██║  ██║ ██╔══╝   ██║╚██╗██║
  ██████╔╝ ███████╗ ██║ ╚████║
  ╚═════╝  ╚══════╝ ╚═╝  ╚═══╝
`

var options = []string{"Direct Chat", "Group Chat", "Identity", "Settings", "Quit"}
var identityOptions = []string{"Change Username", "Show Public Key", "Export Identity", "Import Identity", "Back"}
var settingsOptions = []string{"Bluetooth           ON", "WebRTC              ON", "Mesh Relaying       ON", "Max Relay Hops      5", "Peer Discovery      ON", "Incoming Requests   ASK", "Timestamps          12H", "Back"}

var use24HourTimestamps bool
var db *sql.DB

// currentUsername returns the stored username or anonymous.
func currentUsername() string {
	if db == nil {
		return "anonymous"
	}
	username, _, _, err := identity.GetIdentity(db)
	if err != nil {
		return "anonymous"
	}
	return username
}

// toggleSetting switches a setting between two labels.
func toggleSetting(items []string, index int, on, off string) {
	if strings.HasSuffix(items[index], on) {
		items[index] = strings.TrimSuffix(items[index], on) + off
	} else {
		items[index] = strings.TrimSuffix(items[index], off) + on
	}
}

// SetDB configures the database used by the TUI.
func SetDB(database *sql.DB) { db = database }

// Run starts the interactive terminal interface.
func Run() error {
	if !isTerminal() {
		return fmt.Errorf("interactive TUI requires a terminal")
	}
	state, err := makeRaw()
	if err != nil {
		return err
	}
	defer restore(state)

	selected := 0
	for {
		render(selected)
		key, err := readKey()
		if err != nil {
			return err
		}
		switch key {
		case "up", "k":
			selected = (selected + len(options) - 1) % len(options)
		case "down", "j":
			selected = (selected + 1) % len(options)
		case "q", "Q":
			return nil
		case "\r", "\n":
			switch options[selected] {
			case "Identity":
				if err := submenu("IDENTITY", identityOptions, identityRender); err != nil {
					return err
				}
			case "Settings":
				if err := submenu("SETTINGS", settingsOptions, settingsRender); err != nil {
					return err
				}
			case "Quit":
				return nil
			}
		}
	}
}

// render draws the main menu.
func render(selected int) {

	const white, offWhite, yellow, reset = "\x1b[38;2;255;255;255m", "\x1b[97m", "\x1b[33m", "\x1b[0m"
	fmt.Print("\x1b[2J\x1b[H")

	fmt.Printf("%s%s%s", white, banner, reset)
	fmt.Println()
	fmt.Printf("%sDecentralized, Private chatting.%s\n", white, reset)
	fmt.Println("────────────────────────────────────")
	for i, option := range options {
		marker, color := "   ", offWhite
		if i == selected {
			marker, color = yellow+">  "+reset, white
		}
		fmt.Printf("%s%s%02d  %s%s\n", marker, color, i+1, option, reset)
	}
	fmt.Println("────────────────────────────────────")
	fmt.Println()
	fmt.Printf("%sBLE: ON  ·  WebRTC: READY  ·  PEERS: 0\n", white)
	fmt.Printf("↑↓ Navigate  ·  ENTER Select  ·  Q Quit%s\n", reset)
	fmt.Println()
	fmt.Println("DEN v0.1.0")
}

type submenuRenderer func([]string, int)

// submenu runs a selectable submenu.
func submenu(title string, items []string, draw submenuRenderer) error {
	selected := 0
	for {
		fmt.Print("\x1b[2J\x1b[H")
		draw(items, selected)
		key, err := readKey()
		if err != nil {
			return err
		}
		switch key {
		case "up", "k":
			selected = (selected + len(items) - 1) % len(items)
		case "down", "j":
			selected = (selected + 1) % len(items)
		case "q", "Q":
			return nil
		case "\r", "\n":
			if title == "SETTINGS" {
				switch selected {
				case 0, 1, 2, 4:
					toggleSetting(items, selected, "ON", "OFF")
					continue
				case 3:
					hops := items[selected][len(items[selected])-1] - '0'
					hops = hops%10 + 1
					items[selected] = fmt.Sprintf("Max Relay Hops      %d", hops)
					continue
				case 5:
					toggleSetting(items, selected, "ACCEPT", "ASK")
					continue
				case 6:
					use24HourTimestamps = !use24HourTimestamps
					if use24HourTimestamps {
						items[selected] = "Timestamps          24H"
					} else {
						items[selected] = "Timestamps          12H"
					}
					continue
				}
			}
			if title == "IDENTITY" && items[selected] == "Change Username" {
				renderUsernameChange()
				continue
			}
			if title == "IDENTITY" && items[selected] == "Show Public Key" {
				showPublicKey()
				continue
			}
			if title == "IDENTITY" && items[selected] == "Export Identity" {
				exportIdentity()
				continue
			}
			if items[selected] == "Back" || title == "IDENTITY" && selected == len(items)-1 {
				return nil
			}
		}
	}
}

// renderUsernameChange reads and saves a new username.
func renderUsernameChange() {
	const white, reset = "\x1b[38;2;255;255;255m", "\x1b[0m"
	fmt.Print("\x1b[2J\x1b[H")
	fmt.Printf("%sCHANGE USERNAME%s\n", white, reset)
	fmt.Printf("%s────────────────────────────────────%s\n", white, reset)
	fmt.Printf("%sCurrent username: %s%s\n", white, currentUsername(), reset)
	fmt.Printf("%sNew username: %s", white, reset)

	var input strings.Builder
	for {
		key, err := readKey()
		if err != nil {
			return
		}
		switch key {
		case "\r", "\n":
			if db == nil {
				return
			}
			if err := identity.ChangeUsername(db, input.String()); err != nil {
				fmt.Printf("\n%sUnable to change username: %v%s\nPress any key...", white, err, reset)
				_, _ = readKey()
			}
			return
		case "\x7f":
			value := input.String()
			if value != "" {
				value = value[:len(value)-1]
				input.Reset()
				input.WriteString(value)
				fmt.Print("\b \b")
			}
		default:
			if len(key) == 1 && key[0] >= ' ' {
				input.WriteString(key)
				fmt.Print(key)
			}
		}
	}
}

// exportIdentity displays the suggested identity export path.
func exportIdentity() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	downloads := filepath.Join(home, "Downloads")
	if runtime.GOOS == "windows" {
		if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
			downloads = filepath.Join(userProfile, "Downloads")
		}
	}
	path := filepath.Join(downloads, "den-identity.enc")
	fmt.Print("\x1b[2J\x1b[H")
	fmt.Println("EXPORT IDENTITY")
	fmt.Println("────────────────────────────────────")
	fmt.Printf("Suggested location:\n%s\n", path)
	fmt.Println()
	fmt.Println("Press any key to return")
	var key [1]byte
	_, _ = os.Stdin.Read(key[:])
}

// showPublicKey displays the public key screen.
func showPublicKey() {
	fmt.Print("\x1b[2J\x1b[H")
	fmt.Println("PUBLIC KEY")
	fmt.Println("────────────────────────────────────")
	fmt.Println("ed25519:ab82...f12e")
	fmt.Println()
	fmt.Println("Press any key to return")
	var key [1]byte
	_, _ = os.Stdin.Read(key[:])
}

// identityRender draws the identity submenu.
func identityRender(items []string, selected int) {
	const white, offWhite, reset = "\x1b[38;2;255;255;255m", "\x1b[97m", "\x1b[0m"
	fmt.Printf("%sIDENTITY%s\n", white, reset)
	fmt.Println("────────────────────────────────────")
	fmt.Printf("%sUsername      %s\n", white, currentUsername())
	fmt.Printf("Peer ID       7f3a...91bc\n")
	fmt.Printf("Public Key    ed25519:ab82...f12e%s\n", reset)
	fmt.Println()
	menuRender(items, selected)
	fmt.Println("────────────────────────────────────")
}

// settingsRender draws the settings submenu.
func settingsRender(items []string, selected int) {
	const white, offWhite, reset = "\x1b[38;2;255;255;255m", "\x1b[97m", "\x1b[0m"
	fmt.Printf("%sSETTINGS%s\n", white, reset)
	fmt.Println("────────────────────────────────────")
	fmt.Printf("%sNETWORK%s\n", white, reset)
	menuRender(items[:4], selected)
	fmt.Println()
	fmt.Printf("%sPRIVACY%s\n", white, reset)
	menuRender(items[4:6], selected-4)
	fmt.Println()
	fmt.Printf("%sAPPEARANCE%s\n", white, reset)
	menuRender(items[6:], selected-6)
	fmt.Println("────────────────────────────────────")
}

// menuRender draws selectable menu items.
func menuRender(items []string, selected int) {
	const white, offWhite, yellow, reset = "\x1b[38;2;255;255;255m", "\x1b[97m", "\x1b[33m", "\x1b[0m"
	for i, item := range items {
		marker, color := "   ", offWhite
		if i == selected {
			marker, color = yellow+">  "+reset, white
		}
		fmt.Printf("%s%s%02d  %s%s\n", marker, color, i+1, item, reset)
	}
}

// readKey reads one keyboard input, including arrow keys.
func readKey() (string, error) {
	var b [3]byte
	n, err := os.Stdin.Read(b[:])
	if err != nil {
		return "", err
	}
	key := string(b[:n])
	if key == "\x1b[A" {
		return "up", nil
	}
	if key == "\x1b[B" {
		return "down", nil
	}
	return key, nil
}

type terminalState struct{ term syscall.Termios }

// isTerminal reports whether standard input is a terminal.
func isTerminal() bool {
	var term syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(os.Stdin.Fd()), uintptr(syscall.TIOCGETA), uintptr(unsafe.Pointer(&term)))
	return errno == 0
}

// makeRaw switches standard input to raw mode.
func makeRaw() (terminalState, error) {
	var state terminalState
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(os.Stdin.Fd()), uintptr(syscall.TIOCGETA), uintptr(unsafe.Pointer(&state.term)))
	if errno != 0 {
		return state, errno
	}
	raw := state.term
	raw.Lflag &^= syscall.ECHO | syscall.ICANON
	raw.Cc[syscall.VMIN], raw.Cc[syscall.VTIME] = 1, 0
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(os.Stdin.Fd()), uintptr(syscall.TIOCSETA), uintptr(unsafe.Pointer(&raw)))
	if errno != 0 {
		return state, errno
	}
	return state, nil
}

// restore returns the terminal to its previous state.
func restore(state terminalState) {
	_, _, _ = syscall.Syscall(syscall.SYS_IOCTL, uintptr(os.Stdin.Fd()), uintptr(syscall.TIOCSETA), uintptr(unsafe.Pointer(&state.term)))
	fmt.Print("\x1b[0m\n")
}
