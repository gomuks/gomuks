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
import { describe, expect, test } from "vitest"
import { listDiff, objectDiff } from "./diff"

describe("listDiff", () => {
	test("returns added and removed items for small arrays", () => {
		const [added, removed] = listDiff([1, 2, 3, 5], [2, 3, 4])
		expect(added).toEqual([1, 5])
		expect(removed).toEqual([4])
	})

	test("returns empty arrays for identical small arrays", () => {
		const [added, removed] = listDiff(["a", "b"], ["a", "b"])
		expect(added).toEqual([])
		expect(removed).toEqual([])
	})

	test("handles empty arrays", () => {
		expect(listDiff([], [1, 2])).toEqual([[], [1, 2]])
		expect(listDiff([1, 2], [])).toEqual([[1, 2], []])
		expect(listDiff([], [])).toEqual([[], []])
	})

	test("uses set-based comparison when either array reaches minSizeForSet", () => {
		const big = Array.from({ length: 10 }, (_, i) => i)
		const [added, removed] = listDiff([...big, 100], big.slice(0, 9))
		expect(added).toEqual([9, 100])
		expect(removed).toEqual([])
	})

	test("set path handles duplicates the same way as the linear path", () => {
		const oldArr = Array.from({ length: 12 }, (_, i) => i)
		const newArr = Array.from({ length: 12 }, (_, i) => i + 6)
		const [added, removed] = listDiff(newArr, oldArr)
		expect(added).toEqual([12, 13, 14, 15, 16, 17])
		expect(removed).toEqual([0, 1, 2, 3, 4, 5])
	})
})

describe("objectDiff", () => {
	test("returns entries for changed values", () => {
		const diff = objectDiff({ a: 1, b: 3 }, { a: 1, b: 2 })
		expect(diff.size).toBe(1)
		expect(diff.get("b")).toEqual({ old: 2, new: 3 })
	})

	test("returns empty map for equal objects", () => {
		expect(objectDiff({ a: 1 }, { a: 1 }).size).toBe(0)
	})

	test("added keys have undefined old value without a default", () => {
		const diff = objectDiff({ a: 1, b: 2 }, { a: 1 })
		expect(diff.size).toBe(1)
		expect(diff.get("b")).toEqual({ old: undefined, new: 2 })
	})

	test("removed keys have undefined new value without a default", () => {
		const diff = objectDiff({ a: 1 }, { a: 1, b: 2 })
		expect(diff.get("b")).toEqual({ old: 2, new: undefined })
	})

	test("defaultValue fills both sides of added and removed keys", () => {
		// b: changed 0 -> 2; c: removed from old side, filled with default 0
		const diff = objectDiff({ a: 1, b: 2 }, { a: 1, b: 0, c: 7 }, 0)
		expect(diff.get("b")).toEqual({ old: 0, new: 2 })
		expect(diff.get("c")).toEqual({ old: 7, new: 0 })
		expect(diff.has("a")).toBe(false)
	})

	test("keys present with the default value are not reported as changed", () => {
		const diff = objectDiff({ a: 0 }, { a: 0 }, 0)
		expect(diff.size).toBe(0)
	})

	test("prevDefaultValue is used for keys missing from the old object", () => {
		const diff = objectDiff({ a: 1, b: 3 }, { a: 1 }, 5, 2)
		// old value for b comes from prevDefaultValue (2), new value is 3
		expect(diff.get("b")).toEqual({ old: 2, new: 3 })
	})

	test("hasOwnProperty distinguishes missing keys from keys holding undefined", () => {
		// b holds an explicit undefined in the old object, so oldVal is undefined,
		// while newVal comes from the new object -> a diff is reported
		const diff = objectDiff({ b: 2 }, { b: undefined })
		expect(diff.get("b")).toEqual({ old: undefined, new: 2 })
	})
})
