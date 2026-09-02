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
import { beforeEach, afterEach, describe, expect, test, vi } from "vitest"

vi.mock("../api/statestore", () => ({
	RoomStateStore: class RoomStateStore {},
	StateStore: class StateStore {},
}))

vi.mock("./MainScreenContext.ts", () => ({}))

import Keybindings, { keyToString } from "./keybindings.ts"

type FakeEventInit = {
	key?: string,
	shiftKey?: boolean,
	altKey?: boolean,
	ctrlKey?: boolean,
	metaKey?: boolean,
	target?: unknown,
	currentTarget?: unknown,
}

function fakeKeyEvent(init: FakeEventInit = {}) {
	return {
		key: init.key ?? "b",
		shiftKey: init.shiftKey ?? false,
		altKey: init.altKey ?? false,
		ctrlKey: init.ctrlKey ?? false,
		metaKey: init.metaKey ?? false,
		target: init.target ?? document.body,
		currentTarget: init.currentTarget ?? document.body,
		preventDefault: vi.fn(),
	} as unknown as KeyboardEvent
}

function makeRoom(roomID: string) {
	return { room_id: roomID }
}

function makeContext(rightPanel: object | null = null) {
	return {
		setActiveRoom: vi.fn(),
		clearActiveRoom: vi.fn(),
		setRightPanel: vi.fn(),
		closeRightPanel: vi.fn(),
		rightPanelValue: rightPanel,
		get currentRightPanel() {
			return this.rightPanelValue
		},
		set currentRightPanel(value: object | null) {
			this.rightPanelValue = value
		},
	}
}

function makeStore(rooms: Array<{ room_id: string }> = []) {
	return {
		getFilteredRoomList: vi.fn(() => rooms),
	}
}

function setup(rooms: Array<{ room_id: string }> = [], rightPanel: object | null = null) {
	const context = makeContext(rightPanel)
	const kb = new Keybindings(makeStore(rooms) as any, context as any)
	return { kb, context }
}

let composer: HTMLInputElement | undefined
let search: HTMLInputElement | undefined

beforeEach(() => {
	localStorage.clear()
	composer = document.createElement("input")
	composer.id = "message-composer"
	document.body.appendChild(composer)
	vi.spyOn(composer, "focus")
	search = document.createElement("input")
	search.id = "room-search"
	document.body.appendChild(search)
	vi.spyOn(search, "focus")
})

afterEach(() => {
	composer?.remove()
	search?.remove()
})

describe("keyToString", () => {
	test("plain key without modifiers", () => {
		expect(keyToString(fakeKeyEvent({ key: "a" }))).toBe("a")
	})

	test("shift modifier", () => {
		expect(keyToString(fakeKeyEvent({ key: "a", shiftKey: true }))).toBe("Shift+a")
	})

	test("alt modifier", () => {
		expect(keyToString(fakeKeyEvent({ key: "Tab", altKey: true }))).toBe("Alt+Tab")
	})

	test("meta modifier", () => {
		expect(keyToString(fakeKeyEvent({ key: "a", metaKey: true }))).toBe("Super+a")
	})

	test("ctrl modifier", () => {
		expect(keyToString(fakeKeyEvent({ key: "k", ctrlKey: true }))).toBe("Ctrl+k")
	})

	test("ctrl+shift stack", () => {
		expect(keyToString(fakeKeyEvent({ key: "a", ctrlKey: true, shiftKey: true })))
			.toBe("Ctrl+Shift+a")
	})

	test("alt+shift stack", () => {
		expect(keyToString(fakeKeyEvent({ key: "ArrowUp", altKey: true, shiftKey: true })))
			.toBe("Alt+Shift+ArrowUp")
	})

	test("all four modifiers stacked", () => {
		expect(keyToString(fakeKeyEvent({
			key: "a", ctrlKey: true, metaKey: true, altKey: true, shiftKey: true,
		}))).toBe("Ctrl+Super+Alt+Shift+a")
	})

	test("alt+ctrl stack prefixes ctrl last", () => {
		expect(keyToString(fakeKeyEvent({ key: "f", altKey: true, ctrlKey: true })))
			.toBe("Ctrl+Alt+f")
	})
})

