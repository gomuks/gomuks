// gomuks - A Matrix client written in Go.
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
import { act } from "react"
import { createElement } from "react"
import { createRoot, type Root } from "react-dom/client"
import { afterEach, beforeEach, describe, expect, test } from "vitest"
import KeybindingsSettings from "./KeybindingsSettings.tsx"

beforeEach(() => {
	(globalThis as unknown as { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT = true
	localStorage.clear()
})

describe("KeybindingsSettings", () => {
	let container: HTMLDivElement
	let root: Root

	beforeEach(() => {
		container = document.createElement("div")
		document.body.appendChild(container)
		root = createRoot(container)
	})

	afterEach(() => {
		act(() => root.unmount())
		container.remove()
	})

	test("renders default shortcuts", () => {
		act(() => root.render(createElement(KeybindingsSettings)))
		expect(container.textContent).toContain("Focus room search")
		expect(container.textContent).toContain("Ctrl+k")
		expect(container.textContent).toContain("Reply to previous message")
		expect(container.textContent).toContain("Ctrl+ArrowUp")
	})

	test("records a new shortcut and persists it", () => {
		act(() => root.render(createElement(KeybindingsSettings)))
		const buttons = [...container.querySelectorAll("button")]
		const searchBtn = buttons.find(btn => btn.textContent === "Ctrl+k")
		expect(searchBtn).toBeTruthy()
		act(() => searchBtn!.click())
		expect(searchBtn!.textContent).toBe("Press a key…")
		act(() => {
			window.dispatchEvent(new KeyboardEvent("keydown", {
				key: "s",
				ctrlKey: true,
				bubbles: true,
				cancelable: true,
			}))
		})
		expect(JSON.parse(localStorage.getItem("gomuks-keybindings")!)).toEqual({
			focus_room_search: "Ctrl+s",
		})
		expect(searchBtn!.textContent).toBe("Ctrl+s")
	})

	test("rejects reserved Enter and shows error", () => {
		act(() => root.render(createElement(KeybindingsSettings)))
		const buttons = [...container.querySelectorAll("button")]
		const searchBtn = buttons.find(btn => btn.textContent === "Ctrl+k")
		act(() => searchBtn!.click())
		act(() => {
			window.dispatchEvent(new KeyboardEvent("keydown", {
				key: "Enter",
				bubbles: true,
				cancelable: true,
			}))
		})
		expect(container.querySelector(".keybind-error")?.textContent).toMatch(/reserved/)
		expect(localStorage.getItem("gomuks-keybindings")).toBeNull()
	})

	test("reset to defaults clears storage", () => {
		localStorage.setItem("gomuks-keybindings", JSON.stringify({ focus_room_search: "Ctrl+s" }))
		act(() => root.render(createElement(KeybindingsSettings)))
		const reset = [...container.querySelectorAll("button")].find(btn => btn.textContent === "Reset to defaults")
		expect(reset).toBeTruthy()
		act(() => reset!.click())
		expect(localStorage.getItem("gomuks-keybindings")).toBeNull()
		expect(container.textContent).toContain("Ctrl+k")
	})
})
