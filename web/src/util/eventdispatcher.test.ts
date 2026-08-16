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
import { CachedEventDispatcher, EventDispatcher, NonNullCachedEventDispatcher, useEventAsState } from "./eventdispatcher"

beforeEach(() => {
	(globalThis as unknown as { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT = true
})

describe("EventDispatcher", () => {
	test("listen registers a listener that emit calls with data", () => {
		const dispatcher = new EventDispatcher<number>()
		const listener = vi.fn()
		dispatcher.listen(listener)
		dispatcher.emit(42)
		expect(listener).toHaveBeenCalledTimes(1)
		expect(listener).toHaveBeenCalledWith(42)
	})

	test("emit calls listeners in registration order", () => {
		const dispatcher = new EventDispatcher<number>()
		const calls: string[] = []
		dispatcher.listen(() => calls.push("first"))
		dispatcher.listen(() => calls.push("second"))
		dispatcher.emit(1)
		expect(calls).toEqual(["first", "second"])
	})

	test("emit with no listeners does nothing", () => {
		const dispatcher = new EventDispatcher<number>()
		expect(() => dispatcher.emit(1)).not.toThrow()
	})

	test("unlisten stops the listener from being called", () => {
		const dispatcher = new EventDispatcher<number>()
		const listener = vi.fn()
		const unlisten = dispatcher.listen(listener)
		unlisten()
		dispatcher.emit(1)
		expect(listener).not.toHaveBeenCalled()
	})

	test("double unlisten is safe", () => {
		const dispatcher = new EventDispatcher<number>()
		const other = vi.fn()
		dispatcher.listen(other)
		const unlisten = dispatcher.listen(vi.fn())
		unlisten()
		unlisten()
		dispatcher.emit(1)
		expect(other).toHaveBeenCalledTimes(1)
	})

	test("unsubscribing during emit prevents later calls to the same listener", () => {
		const dispatcher = new EventDispatcher<number>()
		let callCount = 0
		const unlisten = dispatcher.listen(() => {
			callCount++
			unlisten()
		})
		dispatcher.emit(1)
		dispatcher.emit(2)
		expect(callCount).toBe(1)
	})

	test("a listener unsubscribing another listener during emit", () => {
		const dispatcher = new EventDispatcher<number>()
		const second = vi.fn()
		let unlistenSecond: () => void = () => {}
		dispatcher.listen(() => unlistenSecond())
		unlistenSecond = dispatcher.listen(second)
		dispatcher.emit(1)
		expect(second).not.toHaveBeenCalled()
	})

	test("a throwing listener stops emit and propagates the error", () => {
		const dispatcher = new EventDispatcher<number>()
		const boom = () => {
			throw new Error("boom")
		}
		const after = vi.fn()
		dispatcher.listen(boom)
		dispatcher.listen(after)
		expect(() => dispatcher.emit(1)).toThrow("boom")
		expect(after).not.toHaveBeenCalled()
	})

	test("hasListeners reflects listener state", () => {
		const dispatcher = new EventDispatcher<number>()
		expect(dispatcher.hasListeners).toBe(false)
		const unlisten = dispatcher.listen(vi.fn())
		expect(dispatcher.hasListeners).toBe(true)
		unlisten()
		expect(dispatcher.hasListeners).toBe(false)
	})

	test("listenChange registers a change listener", () => {
		const dispatcher = new EventDispatcher<number>()
		const listener = vi.fn()
		const unlisten = dispatcher.listenChange(listener)
		expect(dispatcher.hasListeners).toBe(true)
		dispatcher.emit(7)
		expect(listener).toHaveBeenCalledTimes(1)
		unlisten()
		expect(dispatcher.hasListeners).toBe(false)
	})

	test("once listener fires exactly one time", () => {
		const dispatcher = new EventDispatcher<number>()
		const listener = vi.fn()
		dispatcher.once(listener)
		dispatcher.emit(1)
		dispatcher.emit(2)
		expect(listener).toHaveBeenCalledTimes(1)
		expect(listener).toHaveBeenCalledWith(1)
		expect(dispatcher.hasListeners).toBe(false)
	})

	test("once listener can be manually unsubscribed before firing", () => {
		const dispatcher = new EventDispatcher<number>()
		const listener = vi.fn()
		const unlisten = dispatcher.once(listener)
		unlisten()
		dispatcher.emit(1)
		expect(listener).not.toHaveBeenCalled()
		expect(dispatcher.hasListeners).toBe(false)
	})
})

describe("CachedEventDispatcher", () => {
	test("cache defaults to null and can be initialized", () => {
		expect(new CachedEventDispatcher<number>().current).toBeNull()
		expect(new CachedEventDispatcher<number>(5).current).toBe(5)
	})

	test("emit updates the cache and notifies listeners", () => {
		const dispatcher = new CachedEventDispatcher<number>()
		const listener = vi.fn()
		dispatcher.listen(listener)
		dispatcher.emit(3)
		expect(dispatcher.current).toBe(3)
		expect(listener).toHaveBeenCalledTimes(1)
	})

	test("emit with an identical cached value is skipped", () => {
		const dispatcher = new CachedEventDispatcher<number>(3)
		const listener = vi.fn()
		dispatcher.listen(listener) // replays cache: 1 call
		dispatcher.emit(3)
		expect(listener).toHaveBeenCalledTimes(1)
		expect(dispatcher.current).toBe(3)
	})

	test("emit distinguishes zero and negative zero via Object.is", () => {
		const dispatcher = new CachedEventDispatcher<number>(0)
		const listener = vi.fn()
		dispatcher.listen(listener)
		dispatcher.emit(-0)
		expect(listener).toHaveBeenCalledTimes(2) // replay + emit
	})

	test("listen replays the cached value immediately", () => {
		const dispatcher = new CachedEventDispatcher<string>()
		dispatcher.emit("cached")
		const listener = vi.fn()
		dispatcher.listen(listener)
		expect(listener).toHaveBeenCalledTimes(1)
		expect(listener).toHaveBeenCalledWith("cached")
	})

	test("listen with a null cache does not call the listener", () => {
		const dispatcher = new CachedEventDispatcher<string>()
		const listener = vi.fn()
		dispatcher.listen(listener)
		expect(listener).not.toHaveBeenCalled()
	})

	test("clearCache resets the cache so the same value emits again", () => {
		const dispatcher = new CachedEventDispatcher<number>()
		dispatcher.emit(1)
		const listener1 = vi.fn()
		dispatcher.listen(listener1) // replay
		dispatcher.clearCache()
		expect(dispatcher.current).toBeNull()
		const listener2 = vi.fn()
		dispatcher.listen(listener2) // no replay after clear
		dispatcher.emit(1)
		expect(listener1).toHaveBeenCalledTimes(2)
		expect(listener2).toHaveBeenCalledTimes(1)
	})
})

describe("NonNullCachedEventDispatcher", () => {
	test("cache is set from the constructor and never null", () => {
		const dispatcher = new NonNullCachedEventDispatcher<boolean>(true)
		expect(dispatcher.current).toBe(true)
	})

	test("clearCache throws", () => {
		const dispatcher = new NonNullCachedEventDispatcher<boolean>(false)
		expect(() => dispatcher.clearCache()).toThrow("Cannot clear cache of NonNullCachedEventDispatcher")
	})
})

describe("useEventAsState", () => {
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

	test("returns the dispatcher's current value and updates on emit", () => {
		const dispatcher = new NonNullCachedEventDispatcher<number>(10)
		let rendered: number | null | undefined
		function Component() {
			rendered = useEventAsState(dispatcher)
			return null
		}
		act(() => root.render(createElement(Component)))
		expect(rendered).toBe(10)
		act(() => dispatcher.emit(20))
		expect(rendered).toBe(20)
	})

	test("unmount unsubscribes from the dispatcher", () => {
		const dispatcher = new NonNullCachedEventDispatcher<number>(1)
		function Component() {
			useEventAsState(dispatcher)
			return null
		}
		act(() => root.render(createElement(Component)))
		expect(dispatcher.hasListeners).toBe(true)
		act(() => root.unmount())
		expect(dispatcher.hasListeners).toBe(false)
	})

	test("undefined dispatcher renders as null", () => {
		let rendered: number | null | undefined = "unset"
		function Component() {
			rendered = useEventAsState<number>()
			return null
		}
		act(() => root.render(createElement(Component)))
		expect(rendered).toBeNull()
	})

	test("nullable dispatcher starts as null and updates on emit", () => {
		const dispatcher = new CachedEventDispatcher<number>()
		let rendered: number | null | undefined = "unset"
		function Component() {
			rendered = useEventAsState(dispatcher)
			return null
		}
		act(() => root.render(createElement(Component)))
		expect(rendered).toBeNull()
		act(() => dispatcher.emit(5))
		expect(rendered).toBe(5)
	})
})