describe("Keybindings keyDownMap", () => {
	test("Escape closes right panel when one is open", () => {
		const { kb, context } = setup([], { type: "pinned-messages" })
		kb.onKeyDown(fakeKeyEvent({ key: "Escape" }))
		expect(context.closeRightPanel).toHaveBeenCalledTimes(1)
		expect(context.clearActiveRoom).not.toHaveBeenCalled()
	})

	test("Escape clears active room when no right panel", () => {
		const { kb, context } = setup()
		kb.onKeyDown(fakeKeyEvent({ key: "Escape" }))
		expect(context.clearActiveRoom).toHaveBeenCalledTimes(1)
		expect(context.closeRightPanel).not.toHaveBeenCalled()
	})

	test("Escape calls preventDefault", () => {
		const { kb } = setup()
		const evt = fakeKeyEvent({ key: "Escape" })
		kb.onKeyDown(evt)
		expect(evt.preventDefault).toHaveBeenCalledTimes(1)
	})

	test("Ctrl+k focuses room search", () => {
		const { kb } = setup()
		kb.onKeyDown(fakeKeyEvent({ key: "k", ctrlKey: true }))
		expect(search!.focus).toHaveBeenCalledTimes(1)
		expect(composer!.focus).not.toHaveBeenCalled()
	})

	test("Ctrl+k does nothing when room search is missing", () => {
		search!.remove()
		const { kb } = setup()
		kb.onKeyDown(fakeKeyEvent({ key: "k", ctrlKey: true }))
		expect(composer!.focus).not.toHaveBeenCalled()
	})

	test("Alt+ArrowUp does nothing without an active room", () => {
		const { kb, context } = setup([makeRoom("!a"), makeRoom("!b")])
		kb.activeRoom = null
		kb.onKeyDown(fakeKeyEvent({ key: "ArrowUp", altKey: true }))
		expect(context.setActiveRoom).not.toHaveBeenCalled()
	})

	test("Alt+ArrowUp selects the next room in the list", () => {
		const { kb, context } = setup([makeRoom("!a"), makeRoom("!b"), makeRoom("!c")])
		kb.activeRoom = { roomID: "!a" } as any
		kb.onKeyDown(fakeKeyEvent({ key: "ArrowUp", altKey: true }))
		expect(context.setActiveRoom).toHaveBeenCalledWith("!b")
	})

	test("Alt+ArrowUp wraps to null at the end of the list", () => {
		const { kb, context } = setup([makeRoom("!a"), makeRoom("!b")])
		kb.activeRoom = { roomID: "!b" } as any
		kb.onKeyDown(fakeKeyEvent({ key: "ArrowUp", altKey: true }))
		expect(context.setActiveRoom).toHaveBeenCalledWith(null)
	})

	test("Alt+ArrowUp with room not in list selects the first entry", () => {
		const { kb, context } = setup([makeRoom("!a"), makeRoom("!b")])
		kb.activeRoom = { roomID: "!zzz" } as any
		kb.onKeyDown(fakeKeyEvent({ key: "ArrowUp", altKey: true }))
		expect(context.setActiveRoom).toHaveBeenCalledWith("!a")
	})

	test("Alt+ArrowDown selects the last room when nothing is active", () => {
		const { kb, context } = setup([makeRoom("!a"), makeRoom("!b"), makeRoom("!c")])
		kb.activeRoom = null
		kb.onKeyDown(fakeKeyEvent({ key: "ArrowDown", altKey: true }))
		expect(context.setActiveRoom).toHaveBeenCalledWith("!c")
	})

	test("Alt+ArrowDown selects the previous room in the list", () => {
		const { kb, context } = setup([makeRoom("!a"), makeRoom("!b"), makeRoom("!c")])
		kb.activeRoom = { roomID: "!b" } as any
		kb.onKeyDown(fakeKeyEvent({ key: "ArrowDown", altKey: true }))
		expect(context.setActiveRoom).toHaveBeenCalledWith("!a")
	})

	test("Alt+ArrowDown does nothing at the top of the list", () => {
		const { kb, context } = setup([makeRoom("!a"), makeRoom("!b")])
		kb.activeRoom = { roomID: "!a" } as any
		kb.onKeyDown(fakeKeyEvent({ key: "ArrowDown", altKey: true }))
		expect(context.setActiveRoom).not.toHaveBeenCalled()
	})

	test("Alt+ArrowDown with active room not in list selects the last entry", () => {
		const { kb, context } = setup([makeRoom("!a"), makeRoom("!b")])
		kb.activeRoom = { roomID: "!zzz" } as any
		kb.onKeyDown(fakeKeyEvent({ key: "ArrowDown", altKey: true }))
		expect(context.setActiveRoom).toHaveBeenCalledWith("!b")
	})

	test("Ctrl+f opens the search right panel", () => {
		const { kb, context } = setup()
		kb.onKeyDown(fakeKeyEvent({ key: "f", ctrlKey: true }))
		expect(context.setRightPanel).toHaveBeenCalledWith({ type: "search" })
	})
})

