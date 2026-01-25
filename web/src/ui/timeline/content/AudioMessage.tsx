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
import type { MemDBEvent } from "@/api/types"
import type { AudioMetadata } from "@/ui/audio"
import { PlayerControls, useAudioPlayer } from "@/ui/audio"

export interface AudioMessageProps {
	src: string
	duration?: number
	event?: MemDBEvent
	senderMemberEvent?: MemDBEvent | null
	roomName?: string | null
}

const AudioMessage = ({
	src,
	duration: initialDurationMs,
	event,
	senderMemberEvent,
	roomName,
}: AudioMessageProps) => {
	const audioPlayer = useAudioPlayer()
	const isThisTrack = audioPlayer.state.mediaURL === src
	const isPlaying = isThisTrack && audioPlayer.state.isPlaying
	const initialDurationSec = initialDurationMs ? initialDurationMs / 1000 : 0

	const { revealGlobalPlayer } = audioPlayer
	const isPlayingRef = useRef(isPlaying)
	isPlayingRef.current = isPlaying

	// Reveal global player only when this component unmounts while actively playing
	// (e.g., when scrolling away from the message)
	useEffect(() => {
		return () => {
			if (isPlayingRef.current) {
				revealGlobalPlayer()
			}
		}
	}, [revealGlobalPlayer])

	const handlePlayPause = () => {
		if (isThisTrack) {
			if (isPlaying) {
				audioPlayer.pause()
			} else {
				audioPlayer.resume()
			}
		} else {
			const metadata: AudioMetadata | undefined = event ? {
				event,
				senderMemberEvent: senderMemberEvent ?? null,
				roomName: roomName ?? null,
				duration: initialDurationSec,
			} : undefined
			audioPlayer.play(src, metadata)
		}
	}

	const handleSeek = (time: number) => {
		if (isThisTrack) {
			audioPlayer.seek(time)
		}
	}

	return (
		<PlayerControls
			isPlaying={isPlaying}
			onPlayPause={handlePlayPause}
			getAudioElement={audioPlayer.getAudioElement}
			isActive={isThisTrack}
			duration={initialDurationSec}
			onSeek={handleSeek}
			className="audio-message-player"
		/>
	)
}

export default AudioMessage
