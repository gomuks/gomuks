import { beforeEach, describe, expect, test } from "vitest"

import {
	DEFAULT_KEYBINDINGS,
	KEYBINDINGS_CHANGED_EVENT,
	buildEffectiveKeymap,
	clearKeybindOverrides,
	loadKeybindOverrides,
	resolveKey,
	saveKeybindOverrides,
	validateKeybindings,
} from "./keyconfig.ts"

describe("DEFAULT_KEYBINDINGS", () => {
	test("has all expected actions", () => {
		expect(Object.keys(DEFAULT_KEYBINDINGS)).toEqual([
			"close_panel",
			"clear_room",
			"focus_room_search",
			"next_room",
			"prev_room",
			"open_search",
			"reply_prev",
			"reply_next",
		])
	})

	test("matches original hardcoded values", () => {
		expect(DEFAULT_KEYBINDINGS.close_panel).toBe("Escape")
		expect(DEFAULT_KEYBINDINGS.focus_room_search).toBe("Ctrl+k")
		expect(DEFAULT_KEYBINDINGS.next_room).toBe("Alt+ArrowUp")
		expect(DEFAULT_KEYBINDINGS.prev_room).toBe("Alt+ArrowDown")
		expect(DEFAULT_KEYBINDINGS.open_search).toBe("Ctrl+f")
		expect(DEFAULT_KEYBINDINGS.reply_prev).toBe("Ctrl+ArrowUp")
		expect(DEFAULT_KEYBINDINGS.reply_next).toBe("Ctrl+ArrowDown")
	})
})

describe("validateKeybindings", () => {
	test("empty overrides are valid", () => {
		expect(validateKeybindings({})).toEqual({ valid: true, errors: [] })
	})

	test("valid override passes", () => {
		const result = validateKeybindings({ close_panel: "Ctrl+q" })
		expect(result.valid).toBe(true)
		expect(result.errors).toEqual([])
	})

	test("unknown action is rejected", () => {
		const result = validateKeybindings({ nonexistent_action: "Ctrl+q" })
		expect(result.valid).toBe(false)
		expect(result.errors).toContain('Unknown action "nonexistent_action"')
	})

	test("empty key is rejected", () => {
		const result = validateKeybindings({ close_panel: "" })
		expect(result.valid).toBe(false)
		expect(result.errors.some(e => e.includes("empty"))).toBe(true)
	})

	test("whitespace-only key is rejected", () => {
		const result = validateKeybindings({ close_panel: "   " })
		expect(result.valid).toBe(false)
	})

	test("reserved Enter key is rejected", () => {
		const result = validateKeybindings({ close_panel: "Enter" })
		expect(result.valid).toBe(false)
		expect(result.errors.some(e => e.includes("reserved"))).toBe(true)
	})

	test("reserved Tab key is rejected", () => {
		const result = validateKeybindings({ close_panel: "Tab" })
		expect(result.valid).toBe(false)
		expect(result.errors.some(e => e.includes("reserved"))).toBe(true)
	})

	test("duplicate keys across actions are rejected", () => {
		const result = validateKeybindings({
			close_panel: "Ctrl+q",
			open_search: "Ctrl+q",
		})
		expect(result.valid).toBe(false)
		expect(result.errors.some(e => e.includes("Duplicate"))).toBe(true)
	})

	test("override colliding with another action default is rejected", () => {
		const result = validateKeybindings({ next_room: "Alt+ArrowDown" })
		expect(result.valid).toBe(false)
		expect(result.errors.some(e => e.includes("Duplicate"))).toBe(true)
	})

	test("non-string value is rejected", () => {
		const result = validateKeybindings({ close_panel: 42 as unknown as string })
		expect(result.valid).toBe(false)
	})

	test("null value is rejected", () => {
		const result = validateKeybindings({ close_panel: null as unknown as string })
		expect(result.valid).toBe(false)
	})
})

describe("localStorage round-trip", () => {
	beforeEach(() => {
		localStorage.clear()
	})

	test("loadKeybindOverrides returns empty when nothing stored", () => {
		expect(loadKeybindOverrides()).toEqual({})
	})

	test("saveKeybindOverrides + loadKeybindOverrides round-trips", () => {
		saveKeybindOverrides({ close_panel: "Ctrl+q" })
		expect(loadKeybindOverrides()).toEqual({ close_panel: "Ctrl+q" })
	})

	test("saveKeybindOverrides notifies listeners", () => {
		let fired = 0
		const onChange = () => { fired += 1 }
		window.addEventListener(KEYBINDINGS_CHANGED_EVENT, onChange)
		saveKeybindOverrides({ close_panel: "Ctrl+q" })
		window.removeEventListener(KEYBINDINGS_CHANGED_EVENT, onChange)
		expect(fired).toBe(1)
	})

	test("saveKeybindOverrides throws on invalid overrides", () => {
		expect(() => saveKeybindOverrides({ bogus: "Ctrl+q" })).toThrow()
	})

	test("saveKeybindOverrides does NOT write when invalid", () => {
		try {
			saveKeybindOverrides({ bogus: "Ctrl+q" })
		} catch {
			// expected
		}
		expect(localStorage.getItem("gomuks-keybindings")).toBeNull()
	})

	test("clearKeybindOverrides removes stored data", () => {
		saveKeybindOverrides({ close_panel: "Ctrl+q" })
		clearKeybindOverrides()
		expect(loadKeybindOverrides()).toEqual({})
	})

	test("loadKeybindOverrides handles corrupt JSON gracefully", () => {
		localStorage.setItem("gomuks-keybindings", "not-json{{{")
		expect(loadKeybindOverrides()).toEqual({})
	})

	test("loadKeybindOverrides handles non-object JSON gracefully", () => {
		localStorage.setItem("gomuks-keybindings", JSON.stringify([1, 2, 3]))
		expect(loadKeybindOverrides()).toEqual({})
	})

	test("loadKeybindOverrides handles null JSON gracefully", () => {
		localStorage.setItem("gomuks-keybindings", "null")
		expect(loadKeybindOverrides()).toEqual({})
	})
})

describe("resolveKey", () => {
	test("returns default when no override", () => {
		expect(resolveKey("close_panel", {})).toBe("Escape")
	})

	test("returns override when present", () => {
		expect(resolveKey("close_panel", { close_panel: "Ctrl+q" })).toBe("Ctrl+q")
	})

	test("returns default when override is empty", () => {
		expect(resolveKey("close_panel", { close_panel: "" })).toBe("Escape")
	})

	test("returns default when override is whitespace", () => {
		expect(resolveKey("close_panel", { close_panel: "  " })).toBe("Escape")
	})
})

describe("buildEffectiveKeymap", () => {
	test("returns defaults when no overrides", () => {
		expect(buildEffectiveKeymap({})).toEqual(DEFAULT_KEYBINDINGS)
	})

	test("overrides replace defaults", () => {
		const result = buildEffectiveKeymap({ close_panel: "Ctrl+q" })
		expect(result.close_panel).toBe("Ctrl+q")
		expect(result.focus_room_search).toBe("Ctrl+k")
	})

	test("ignores unknown actions", () => {
		const result = buildEffectiveKeymap({ unknown_action: "Ctrl+q" })
		expect(result).toEqual(DEFAULT_KEYBINDINGS)
	})

	test("does not mutate defaults", () => {
		const overrides = { close_panel: "Ctrl+q" }
		buildEffectiveKeymap(overrides)
		expect(DEFAULT_KEYBINDINGS.close_panel).toBe("Escape")
	})
})
