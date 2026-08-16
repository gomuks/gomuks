// gomuks - A Matrix client written in Go.
// Copyright (C) 2026 Tulir Asokan
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
import { beforeAll, beforeEach, describe, expect, test, vi } from "vitest"

const mediaURLMock = vi.hoisted(() => vi.fn((mxc: string) => `_gomuks/media/${mxc}`))
vi.mock("@/api/media.ts", () => ({
	getMediaURL: mediaURLMock,
}))

class FakeAudioImpl {
	src: string
	volume = 1
	play = vi.fn()

	constructor(src: string) {
		this.src = src
		audioInstances.push(this)
	}
}

// Regular function so `new AudioMock(src)` works; returning an object overrides `this`.
const AudioMock = vi.fn(function(this: unknown, src: string) {
	return new FakeAudioImpl(src)
})

const audioInstances: FakeAudioImpl[] = []

let logMock: ReturnType<typeof vi.spyOn>
let playSound: (url: string, volume?: number) => void

beforeAll(async () => {
	vi.stubGlobal("Audio", AudioMock)
	logMock = vi.spyOn(console, "log").mockImplementation(() => {})
	;({ playSound } = await import("./sound"))
})

beforeEach(() => {
	mediaURLMock.mockClear()
	AudioMock.mockClear()
	logMock?.mockClear()
})

describe("playSound", () => {
	test("empty and unknown URLs are ignored without constructing Audio", () => {
		playSound("")
		playSound("not-a-sound")
		playSound("sounds/badextension.exe")
		playSound("sounds/UPPER.flac")
		expect(mediaURLMock).not.toHaveBeenCalled()
		expect(AudioMock).not.toHaveBeenCalled()
		for (const audio of audioInstances) {
			expect(audio.play).not.toHaveBeenCalled()
		}
	})

	test("mxc URLs resolve through getMediaURL, create a new Audio and play", () => {
		playSound("mxc://example.org/abcdef")
		expect(mediaURLMock).toHaveBeenCalledWith("mxc://example.org/abcdef")
		expect(logMock).toHaveBeenCalledWith("Loading new notification sound from", "_gomuks/media/mxc://example.org/abcdef")
		const audio = audioInstances.at(-1)!
		expect(audio.src).toBe("_gomuks/media/mxc://example.org/abcdef")
		expect(audio.volume).toBe(1)
		expect(audio.play).toHaveBeenCalledTimes(1)
	})

	test("a valid local sound URL plays without loading a new Audio", () => {
		playSound("sounds/bright.flac")
		expect(mediaURLMock).not.toHaveBeenCalled()
		expect(logMock).not.toHaveBeenCalled()
		expect(AudioMock).not.toHaveBeenCalledWith("sounds/bright.flac")
		const preloaded = audioInstances.find(audio => audio.src === "sounds/bright.flac")
		expect(preloaded).toBeDefined()
		expect(preloaded!.play).toHaveBeenCalledTimes(1)
	})

	test("other local sound extensions match the whitelist regex", () => {
		playSound("sounds/chime_v2.ogg")
		expect(AudioMock).toHaveBeenCalledWith("sounds/chime_v2.ogg")
		expect(audioInstances.at(-1)!.play).toHaveBeenCalledTimes(1)
	})

	test("the same mxc sound is cached and not reconstructed", () => {
		playSound("mxc://example.org/cached")
		playSound("mxc://example.org/cached")
		const instances = audioInstances.filter(audio => audio.src === "_gomuks/media/mxc://example.org/cached")
		expect(instances).toHaveLength(1)
		expect(instances[0].play).toHaveBeenCalledTimes(2)
	})

	test("volume is clamped to [0, 1]", () => {
		const preloaded = audioInstances.find(audio => audio.src === "sounds/bright.flac")!
		preloaded.play.mockClear()
		playSound("sounds/bright.flac", 150)
		playSound("sounds/bright.flac", -5)
		playSound("sounds/bright.flac", 50)
		expect(preloaded.volume).toBeCloseTo(0.5)
		expect(preloaded.play).toHaveBeenCalledTimes(3)
	})
})
