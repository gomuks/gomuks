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
import { createElement, type CSSProperties, type ReactNode } from "react"
import { act } from "react"
import { createRoot, type Root } from "react-dom/client"
import { afterEach, beforeEach, describe, expect, test } from "vitest"
import useContentVisibility from "./contentvisibility"

function fireContentVisibility(
	element: Element,
	skipped: boolean,
	fromChild = false,
) {
	const target = fromChild ? element.firstElementChild! : element
	const evt = new CustomEvent("contentvisibilityautostatechange", {
		bubbles: true,
	})
	;(evt as unknown as { skipped: boolean }).skipped = skipped
	target.dispatchEvent(evt)
}

describe("useContentVisibility", () => {
	let container: HTMLDivElement
	let root: Root

	beforeEach(() => {
		(globalThis as unknown as { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT = true
		container = document.createElement("div")
		document.body.appendChild(container)
		root = createRoot(container)
	})

	afterEach(() => {
		act(() => root.unmount())
		container.remove()
	})

	function renderHook(allowRevert = false): HTMLDivElement {
		let host: HTMLDivElement | null = null
		function Component() {
			const [isVisible, ref] = useContentVisibility<HTMLDivElement>(allowRevert)
			const style: CSSProperties = { display: isVisible ? "block" : "none" }
			return createElement(
				"div",
				{ ref: (el: HTMLDivElement | null) => { ref.current = el; host = el }, style },
				createElement("span"),
			)
		}
		act(() => root.render(createElement(Component)))
		return host!
	}

	test("starts invisible and becomes visible when content is rendered", () => {
		const host = renderHook()
		expect(host.style.display).toBe("none")
		act(() => fireContentVisibility(host, false))
		expect(host.style.display).toBe("block")
	})

	test("events from a child target are ignored (chromium bug workaround)", () => {
		const host = renderHook()
		act(() => fireContentVisibility(host, false, true))
		expect(host.style.display).toBe("none")
	})

	test("skipped events revert visibility only when allowRevert is true", () => {
		const host = renderHook(true)
		act(() => fireContentVisibility(host, false))
		expect(host.style.display).toBe("block")
		act(() => fireContentVisibility(host, true))
		expect(host.style.display).toBe("none")
	})

	test("skipped events keep visibility when allowRevert is false", () => {
		const host = renderHook(false)
		act(() => fireContentVisibility(host, false))
		expect(host.style.display).toBe("block")
		act(() => fireContentVisibility(host, true))
		expect(host.style.display).toBe("block")
	})

	test("without an attached element the hook stays invisible and does not throw", () => {
		let renderedVisible: boolean | ReactNode = "unset"
		function NoElementComponent() {
			const [isVisible] = useContentVisibility<HTMLDivElement>()
			renderedVisible = isVisible
			return null
		}
		act(() => root.render(createElement(NoElementComponent)))
		expect(renderedVisible).toBe(false)
	})

	test("unmount removes the event listener without errors", () => {
		const host = renderHook()
		act(() => fireContentVisibility(host, false))
		act(() => root.unmount())
		expect(() => fireContentVisibility(host, false)).not.toThrow()
	})
})
