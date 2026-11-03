// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2026 Tulir Asokan, buihongduc132
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

package termimg

import (
	"fmt"
	"testing"
)

func clearEnvs(t *testing.T) {
	t.Helper()
	envs := []string{
		"TERM_PROGRAM",
		"WEZTERM_PANE",
		"KITTY_WINDOW_ID",
		"GHOSTTY_RESOURCES_DIR",
		"LC_TERMINAL",
		"ITERM_SESSION_ID",
		"TMUX",
		"STY",
	}
	for _, env := range envs {
		t.Setenv(env, "")
	}
}

func TestDetectTerminal_WezTerm(t *testing.T) {
	t.Run("via TERM_PROGRAM", func(t *testing.T) {
		clearEnvs(t)
		t.Setenv("TERM_PROGRAM", "WezTerm")

		cap := DetectTerminal()
		if !cap.IsWezTerm {
			t.Errorf("expected IsWezTerm to be true")
		}
		if cap.Protocol != ProtocolKitty {
			t.Errorf("expected Protocol to be %v, got %v", ProtocolKitty, cap.Protocol)
		}
	})

	t.Run("via WEZTERM_PANE", func(t *testing.T) {
		clearEnvs(t)
		t.Setenv("WEZTERM_PANE", "42")

		cap := DetectTerminal()
		if !cap.IsWezTerm {
			t.Errorf("expected IsWezTerm to be true")
		}
		if cap.Protocol != ProtocolKitty {
			t.Errorf("expected Protocol to be %v, got %v", ProtocolKitty, cap.Protocol)
		}
	})
}

func TestDetectTerminal_Kitty(t *testing.T) {
	t.Run("via KITTY_WINDOW_ID", func(t *testing.T) {
		clearEnvs(t)
		t.Setenv("KITTY_WINDOW_ID", "1")

		cap := DetectTerminal()
		if !cap.IsKitty {
			t.Errorf("expected IsKitty to be true")
		}
		if cap.Protocol != ProtocolKitty {
			t.Errorf("expected Protocol to be %v, got %v", ProtocolKitty, cap.Protocol)
		}
	})
}

func TestDetectTerminal_ITerm2(t *testing.T) {
	t.Run("via TERM_PROGRAM", func(t *testing.T) {
		clearEnvs(t)
		t.Setenv("TERM_PROGRAM", "iTerm.app")

		cap := DetectTerminal()
		if !cap.IsITerm2 {
			t.Errorf("expected IsITerm2 to be true")
		}
		if cap.Protocol != ProtocolITerm2 {
			t.Errorf("expected Protocol to be %v, got %v", ProtocolITerm2, cap.Protocol)
		}
	})

	t.Run("via LC_TERMINAL", func(t *testing.T) {
		clearEnvs(t)
		t.Setenv("LC_TERMINAL", "iTerm2")

		cap := DetectTerminal()
		if !cap.IsITerm2 {
			t.Errorf("expected IsITerm2 to be true")
		}
		if cap.Protocol != ProtocolITerm2 {
			t.Errorf("expected Protocol to be %v, got %v", ProtocolITerm2, cap.Protocol)
		}
	})

	t.Run("via ITERM_SESSION_ID", func(t *testing.T) {
		clearEnvs(t)
		t.Setenv("ITERM_SESSION_ID", "w0t0p0:deadbeef")

		cap := DetectTerminal()
		if !cap.IsITerm2 {
			t.Errorf("expected IsITerm2 to be true")
		}
		if cap.Protocol != ProtocolITerm2 {
			t.Errorf("expected Protocol to be %v, got %v", ProtocolITerm2, cap.Protocol)
		}
	})
}