describe("Keybindings auto-focus composer fallback", () => {
	test("unbound plain key focuses the composer", () => {
		const { kb } = setup()
		kb.onKeyDown(fakeKeyEvent({ key: "b" }))
		expect(composer!.focus).toHaveBeenCalledTimes(1)
	})

	test("unbound key with ctrl held does not focus the composer", () => {
		const { kb } = setup()
		kb.onKeyDown(fakeKeyEvent({ key: "b", ctrlKey: true }))
		expect(composer!.focus).not.toHaveBeenCalled()
	})

	test("unbound key with meta held does not focus the composer", () => {
		const { kb } = setup()
		kb.onKeyDown(fakeKeyEvent({ key: "b", metaKey: true }))
		expect(composer!.focus).not.toHaveBeenCalled()
	})

	test("ctrl+v still focuses the composer", () => {
		const { kb } = setup()
		kb.onKeyDown(fakeKeyEvent({ key: "v", ctrlKey: true }))
		expect(composer!.focus).toHaveBeenCalledTimes(1)
	})

	test("ctrl+a still focuses the composer", () => {
		const { kb } = setup()
		kb.onKeyDown(fakeKeyEvent({ key: "a", ctrlKey: true }))
		expect(composer!.focus).toHaveBeenCalledTimes(1)
	})

	test("alt-held key does not focus the composer", () => {
		const { kb } = setup()
		kb.onKeyDown(fakeKeyEvent({ key: "b", altKey: true }))
		expect(composer!.focus).not.toHaveBeenCalled()
	})

	test("shift-held key still focuses the composer", () => {
		const { kb } = setup()
		kb.onKeyDown(fakeKeyEvent({ key: "B", shiftKey: true }))
		expect(composer!.focus).toHaveBeenCalledTimes(1)
	})

	for (const navKey of ["PageUp", "PageDown", "Home", "End"]) {
		test(`${navKey} does not focus the composer`, () => {
			const { kb } = setup()
			kb.onKeyDown(fakeKeyEvent({ key: navKey }))
			expect(composer!.focus).not.toHaveBeenCalled()
		})
	}

	test("event targeting a child element does not focus the composer", () => {
		const { kb } = setup()
		const child = document.createElement("div")
		document.body.appendChild(child)
		kb.onKeyDown(fakeKeyEvent({ key: "b", target: child, currentTarget: document.body }))
		expect(composer!.focus).not.toHaveBeenCalled()
		child.remove()
	})

	test("key handled by keyUpMap does not focus the composer", () => {
		const { kb } = setup()
		const handler = vi.fn()
		;(kb as any).keyUpMap["F9"] = handler
		kb.onKeyDown(fakeKeyEvent({ key: "F9" }))
		expect(composer!.focus).not.toHaveBeenCalled()
		delete (kb as any).keyUpMap["F9"]
	})

	test("fallback does not call preventDefault", () => {
		const { kb } = setup()
		const evt = fakeKeyEvent({ key: "b" })
		kb.onKeyDown(evt)
		expect(evt.preventDefault).not.toHaveBeenCalled()
	})

	test("composer focus is a no-op when the composer is missing", () => {
		composer!.remove()
		const { kb } = setup()
		kb.onKeyDown(fakeKeyEvent({ key: "b" }))
		expect(true).toBe(true)
	})
})

