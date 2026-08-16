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
import { afterEach, describe, expect, test, vi } from "vitest"

// Both constants are evaluated at import time against `window`/`navigator`,
// so each scenario stubs the globals and re-imports the module fresh.
async function loadModule(win: Record<string, unknown>, userAgent: string) {
	vi.resetModules()
	vi.stubGlobal("window", win)
	vi.stubGlobal("navigator", { userAgent })
	return await import("./ismobile")
}

afterEach(() => {
	vi.unstubAllGlobals()
})

describe("isMobileDevice", () => {
	test("true when touch is supported and the viewport is narrow", async () => {
		const { isMobileDevice } = await loadModule({ ontouchstart: null, innerWidth: 799 }, "test")
		expect(isMobileDevice).toBe(true)
	})

	test("false when touch is supported but the viewport is wide", async () => {
		const { isMobileDevice } = await loadModule({ ontouchstart: null, innerWidth: 800 }, "test")
		expect(isMobileDevice).toBe(false)
	})

	test("false when touch is unsupported regardless of width", async () => {
		const { isMobileDevice } = await loadModule({ innerWidth: 400 }, "test")
		expect(isMobileDevice).toBe(false)
	})
})

describe("hackyIsSafari", () => {
	test("true for a WebKit user agent without Chrome", async () => {
		const { hackyIsSafari } = await loadModule(
			{}, "Mozilla/5.0 (Macintosh) AppleWebKit/605.1.15 Version/17.4 Safari/605.1.15",
		)
		expect(hackyIsSafari).toBe(true)
	})

	test("false for a Chrome user agent containing WebKit", async () => {
		const { hackyIsSafari } = await loadModule(
			{}, "Mozilla/5.0 AppleWebKit/537.36 Chrome/124.0 Safari/537.36",
		)
		expect(hackyIsSafari).toBe(false)
	})

	test("false when window.chrome exists", async () => {
		const { hackyIsSafari } = await loadModule(
			{ chrome: {} }, "Mozilla/5.0 AppleWebKit/605.1.15 Version/17.4",
		)
		expect(hackyIsSafari).toBe(false)
	})

	test("false for a non-WebKit user agent", async () => {
		const { hackyIsSafari } = await loadModule({}, "Mozilla/5.0 Firefox/125.0")
		expect(hackyIsSafari).toBe(false)
	})
})
