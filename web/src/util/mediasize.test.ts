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
import {
	calculateMediaSize,
	defaultImageContainerSize,
	defaultVideoContainerSize,
} from "./mediasize"

describe("default container sizes", () => {
	test("has expected default dimensions", () => {
		expect(defaultImageContainerSize).toEqual({ width: 320, height: 240 })
		expect(defaultVideoContainerSize).toEqual({ width: 400, height: 320 })
	})
})

describe("calculateMediaSize", () => {
	test("returns default container when dimensions are missing or zero", () => {
		for (const [width, height] of [[undefined, undefined], [0, 0], [320, 0], [undefined, 240]] as const) {
			const result = calculateMediaSize(width, height)
			expect(result.container).toStrictEqual({
				width: "320px",
				height: "240px",
				containIntrinsicWidth: "320px",
				containIntrinsicHeight: "240px",
				contentVisibility: "auto",
				contain: "strict",
			})
			expect(result.media).toStrictEqual({})
		}
	})

	test("uses given container size when dimensions are missing", () => {
		const result = calculateMediaSize(undefined, undefined, { width: 100, height: 50 })
		expect(result.container.width).toBe("100px")
		expect(result.container.height).toBe("50px")
	})

	test("keeps small images inside container unchanged when above minimums", () => {
		const result = calculateMediaSize(100, 100)
		expect(result.container.width).toBe("100px")
		expect(result.container.height).toBe("100px")
		expect(result.media).toStrictEqual({ aspectRatio: "100 / 100" })
	})

	test("caps images wider than the container aspect ratio by width", () => {
		// aspect ratio 2.0 > container 4/3: width caps at 320, height becomes 160
		const result = calculateMediaSize(640, 320)
		expect(result.container.width).toBe("320px")
		expect(result.container.height).toBe("160px")
		expect(result.media.aspectRatio).toBe("640 / 320")
	})

	test("caps images taller than the container aspect ratio by height", () => {
		// aspect ratio 0.5 < 4/3: height caps at 240, width becomes 120
		const result = calculateMediaSize(240, 480)
		expect(result.container.width).toBe("120px")
		expect(result.container.height).toBe("240px")
	})

	test("caps images with exactly the container aspect ratio on both axes", () => {
		const result = calculateMediaSize(640, 480)
		expect(result.container.width).toBe("320px")
		expect(result.container.height).toBe("240px")
	})

	test("keeps aspect ratio in media style pointing at original dimensions", () => {
		const result = calculateMediaSize(640, 320)
		expect(result.media.aspectRatio).toBe("640 / 320")
	})

	test("enforces minimum height with cover-cropped media for very short images", () => {
		// 60x20: height 20 < 40 -> height forced to 40, width to 120, media cropped
		const result = calculateMediaSize(60, 20)
		expect(result.container.width).toBe("120px")
		expect(result.container.height).toBe("40px")
		expect(result.media.objectFit).toBe("cover")
		expect(result.media.height).toBe("100%")
		expect(result.media.width).toBeUndefined()
	})

	test("enforces minimum width with cover-cropped media for very narrow images", () => {
		// 20x60: width 20 < 40 -> width forced to 40, height recomputed to 120
		const result = calculateMediaSize(20, 60)
		expect(result.container.width).toBe("40px")
		expect(result.container.height).toBe("120px")
		expect(result.media.objectFit).toBe("cover")
		expect(result.media.width).toBe("100%")
	})

	test("square tiny images are upscaled to the minimum via the height branch", () => {
		const result = calculateMediaSize(10, 10)
		expect(result.container.width).toBe("40px")
		expect(result.container.height).toBe("40px")
		expect(result.media.objectFit).toBe("cover")
		expect(result.media.height).toBe("100%")
	})

	test("respects custom image container size", () => {
		// container 400x320 (default video) vs 640x320 image: AR 2.0 > 1.25
		const result = calculateMediaSize(640, 320, defaultVideoContainerSize)
		expect(result.container.width).toBe("400px")
		expect(result.container.height).toBe("200px")
	})

	test("always sets content visibility hints on the container", () => {
		const result = calculateMediaSize(100, 100)
		expect(result.container.contentVisibility).toBe("auto")
		expect(result.container.contain).toBe("strict")
		expect(result.container.containIntrinsicWidth).toBe("100px")
		expect(result.container.containIntrinsicHeight).toBe("100px")
	})
})
