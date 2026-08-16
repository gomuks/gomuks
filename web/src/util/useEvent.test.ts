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
import useEvent from "./useEvent"

beforeEach(() => {
	(globalThis as unknown as { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT = true
})

describe("useEvent", () => {
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

	test("the returned function calls the latest handler with all arguments", () => {
		const handler1 = vi.fn()
		const handler2 = vi.fn()
		let stable: ((...args: Array<unknown>) => void) | undefined
		function Component({ handler }: { handler: (...args: Array<unknown>) => void }) {
			stable = useEvent(handler)
			return null
		}
		act(() => root.render(createElement(Component, { handler: handler1 })))
		stable!("a", 1)
		expect(handler1).toHaveBeenCalledWith("a", 1)
		act(() => root.render(createElement(Component, { handler: handler2 })))
		stable!("b", 2)
		expect(handler1).toHaveBeenCalledTimes(1)
		expect(handler2).toHaveBeenCalledWith("b", 2)
	})

	test("the returned function identity is stable across re-renders", () => {
		let first: ((...args: Array<unknown>) => void) | undefined
		let current: ((...args: Array<unknown>) => void) | undefined
		function Component() {
			current = useEvent(() => {})
			first ??= current
			return null
		}
		act(() => root.render(createElement(Component)))
		act(() => root.render(createElement(Component)))
		expect(current).toBe(first)
	})

	test("handler is never called directly by rendering", () => {
		const handler = vi.fn()
		function Component() {
			useEvent(handler)
			return null
		}
		act(() => root.render(createElement(Component)))
		act(() => root.render(createElement(Component)))
		expect(handler).not.toHaveBeenCalled()
	})

	test("the callback works with zero arguments", () => {
		const handler = vi.fn()
		let stable: (() => void) | undefined
		function Component() {
			stable = useEvent(handler)
			return null
		}
		act(() => root.render(createElement(Component)))
		stable!()
		expect(handler).toHaveBeenCalledTimes(1)
	})
})
