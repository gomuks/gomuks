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
import { getMediaURL } from "@/api/media.ts"

function getRealURL(url: string): string | undefined {
	if (!url) {
		return undefined
	}
	if (url.startsWith("mxc://")) {
		return getMediaURL(url)
	} else if (/^sounds\/[a-z0-9_-]+\.(flac|ogg|wav|mp3)$/.test(url)) {
		return url
	}
	return undefined
}

const sounds = new Map<string, HTMLAudioElement>([
	["sounds/bright.flac", new Audio("sounds/bright.flac")],
])

export function playSound(url: string, volume: number = 100) {
	const realURL = getRealURL(url)
	if (!realURL) {
		return
	}
	let audio = sounds.get(realURL)
	if (!audio) {
		console.log("Loading new notification sound from", realURL)
		audio = new Audio(realURL)
		sounds.set(realURL, audio)
	}
	audio.volume = Math.max(Math.min(volume, 100), 0) / 100
	audio.play()
}