describe("Keybindings keyUpMap", () => {
	test("keyup with no registered handler is a no-op", () => {
		const { kb } = setup()
		expect(() => kb.onKeyUp(fakeKeyEvent({ key: "b" }))).not.toThrow()
	})

	test("keyup dispatches to a registered keyUpMap handler", () => {
		const { kb } = setup()
		const handler = vi.fn()
		;(kb as any).keyUpMap["Alt+x"] = handler
		const evt = fakeKeyEvent({ key: "x", altKey: true })
		kb.onKeyUp(evt)
		expect(handler).toHaveBeenCalledTimes(1)
		expect(handler).toHaveBeenCalledWith(evt)
	})
})

describe("Keybindings listen", () => {
	test("listen attaches keydown and keyup handlers and dispose detaches them", () => {
		const addSpy = vi.spyOn(document.body, "addEventListener")
		const removeSpy = vi.spyOn(document.body, "removeEventListener")
		const winAdd = vi.spyOn(window, "addEventListener")
		const winRemove = vi.spyOn(window, "removeEventListener")
		const { kb } = setup()
		const dispose = kb.listen()
		expect(addSpy).toHaveBeenCalledWith("keydown", kb.onKeyDown)
		expect(addSpy).toHaveBeenCalledWith("keyup", kb.onKeyUp)
		expect(winAdd).toHaveBeenCalledWith("gomuks-keybindings-changed", kb.reload)
		dispose()
		expect(removeSpy).toHaveBeenCalledWith("keydown", kb.onKeyDown)
		expect(removeSpy).toHaveBeenCalledWith("keyup", kb.onKeyUp)
		expect(winRemove).toHaveBeenCalledWith("gomuks-keybindings-changed", kb.reload)
		addSpy.mockRestore()
		removeSpy.mockRestore()
		winAdd.mockRestore()
		winRemove.mockRestore()
	})

	test("attached listeners dispatch real keydown events", () => {
		const { kb, context } = setup()
		const dispose = kb.listen()
		document.body.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }))
		expect(context.clearActiveRoom).toHaveBeenCalledTimes(1)
		document.body.dispatchEvent(new KeyboardEvent("keyup", { key: "Escape" }))
		dispose()
		document.body.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }))
		expect(context.clearActiveRoom).toHaveBeenCalledTimes(1)
	})

	test("listen twice returns independent disposers", () => {
		const { kb } = setup()
		const dispose1 = kb.listen()
		const dispose2 = kb.listen()
		expect(() => {
			dispose1()
			dispose2()
		}).not.toThrow()
	})

	test("reload picks up localStorage overrides", () => {
		const { kb, context } = setup()
		localStorage.setItem("gomuks-keybindings", JSON.stringify({ close_panel: "F10" }))
		kb.reload()
		kb.onKeyDown(fakeKeyEvent({ key: "Escape" }))
		expect(context.clearActiveRoom).not.toHaveBeenCalled()
		kb.onKeyDown(fakeKeyEvent({ key: "F10" }))
		expect(context.clearActiveRoom).toHaveBeenCalledTimes(1)
	})
})
