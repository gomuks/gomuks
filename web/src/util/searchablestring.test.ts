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
import toSearchableString from "./searchablestring"

describe("toSearchableString", () => {
	test("empty string", () => {
		expect(toSearchableString("")).toBe("")
	})

	test("lowercases ASCII", () => {
		expect(toSearchableString("Hello World")).toBe("helloworld")
	})

	test("removes ASCII punctuation", () => {
		expect(toSearchableString("hello, world! (test) [x] {y}")).toBe("helloworldtestxy")
	})

	test("unhomoglyph maps confusable ASCII before punctuation is stripped", () => {
		// 1 -> l, m -> rn per the confusables table
		expect(toSearchableString("#1")).toBe("l")
		expect(toSearchableString("m")).toBe("rn")
	})

	test("removes whitespace", () => {
		expect(toSearchableString("a b\tc\nd\re\ufeff")).toBe("abcde")
	})

	test("removes hidden and formatting characters", () => {
		// en quad, RTL/LTR embeds, combining circumflex, arabic letter mark, braille blank, invisible separators
		expect(toSearchableString("a\u2002b\u202Ac\u202Ed\u0302e\u061Cf\u2800g\u2062h\u2063")).toBe("abcdefgh")
	})

	test("removes general punctuation and supplemental punctuation blocks", () => {
		expect(toSearchableString("a\u2013b\u2050c\u2e00d\u2e7f")).toBe("abcd")
	})

	test("strips combining marks via NFD", () => {
		// composed é (U+00E9) and decomposed e + combining acute (U+0065 U+0301) both normalize the same
		expect(toSearchableString("\u00E9")).toBe("e")
		expect(toSearchableString("e\u0301")).toBe("e")
	})

	test("maps homoglyphs to ascii equivalents", () => {
		// Cyrillic а, ѕ and о look like latin a, s and o; latin m itself maps to rn
		expect(toSearchableString("аdmin")).toBe("adrnin")
		expect(toSearchableString("ѕоme")).toBe(toSearchableString("some"))
	})

	test("cyrillic and latin lookalike strings collide", () => {
		expect(toSearchableString("ѕоme")).toBe(toSearchableString("some"))
	})
})
