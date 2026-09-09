// gomuks - A Matrix client written in Go.
// Copyright (C) 2024 Tulir Asokan
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

/**
 * Configurable keybinding system for the web client.
 *
 * Storage: localStorage key "gomuks-keybindings" holds a JSON object
 * mapping action names to key strings (e.g. {"close_panel": "Escape"}).
 *
 * Defaults match the original hardcoded map in keybindings.ts plus the
 * composer reply-select keys.
 *
 * Validation rules:
 * - Key must be non-empty string
 * - Action must be a known action name
 * - No duplicate keys across independently bindable actions
 * - Reserved keys (Enter, Tab) cannot be bound
 */

export type ActionName =
	| "close_panel"
	| "clear_room"
	| "focus_room_search"
	| "next_room"
	| "prev_room"
	| "open_search"
	| "reply_prev"
	| "reply_next"

/** Actions shown in the settings panel (independently bindable). */
export const BINDABLE_ACTIONS: ActionName[] = [
	"close_panel",
	"focus_room_search",
	"next_room",
	"prev_room",
	"open_search",
	"reply_prev",
	"reply_next",
]

export const ACTION_LABELS: Record<ActionName, string> = {
	close_panel: "Close right panel / leave room",
	clear_room: "Leave room (same key as close panel)",
	focus_room_search: "Focus room search",
	next_room: "Next room",
	prev_room: "Previous room",
	open_search: "Open search panel",
	reply_prev: "Reply to previous message",
	reply_next: "Reply to next message",
}

/** Default keybindings matching the original hardcoded map */
export const DEFAULT_KEYBINDINGS: Record<ActionName, string> = {
	close_panel: "Escape",
	clear_room: "Escape",
	focus_room_search: "Ctrl+k",
	next_room: "Alt+ArrowUp",
	prev_room: "Alt+ArrowDown",
	open_search: "Ctrl+f",
	reply_prev: "Ctrl+ArrowUp",
	reply_next: "Ctrl+ArrowDown",
}

export const STORAGE_KEY = "gomuks-keybindings"
export const KEYBINDINGS_CHANGED_EVENT = "gomuks-keybindings-changed"

/** Keys that are always reserved for composer input */
const RESERVED_KEYS = new Set(["Enter", "Tab"])

export interface KeybindValidationResult {
	valid: boolean
	errors: string[]
}

function notifyKeybindingsChanged(): void {
	if (typeof window === "undefined") {
		return
	}
	window.dispatchEvent(new Event(KEYBINDINGS_CHANGED_EVENT))
}

/**
 * Validate a keybinding override map.
 * Returns errors for: empty keys, unknown actions, duplicate keys, reserved keys.
 * Duplicate detection uses the merged (defaults + overrides) bindable map so
 * an override cannot silently collide with another action's default.
 */
export function validateKeybindings(overrides: Record<string, string>): KeybindValidationResult {
	const errors: string[] = []
	const knownActions = new Set<string>(Object.keys(DEFAULT_KEYBINDINGS))

	for (const [action, key] of Object.entries(overrides)) {
		if (typeof key !== "string" || key.trim() === "") {
			errors.push(`Action "${action}" has empty or non-string key`)
			continue
		}
		if (!knownActions.has(action)) {
			errors.push(`Unknown action "${action}"`)
			continue
		}
		if (RESERVED_KEYS.has(key)) {
			errors.push(`Key "${key}" is reserved and cannot be bound`)
		}
	}

	const merged = buildEffectiveKeymap(overrides)
	const seenKeys = new Map<string, string>()
	for (const action of BINDABLE_ACTIONS) {
		const key = merged[action]
		const existingAction = seenKeys.get(key)
		if (existingAction) {
			errors.push(`Duplicate key "${key}" bound to both "${existingAction}" and "${action}"`)
		} else {
			seenKeys.set(key, action)
		}
	}

	return { valid: errors.length === 0, errors }
}

/**
 * Load user keybinding overrides from localStorage.
 * Returns empty object if nothing stored or parse fails.
 */
export function loadKeybindOverrides(): Record<string, string> {
	try {
		const raw = localStorage.getItem(STORAGE_KEY)
		if (!raw) return {}
		const parsed = JSON.parse(raw)
		if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) return {}
		return parsed as Record<string, string>
	} catch {
		return {}
	}
}

/**
 * Save keybinding overrides to localStorage.
 * Validates before saving; throws if invalid.
 */
export function saveKeybindOverrides(overrides: Record<string, string>): void {
	const result = validateKeybindings(overrides)
	if (!result.valid) {
		throw new Error(`Invalid keybindings: ${result.errors.join("; ")}`)
	}
	localStorage.setItem(STORAGE_KEY, JSON.stringify(overrides))
	notifyKeybindingsChanged()
}

/**
 * Clear all user overrides, reverting to defaults.
 */
export function clearKeybindOverrides(): void {
	localStorage.removeItem(STORAGE_KEY)
	notifyKeybindingsChanged()
}

/**
 * Resolve the effective keybinding for an action.
 * User override wins if present and valid; otherwise default.
 */
export function resolveKey(action: ActionName, overrides: Record<string, string>): string {
	const userKey = overrides[action]
	if (typeof userKey === "string" && userKey.trim() !== "") {
		return userKey
	}
	return DEFAULT_KEYBINDINGS[action]
}

/**
 * Build the full effective keymap: action → key, with overrides merged in.
 */
export function buildEffectiveKeymap(overrides: Record<string, string>): Record<ActionName, string> {
	const result = { ...DEFAULT_KEYBINDINGS }
	for (const [action, key] of Object.entries(overrides)) {
		if (action in result && typeof key === "string" && key.trim() !== "") {
			(result as Record<string, string>)[action] = key
		}
	}
	return result
}
