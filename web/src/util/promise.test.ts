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
import { describe, expect, test, vi } from "vitest"
import { CancellablePromise } from "./promise"

describe("CancellablePromise", () => {
	test("is a Promise instance and awaits resolved value", async () => {
		const p = new CancellablePromise<number>(resolve => resolve(42), () => {})
		expect(p).toBeInstanceOf(Promise)
		await expect(p).resolves.toBe(42)
	})

	test("rejects through the executor", async () => {
		const p = new CancellablePromise<number>(
			(_resolve, reject) => reject(new Error("nope")),
			() => {},
		)
		await expect(p).rejects.toThrow("nope")
	})

	test("stores the cancel callback and invokes it on demand", async () => {
		const cancel = vi.fn()
		const p = new CancellablePromise<number>(resolve => resolve(1), cancel)
		await p
		expect(p.cancel).toBe(cancel)
		p.cancel("user navigated away")
		expect(cancel).toHaveBeenCalledExactlyOnceWith("user navigated away")
	})

	test("works with thenable chaining", async () => {
		const p = new CancellablePromise<string>(resolve => resolve("hi"), () => {})
		expect(await p.then(s => s + "!")).toBe("hi!")
	})
})
