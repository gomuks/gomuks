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
import reverseMap from "./reversemap"

describe("reverseMap", () => {
	test("maps in reverse order", () => {
		expect(reverseMap([1, 2, 3], x => x * 2)).toEqual([6, 4, 2])
	})

	test("passes the original (pre-reverse) index to the callback", () => {
		const indexes: number[] = []
		reverseMap(["a", "b", "c"], (_, i) => {
			indexes.push(i)
		})
		expect(indexes).toEqual([2, 1, 0])
	})

	test("callback receives the reversed element with its original index", () => {
		const seen: Array<[string, number]> = []
		reverseMap(["a", "b", "c"], (el, i) => {
			seen.push([el, i])
		})
		expect(seen).toEqual([["c", 2], ["b", 1], ["a", 0]])
	})

	test("empty array returns empty array without calling fn", () => {
		const fn = vi.fn()
		expect(reverseMap([], fn)).toEqual([])
		expect(fn).not.toHaveBeenCalled()
	})

	test("preserves output type", () => {
		expect(reverseMap([1, 2], x => x.toString())).toEqual(["2", "1"])
	})
})
