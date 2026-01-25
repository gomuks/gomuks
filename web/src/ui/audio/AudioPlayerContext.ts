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
import React, { createContext, use } from "react"
import type { MemDBEvent } from "@/api/types"

export const calculateClickPercent = (e: React.MouseEvent<HTMLElement>): number => {
	const rect = e.currentTarget.getBoundingClientRect()
	return Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width))
}

export interface AudioMetadata {
	event: MemDBEvent
	senderMemberEvent: MemDBEvent | null
	roomName: string | null
	duration?: number
}

export interface AudioPlayerState {
	mediaURL: string | null
	isPlaying: boolean
	duration: number
	metadata: AudioMetadata | null
}

export const INITIAL_AUDIO_PLAYER_STATE: AudioPlayerState = {
	mediaURL: null,
	isPlaying: false,
	duration: 0,
	metadata: null,
}

export interface AudioPlayerContextFields {
	state: AudioPlayerState
	play: (mediaURL: string, metadata?: AudioMetadata) => void
	pause: () => void
	resume: () => void
	seek: (time: number) => void
	close: () => void
	getAudioElement: () => HTMLAudioElement | null
	revealGlobalPlayer: () => void
}

const AudioPlayerContext = createContext<AudioPlayerContextFields | null>(null)

export const useAudioPlayer = (): AudioPlayerContextFields => {
	const context = use(AudioPlayerContext)
	if (!context) {
		throw new Error("useAudioPlayer must be used within an AudioPlayer provider")
	}
	return context
}

export default AudioPlayerContext
