// gomuks - A Matrix client written in Go.
// Copyright (C) 2025 Tulir Asokan
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
import { formatDate, formatFullTime, formatShortTime, newSafeDate } from "./datetime"

describe("formatShortTime", () => {
	test("pads single-digit hours and minutes", () => {
		expect(formatShortTime(new Date(2026, 0, 2, 5, 7))).toBe("05:07")
	})

	test("does not pad double-digit hours and minutes", () => {
		expect(formatShortTime(new Date(2026, 0, 2, 13, 42))).toBe("13:42")
	})

	test("handles midnight and 23:59", () => {
		expect(formatShortTime(new Date(2026, 0, 2, 0, 0))).toBe("00:00")
		expect(formatShortTime(new Date(2026, 0, 2, 23, 59))).toBe("23:59")
	})
})

describe("formatFullTime", () => {
	test("formats full date with medium time", () => {
		const date = new Date(2026, 0, 2, 13, 42, 5)
		expect(formatFullTime(date)).toContain("13:42:05")
		expect(formatFullTime(date)).toContain("2026")
	})
})

describe("formatDate", () => {
	test("formats full date without time", () => {
		const date = new Date(2026, 0, 2)
		expect(formatDate(date)).toContain("2026")
		expect(formatDate(date)).not.toContain(":")
	})
})

describe("newSafeDate", () => {
	test("returns the date for a valid timestamp", () => {
		const ts = Date.UTC(2026, 0, 2, 13, 42, 5)
		expect(+newSafeDate(ts)).toBe(ts)
	})

	test("returns epoch for an invalid timestamp", () => {
		expect(+newSafeDate(NaN)).toBe(0)
		expect(+newSafeDate(Infinity)).toBe(0)
	})
})
