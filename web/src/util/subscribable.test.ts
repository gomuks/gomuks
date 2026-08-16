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
// GNU General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
import { describe, expect, test, vi } from "vitest"
import Subscribable, { MultiSubscribable, NoDataSubscribable } from "./subscribable"

describe("Subscribable", () => {
	test("subscribe registers a callback that notify calls", () => {
		const sub = new Subscribable()
		const cb = vi.fn()
		sub.subscribe(cb)
		sub.notify()
		expect(cb).toHaveBeenCalledTimes(1)
	})

	test("notify calls all subscribers", () => {
		const sub = new Subscribable()
		const cb1 = vi.fn()
		const cb2 = vi.fn()
		sub.subscribe(cb1)
		sub.subscribe(cb2)
		sub.notify()
		expect(cb1).toHaveBeenCalledTimes(1)
		expect(cb2).toHaveBeenCalledTimes(1)
	})

	test("unsubscribe removes the callback", () => {
		const sub = new Subscribable()
		const cb = vi.fn()
		const unsubscribe = sub.subscribe(cb)
		unsubscribe()
		sub.notify()
		expect(cb).not.toHaveBeenCalled()
		expect(sub.subscribers.size).toBe(0)
	})

	test("same callback subscribed twice is only called once", () => {
		const sub = new Subscribable()
		const cb = vi.fn()
		sub.subscribe(cb)
		sub.subscribe(cb)
		sub.notify()
		expect(cb).toHaveBeenCalledTimes(1)
	})

	test("notify with no subscribers is a no-op", () => {
		expect(() => new Subscribable().notify()).not.toThrow()
	})
})

describe("NoDataSubscribable", () => {
	test("getData counts notifications", () => {
		const sub = new NoDataSubscribable()
		expect(sub.getData()).toBe(0)
		sub.notify()
		sub.notify()
		expect(sub.getData()).toBe(2)
	})

	test("notifications reach subscribers", () => {
		const sub = new NoDataSubscribable()
		const cb = vi.fn()
		sub.subscribe(cb)
		sub.notify()
		expect(cb).toHaveBeenCalledTimes(1)
		expect(sub.getData()).toBe(1)
	})
})

describe("MultiSubscribable", () => {
	test("subscribers only receive notifications for their key", () => {
		const multi = new MultiSubscribable()
		const cbA = vi.fn()
		const cbB = vi.fn()
		multi.getSubscriber("a")(cbA)
		multi.getSubscriber("b")(cbB)
		multi.notify("a")
		expect(cbA).toHaveBeenCalledTimes(1)
		expect(cbB).not.toHaveBeenCalled()
	})

	test("getSubscriber caches the subscribe function per key", () => {
		const multi = new MultiSubscribable()
		const first = multi.getSubscriber("a")
		const second = multi.getSubscriber("a")
		expect(first).toBe(second)
		expect(multi.getSubscriber("b")).not.toBe(first)
	})

	test("unsubscribed callbacks stop receiving notifications", () => {
		const multi = new MultiSubscribable()
		const cb = vi.fn()
		const unsubscribe = multi.getSubscriber("a")(cb)
		multi.notify("a")
		unsubscribe()
		multi.notify("a")
		expect(cb).toHaveBeenCalledTimes(1)
	})

	test("notify with unknown key is a no-op", () => {
		const multi = new MultiSubscribable()
		expect(() => multi.notify("missing")).not.toThrow()
	})

	test("notify with no subscribers for a known key is a no-op", () => {
		const multi = new MultiSubscribable()
		multi.getSubscriber("a")
		expect(() => multi.notify("a")).not.toThrow()
	})
})
