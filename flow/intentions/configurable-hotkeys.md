# Intention: Make gomuks hotkeys configurable

**Date**: 2026-08-16
**Status**: planned (implementation NOT started)
**Fork purpose**: This fork exists SOLELY for this feature (until upstreamed)

## Why

Upstream gomuks web client ships 5 hardcoded keybindings in
`web/src/ui/keybindings.ts` (`keyDownMap`). No user configurability:

- 5 bindings too thin vs Element Web (~20+) or iamb (vim-modal, dozens)
- Users want: jump-to-room-N, mark-read, reply, edit-last, toggle panels, etc.
- Live deployment (chat-stack) confirmed the gap after hotkey extraction
  from served bundle (`index-CHh6NJER.js`)

## Requirement (DOD)

1. Keybindings loadable from user config — file-based (daemon serves config
   to web client) or localStorage (pure client-side)
2. Defaults = current hardcoded map (backward-compat, zero regression)
3. Settings UI: view/edit bindings in web client (right panel or modal)
4. Validation: duplicate detection, reserved keys guarded
   (Enter/Tab/printable-in-composer must never be bindable)
5. `keyToString` unchanged (Shift+Alt+Super+Ctrl prefix chain preserved)
6. Tests: unit test keymap load/merge/validate; e2e manual checklist
7. Rebase-friendly: minimal diff, no unrelated refactors

## Approach options (decide at impl time)

| Option | Storage | Pros | Cons |
|--------|---------|------|------|
| A: server config file (`~/.config/gomuks/keybindings.yaml`, daemon injects into web) | backend | Consistent across devices/browsers | Needs daemon API change |
| B: localStorage (pure client) | frontend | Zero backend change | Per-browser only |
| C: both (file = default, localStorage overrides) | hybrid | Best UX | Most code |

Recommendation: **C** — file = shareable baseline, localStorage = per-device
tweaks. If backend change too invasive → start B, file support later.

## Key files

| File | Role |
|------|------|
| `web/src/ui/keybindings.ts` | THE hotkey code (keyDownMap, keyUpMap, keyToString, listen/onKeyDown/onKeyUp) |
| `web/src/ui/MainScreenContext.ts` | context API (setActiveRoom, setRightPanel, closeRightPanel, clearActiveRoom) — actions bindings call |
| `web/src/api/statestore` | StateStore, RoomStateStore (getFilteredRoomList) — data for room-nav bindings |
| `cmd/gomuks/` | daemon entry (if Option A/C: config endpoint) |
| `~/.config/gomuks/config.yaml` | existing daemon+web config (username, password_hash, insecure_cookies, listen_address) — keybindings.yaml would sit beside it |

## Current default map (must preserve as fallback)

| Key | Action (context method) |
|-----|------------------------|
| Escape | closeRightPanel() else clearActiveRoom() |
| Ctrl+k | focus #room-search |
| Alt+ArrowUp | setActiveRoom(next in getFilteredRoomList()) |
| Alt+ArrowDown | setActiveRoom(prev in getFilteredRoomList()) |
| Ctrl+f | setRightPanel({type:"search"}) |

## Auto-focus rule (must not break)

`onKeyDown`: unhandled key + target=body + not modifier-only + not PageUp/Down/Home/End
→ focuses `#message-composer`. Configurable bindings must keep this escape
hatch intact (i.e., custom binding registered = handled = no auto-focus).

## Upstreaming

Target: PR to gomuks/gomuks after feature stable. AGPL-3.0 — fork obligations
met (source open, license preserved).

## Non-goals

- No vim-modal editing (iamb exists for that)
- No keybinding sync via Matrix account data (YAGNI for v1)
- No chord sequences (e.g., "g g") for v1
