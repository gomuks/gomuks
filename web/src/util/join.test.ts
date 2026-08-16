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
import { describe, expect, test } from "vitest"
import { humanJoin } from "./join"

describe("humanJoin", () => {
	test("empty array returns empty string", () => {
		expect(humanJoin([])).toBe("")
	})

	test("single element is returned as-is", () => {
		expect(humanJoin(["alice"])).toBe("alice")
	})

	test("two elements are joined with lastSep", () => {
		expect(humanJoin(["alice", "bob"])).toBe("alice and bob")
	})

	test("three elements use sep for all but the last pair", () => {
		expect(humanJoin(["alice", "bob", "carol"])).toBe("alice, bob and carol")
	})

	test("many elements", () => {
		expect(humanJoin(["a", "b", "c", "d"])).toBe("a, b, c and d")
	})

	test("custom separators", () => {
		expect(humanJoin(["a", "b", "c"], "; ", " oder ")).toBe("a; b oder c")
		expect(humanJoin(["a", "b"], "; ", " oder ")).toBe("a oder b")
	})

	test("strips bidi control characters from every element", () => {
		expect(humanJoin(["\u202Eevil\u202E"])).toBe("evil")
		expect(humanJoin(["\u202Abob\u202C", "alice"])).toBe("bob and alice")
		expect(humanJoin(["a", "\u202Eb\u202E", "c"])).toBe("a, b and c")
	})
})
