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
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Protocol identifies the graphics rendering protocol.
type Protocol string

const (
	ProtocolAuto       Protocol = "auto"
	ProtocolKitty      Protocol = "kitty"
	ProtocolITerm2     Protocol = "iterm2"
	ProtocolHalfBlocks Protocol = "halfblocks"
	ProtocolDisabled   Protocol = "disabled"
)

// TerminalCapability summarizes detected emulator features.
type TerminalCapability struct {
	Protocol        Protocol
	IsWezTerm       bool
	IsKitty         bool
	IsITerm2        bool
	IsTmux          bool
	IsScreen        bool
	TmuxPassthrough bool
}

// tmuxPassthroughChecker allows mocking or overriding tmux check in tests while keeping
// default execution running genuine tmux command.
var tmuxPassthroughChecker = defaultCheckTmuxPassthrough

func parseTmuxPassthrough(out string) bool {
	val := strings.ToLower(strings.TrimSpace(out))
	return val == "on" || val == "1" || val == "yes" || val == "all"
}

func defaultCheckTmuxPassthrough() bool {
	if os.Getenv("TMUX") == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "tmux", "show", "-gv", "allow-passthrough")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return parseTmuxPassthrough(string(out))
}

// IsTmuxPassthroughSupported checks if tmux allow-passthrough option is enabled.
func IsTmuxPassthroughSupported() bool {
	return tmuxPassthroughChecker()
}

// DetectTerminal inspects the environment to determine terminal capabilities and protocol.
func DetectTerminal() TerminalCapability {
	termProg := os.Getenv("TERM_PROGRAM")
	wezPane := os.Getenv("WEZTERM_PANE")
	kittyWindowID := os.Getenv("KITTY_WINDOW_ID")
	ghosttyRes := os.Getenv("GHOSTTY_RESOURCES_DIR")
	lcTerm := os.Getenv("LC_TERMINAL")
	itermSession := os.Getenv("ITERM_SESSION_ID")
	tmux := os.Getenv("TMUX")
	sty := os.Getenv("STY")

	isWezTerm := termProg == "WezTerm" || wezPane != ""
	isKitty := kittyWindowID != "" || ghosttyRes != "" || termProg == "ghostty" || termProg == "kitty"
	isITerm2 := termProg == "iTerm.app" || lcTerm == "iTerm2" || itermSession != ""
	isTmux := tmux != ""
	isScreen := sty != ""
	tmuxPassthrough := false

	if isTmux {
		tmuxPassthrough = IsTmuxPassthroughSupported()
	}

	var proto Protocol
	if isScreen {
		// GNU Screen does not reliably support graphics passthrough
		proto = ProtocolHalfBlocks
	} else if isTmux {
		if tmuxPassthrough && (isWezTerm || isKitty) {
			proto = ProtocolKitty
		} else if tmuxPassthrough && isITerm2 {
			proto = ProtocolITerm2
		} else {
			proto = ProtocolHalfBlocks
		}
	} else if isWezTerm || isKitty {
		proto = ProtocolKitty
	} else if isITerm2 {
		proto = ProtocolITerm2
	} else {
		proto = ProtocolHalfBlocks
	}

	return TerminalCapability{
		Protocol:        proto,
		IsWezTerm:       isWezTerm,
		IsKitty:         isKitty,
		IsITerm2:        isITerm2,
		IsTmux:          isTmux,
		IsScreen:        isScreen,
		TmuxPassthrough: tmuxPassthrough,
	}
}

// ResolveProtocol resolves the effective Protocol given a user preference string.
func ResolveProtocol(userPref string) Protocol {
	pref := strings.ToLower(strings.TrimSpace(userPref))
	switch Protocol(pref) {
	case ProtocolKitty:
		return ProtocolKitty
	case ProtocolITerm2:
		return ProtocolITerm2
	case ProtocolHalfBlocks:
		return ProtocolHalfBlocks
	case ProtocolDisabled:
		return ProtocolDisabled
	case "", ProtocolAuto:
		return DetectTerminal().Protocol
	default:
		return DetectTerminal().Protocol
	}
}