func TestDetectTerminal_Tmux(t *testing.T) {
	t.Run("tmux without passthrough degrades to halfblocks", func(t *testing.T) {
		clearEnvs(t)
		t.Setenv("TMUX", "/tmp/tmux-1000/default,1234,0")
		t.Setenv("TERM_PROGRAM", "WezTerm")

		oldChecker := tmuxPassthroughChecker
		defer func() { tmuxPassthroughChecker = oldChecker }()
		tmuxPassthroughChecker = func() bool { return false }

		cap := DetectTerminal()
		if !cap.IsTmux {
			t.Errorf("expected IsTmux to be true")
		}
		if cap.TmuxPassthrough {
			t.Errorf("expected TmuxPassthrough to be false")
		}
		if cap.Protocol != ProtocolHalfBlocks {
			t.Errorf("expected Protocol to degrade to %v, got %v", ProtocolHalfBlocks, cap.Protocol)
		}
	})

	t.Run("tmux with passthrough and WezTerm enables kitty", func(t *testing.T) {
		clearEnvs(t)
		t.Setenv("TMUX", "/tmp/tmux-1000/default,1234,0")
		t.Setenv("TERM_PROGRAM", "WezTerm")

		oldChecker := tmuxPassthroughChecker
		defer func() { tmuxPassthroughChecker = oldChecker }()
		tmuxPassthroughChecker = func() bool { return true }

		cap := DetectTerminal()
		if !cap.IsTmux {
			t.Errorf("expected IsTmux to be true")
		}
		if !cap.TmuxPassthrough {
			t.Errorf("expected TmuxPassthrough to be true")
		}
		if cap.Protocol != ProtocolKitty {
			t.Errorf("expected Protocol to be %v, got %v", ProtocolKitty, cap.Protocol)
		}
	})

	t.Run("tmux with allow-passthrough 'all' and WezTerm enables kitty", func(t *testing.T) {
		clearEnvs(t)
		t.Setenv("TMUX", "/tmp/tmux-1000/default,1234,0")
		t.Setenv("TERM_PROGRAM", "WezTerm")

		oldChecker := tmuxPassthroughChecker
		defer func() { tmuxPassthroughChecker = oldChecker }()
		tmuxPassthroughChecker = func() bool { return parseTmuxPassthrough("all") }

		cap := DetectTerminal()
		if !cap.IsTmux {
			t.Errorf("expected IsTmux to be true")
		}
		if !cap.TmuxPassthrough {
			t.Errorf("expected TmuxPassthrough to be true")
		}
		if cap.Protocol != ProtocolKitty {
			t.Errorf("expected Protocol to be %v, got %v", ProtocolKitty, cap.Protocol)
		}
	})

	t.Run("tmux with passthrough and generic terminal stays halfblocks", func(t *testing.T) {
		clearEnvs(t)
		t.Setenv("TMUX", "/tmp/tmux-1000/default,1234,0")

		oldChecker := tmuxPassthroughChecker
		defer func() { tmuxPassthroughChecker = oldChecker }()
		tmuxPassthroughChecker = func() bool { return true }

		cap := DetectTerminal()
		if cap.Protocol != ProtocolHalfBlocks {
			t.Errorf("expected Protocol to be %v, got %v", ProtocolHalfBlocks, cap.Protocol)
		}
	})
}

func TestParseTmuxPassthrough(t *testing.T) {
	cases := []struct {
		input    string
		expected bool
	}{
		{"on", true},
		{"ON", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"all", true},
		{"ALL", true},
		{"  all\n", true},
		{"off", false},
		{"0", false},
		{"no", false},
		{"", false},
		{"   ", false},
		{"invalid", false},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("input_%q", tc.input), func(t *testing.T) {
			got := parseTmuxPassthrough(tc.input)
			if got != tc.expected {
				t.Errorf("parseTmuxPassthrough(%q) = %v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestDetectTerminal_Screen(t *testing.T) {
	clearEnvs(t)
	t.Setenv("STY", "12345.pts-0.host")
	t.Setenv("TERM_PROGRAM", "WezTerm")

	cap := DetectTerminal()
	if !cap.IsScreen {
		t.Errorf("expected IsScreen to be true")
	}
	if cap.Protocol != ProtocolHalfBlocks {
		t.Errorf("expected Screen to force %v, got %v", ProtocolHalfBlocks, cap.Protocol)
	}
}

func TestDetectTerminal_GenericFallback(t *testing.T) {
	clearEnvs(t)
	cap := DetectTerminal()
	if cap.Protocol != ProtocolHalfBlocks {
		t.Errorf("expected generic terminal to default to %v, got %v", ProtocolHalfBlocks, cap.Protocol)
	}
}

func TestResolveProtocol_Overrides(t *testing.T) {
	clearEnvs(t)
	t.Setenv("TERM_PROGRAM", "WezTerm")

	cases := []struct {
		name     string
		userPref string
		expected Protocol
	}{
		{"empty string resolves auto (WezTerm -> kitty)", "", ProtocolKitty},
		{"auto resolves auto (WezTerm -> kitty)", "auto", ProtocolKitty},
		{"explicit kitty keeps kitty", "kitty", ProtocolKitty},
		{"explicit halfblocks overrides WezTerm", "halfblocks", ProtocolHalfBlocks},
		{"explicit disabled overrides WezTerm", "disabled", ProtocolDisabled},
		{"explicit iterm2 overrides WezTerm to iterm2", "iterm2", ProtocolITerm2},
		{"case-insensitive and trimmed", "  HALFBLOCKS  ", ProtocolHalfBlocks},
		{"unknown pref defaults to detected", "unknown-proto", ProtocolKitty},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveProtocol(tc.userPref)
			if got != tc.expected {
				t.Errorf("ResolveProtocol(%q) = %v, expected %v", tc.userPref, got, tc.expected)
			}
		})
	}

	// Test with generic terminal
	t.Run("explicit iterm2 forces iterm2 on generic terminal", func(t *testing.T) {
		clearEnvs(t)
		got := ResolveProtocol("iterm2")
		if got != ProtocolITerm2 {
			t.Errorf("expected explicit iterm2 to force %v, got %v", ProtocolITerm2, got)
		}
	})
}

func TestIsTmuxPassthroughSupported_NoTmux(t *testing.T) {
	clearEnvs(t)
	if IsTmuxPassthroughSupported() {
		t.Errorf("expected IsTmuxPassthroughSupported to return false when TMUX is unset")
	}
}
