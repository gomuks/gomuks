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
import { useCallback, useMemo, useRef, useState } from "react"
import type { AudioMetadata, AudioPlayerContextFields } from "./AudioPlayerContext.ts"
import AudioPlayerContext, { INITIAL_AUDIO_PLAYER_STATE } from "./AudioPlayerContext.ts"
import PlayerControls from "./PlayerControls.tsx"
import "./AudioPlayer.css"

interface AudioPlayerProps {
	children: React.ReactNode
	roomListWidth?: number
	rightPanelWidth?: number
}

const AudioPlayer = ({ children, roomListWidth, rightPanelWidth }: AudioPlayerProps) => {
	const [state, setState] = useState(INITIAL_AUDIO_PLAYER_STATE)
	const audioRef = useRef<HTMLAudioElement | null>(null)
	const [globalPlayerRevealed, setGlobalPlayerRevealed] = useState(false)
	const showGlobalPlayer = state.mediaURL !== null && globalPlayerRevealed

	const revealGlobalPlayer = useCallback(() => setGlobalPlayerRevealed(true), [])

	const getAudioElement = useCallback(() => audioRef.current, [])

	const play = useCallback((mediaURL: string, metadata?: AudioMetadata) => {
		if (audioRef.current && audioRef.current.src === mediaURL) {
			audioRef.current.play()
			return
		}

		if (audioRef.current) {
			audioRef.current.pause()
			audioRef.current = null
		}

		const audio = new Audio(mediaURL)
		audioRef.current = audio

		const cleanupAudioHandlers = () => {
			audio.onloadedmetadata = null
			audio.onplay = null
			audio.onpause = null
			audio.onended = null
			audio.onerror = null
		}

		audio.onloadedmetadata = () => {
			setState(s => ({ ...s, duration: audio.duration }))
		}

		audio.onplay = () => {
			setState(s => ({ ...s, isPlaying: true }))
		}

		const handlePlaybackStop = () => setState(s => ({ ...s, isPlaying: false }))
		audio.onpause = handlePlaybackStop
		audio.onended = handlePlaybackStop

		audio.onerror = () => {
			const mediaError = audio.error
			console.error("Audio playback failed:", {
				src: audio.src,
				errorCode: mediaError?.code,
				errorMessage: mediaError?.message,
			})
			cleanupAudioHandlers()
			setState(INITIAL_AUDIO_PLAYER_STATE)
			audioRef.current = null
		}

		audio.play().catch((error: Error) => {
			console.error("Failed to start audio playback:", {
				name: error.name,
				message: error.message,
				src: audio.src,
			})
			cleanupAudioHandlers()
			setState(INITIAL_AUDIO_PLAYER_STATE)
			audioRef.current = null
		})

		setState({
			mediaURL,
			isPlaying: true,
			duration: metadata?.duration ?? 0,
			metadata: metadata ?? null,
		})
	}, [])

	const pause = useCallback(() => {
		audioRef.current?.pause()
	}, [])

	const resume = useCallback(() => {
		audioRef.current?.play()?.catch((error: Error) => {
			console.error("Failed to resume audio:", error.name, error.message)
		})
	}, [])

	const seek = useCallback((time: number) => {
		if (audioRef.current) {
			audioRef.current.currentTime = time
		}
	}, [])

	const close = useCallback(() => {
		if (audioRef.current) {
			audioRef.current.pause()
			audioRef.current = null
		}
		setState(INITIAL_AUDIO_PLAYER_STATE)
		setGlobalPlayerRevealed(false)
	}, [])

	const contextValue: AudioPlayerContextFields = useMemo(() => ({
		state,
		play,
		pause,
		resume,
		seek,
		close,
		getAudioElement,
		revealGlobalPlayer,
	}), [state, play, pause, resume, seek, close, getAudioElement, revealGlobalPlayer])

	return (
		<AudioPlayerContext value={contextValue}>
			{children}
			{showGlobalPlayer && (
				<PlayerControls
					isPlaying={state.isPlaying}
					onPlayPause={state.isPlaying ? pause : resume}
					getAudioElement={getAudioElement}
					isActive={showGlobalPlayer}
					duration={state.duration}
					onSeek={seek}
					event={state.metadata?.event}
					senderMemberEvent={state.metadata?.senderMemberEvent}
					roomName={state.metadata?.roomName}
					onClose={close}
					className="global-audio-player"
					style={{
						"--room-list-width": `${roomListWidth}px`,
						"--right-panel-width": `${rightPanelWidth ?? 0}px`,
					} as React.CSSProperties}
				/>
			)}
		</AudioPlayerContext>
	)
}

export default AudioPlayer
