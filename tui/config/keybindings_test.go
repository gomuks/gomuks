// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2026 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func newTestConfig(t *testing.T) *Config {
	t.Helper()
	dir := t.TempDir()
	cfg := NewConfig()
	cfg.Dir = dir
	cfg.nosave = true
	return cfg
}

func writeKeybindingsFile(t *testing.T, cfg *Config, content string) {
	t.Helper()
	path := filepath.Join(cfg.Dir, "terminal-keybindings.yaml")
	if err := os.MkdirAll(cfg.Dir, 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write keybindings: %v", err)
	}
}

func TestParseKeybindings_Defaults(t *testing.T) {
	cfg := newTestConfig(t)
	cfg.LoadKeybindings()

	cases := []struct {
		name   string
		binds  map[Keybind]string
		key    Keybind
		action string
	}{
		{
			name:   "Ctrl+k opens room search",
			binds:  cfg.Keybindings.Main,
			key:    Keybind{Mod: tcell.ModCtrl, Key: tcell.KeyCtrlK, Ch: 11},
			action: "search_rooms",
		},
		{
			name:   "Enter confirms in modal",
			binds:  cfg.Keybindings.Modal,
			key:    Keybind{Key: tcell.KeyEnter, Ch: 13},
			action: "confirm",
		},
		{
			name:   "Enter sends in room",
			binds:  cfg.Keybindings.Room,
			key:    Keybind{Key: tcell.KeyEnter, Ch: 13},
			action: "send",
		},
		{
			name:   "j selects next in visual mode",
			binds:  cfg.Keybindings.Visual,
			key:    Keybind{Key: tcell.KeyRune, Ch: 'j'},
			action: "select_next",
		},
		{
			name:   "Ctrl+r starts a reply",
			binds:  cfg.Keybindings.Room,
			key:    Keybind{Mod: tcell.ModCtrl, Key: tcell.KeyCtrlR, Ch: 18},
			action: "reply",
		},
		{
			name:   "Alt+r starts a reaction",
			binds:  cfg.Keybindings.Room,
			key:    Keybind{Mod: tcell.ModAlt, Key: tcell.KeyRune, Ch: 'r'},
			action: "react",
		},
		{
			name:   "Alt+c starts a copy",
			binds:  cfg.Keybindings.Room,
			key:    Keybind{Mod: tcell.ModAlt, Key: tcell.KeyRune, Ch: 'c'},
			action: "copy",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.binds[tc.key]; got != tc.action {
				t.Errorf("expected action %q, got %q", tc.action, got)
			}
		})
	}
}

func TestParseKeybindings_EscapeClearsRune(t *testing.T) {
	// Escape decodes with a stray rune; the parser must zero it so lookups by
	// KeyEscape with Ch==0 match what tcell delivers at runtime.
	out := parseKeybindings(map[string]string{"Escape": "cancel"})
	if got := out[Keybind{Key: tcell.KeyEscape, Ch: 0}]; got != "cancel" {
		t.Errorf("expected Escape to map to cancel with zeroed rune, got %q", got)
	}
}

func TestParseKeybindings_InvalidShortcutIsSkipped(t *testing.T) {
	// A typo in a user's keybindings file must not take down the client.
	out := parseKeybindings(map[string]string{
		"Ctrl+k":          "search_rooms",
		"NotARealKey+Zzz": "explode",
	})
	// Valid bindings must survive an invalid one.
	if got := out[Keybind{Mod: tcell.ModCtrl, Key: tcell.KeyCtrlK, Ch: 11}]; got != "search_rooms" {
		t.Errorf("expected valid binding to survive, got %q", got)
	}
	// Invalid binding must be absent from the map.
	for kb, action := range out {
		if action == "explode" {
			t.Errorf("invalid binding was not skipped: %+v -> %s", kb, action)
		}
	}
}

func TestLoadKeybindings_UserOverridesDefaults(t *testing.T) {
	cfg := newTestConfig(t)
	writeKeybindingsFile(t, cfg, "main:\n    'Ctrl+g': search_rooms\n")
	cfg.LoadKeybindings()

	if got := cfg.Keybindings.Main[Keybind{Mod: tcell.ModCtrl, Key: tcell.KeyCtrlG, Ch: 7}]; got != "search_rooms" {
		t.Errorf("expected user binding Ctrl+g to be applied, got %q", got)
	}
}

func TestLoadKeybindings_UserFileKeepsUntouchedSections(t *testing.T) {
	cfg := newTestConfig(t)
	writeKeybindingsFile(t, cfg, "main:\n    'Ctrl+g': search_rooms\n")
	cfg.LoadKeybindings()

	// Overriding `main` must not wipe the default `modal` bindings.
	if got := cfg.Keybindings.Modal[Keybind{Key: tcell.KeyEnter, Ch: 13}]; got != "confirm" {
		t.Errorf("expected default modal binding to survive, got %q", got)
	}
}

func TestLoadKeybindings_MalformedFileDoesNotPanic(t *testing.T) {
	cfg := newTestConfig(t)
	writeKeybindingsFile(t, cfg, "main:\n    'TotallyBogus+!!': nonsense\n")

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("LoadKeybindings panicked on a malformed user file: %v", r)
		}
	}()
	cfg.LoadKeybindings()

	// Defaults must still be usable after rejecting the bad entry.
	if got := cfg.Keybindings.Modal[Keybind{Key: tcell.KeyEnter, Ch: 13}]; got != "confirm" {
		t.Errorf("expected defaults to remain loaded, got %q", got)
	}
}
