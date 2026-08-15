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
import { useEffect, useRef } from "react"

interface UseAudioAnimationOptions {
	getAudioElement: () => HTMLAudioElement | null
	isPlaying: boolean
	isActive: boolean
	onUpdate: (currentTime: number, duration: number, progress: number) => void
}

export function useAudioAnimation({
	getAudioElement,
	isPlaying,
	isActive,
	onUpdate,
}: UseAudioAnimationOptions): void {
	const animationFrameRef = useRef<number | null>(null)
	const onUpdateRef = useRef(onUpdate)
	onUpdateRef.current = onUpdate

	useEffect(() => {
		if (!isPlaying || !isActive) {
			if (animationFrameRef.current) {
				cancelAnimationFrame(animationFrameRef.current)
				animationFrameRef.current = null
			}
			return
		}

		const update = () => {
			const audio = getAudioElement()
			if (audio) {
				const progress = audio.duration > 0 ? audio.currentTime / audio.duration : 0
				onUpdateRef.current(audio.currentTime, audio.duration, progress)
			}
			animationFrameRef.current = requestAnimationFrame(update)
		}

		animationFrameRef.current = requestAnimationFrame(update)

		return () => {
			if (animationFrameRef.current) {
				cancelAnimationFrame(animationFrameRef.current)
			}
		}
	}, [isPlaying, isActive, getAudioElement])
}
