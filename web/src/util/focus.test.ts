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
import { createElement } from "react"
import { act } from "react"
import { createRoot, type Root } from "react-dom/client"
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest"
import useFocus, { focused } from "./focus"

beforeEach(() => {
	(globalThis as unknown as { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT = true
})

describe("focus", () => {
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

	test("focused is initialized from document.hasFocus", () => {
		expect(focused.current).toBe(document.hasFocus())
	})

	test("window focus and blur events update the focused dispatcher", () => {
		focused.clearCache = () => {
			throw new Error("unexpected")
		}
		const initialState = focused.current
		act(() => {
			window.dispatchEvent(new Event(initialState ? "blur" : "focus"))
		})
		expect(focused.current).toBe(!initialState)
		act(() => {
			window.dispatchEvent(new Event(initialState ? "focus" : "blur"))
		})
		expect(focused.current).toBe(initialState)
	})

	test("useFocus reflects window focus state changes", () => {
		let rendered: boolean | null = null
		function Component() {
			rendered = useFocus()
			return null
		}
		act(() => root.render(createElement(Component)))
		expect(rendered).toBe(focused.current)
		const initialState = focused.current
		act(() => {
			window.dispatchEvent(new Event(initialState ? "blur" : "focus"))
		})
		expect(rendered).toBe(!initialState)
	})
})

// NonNullCachedEventDispatcher.clearCache throwing is covered in
// eventdispatcher.test.ts; here we just guard that we accidentally
// overwrote it in the test above.
afterEach(() => {
	vi.restoreAllMocks()
})
