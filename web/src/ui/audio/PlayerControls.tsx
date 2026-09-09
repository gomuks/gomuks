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
import React, { useCallback, useRef } from "react"
import { getAvatarThumbnailURL, getUserColorIndex } from "@/api/media.ts"
import type { MemDBEvent } from "@/api/types"
import { formatTime } from "@/util/time.ts"
import { getDisplayname } from "@/util/validation.ts"
import { calculateClickPercent } from "./AudioPlayerContext.ts"
import { useAudioAnimation } from "./useProgressBarAnimation.ts"
import CloseIcon from "@/icons/close.svg?react"
import PauseIcon from "@/icons/pause.svg?react"
import PlayIcon from "@/icons/play.svg?react"

interface PlayerControlsProps {
	// Core (required)
	isPlaying: boolean
	onPlayPause: () => void
	getAudioElement: () => HTMLAudioElement | null
	isActive: boolean
	duration: number
	onSeek: (time: number) => void

	// Optional features - event info for navigation and sender display
	event?: MemDBEvent
	senderMemberEvent?: MemDBEvent | null
	roomName?: string | null
	onClose?: () => void

	// Styling
	className?: string
	style?: React.CSSProperties
}

const PlayerControls = ({
	isPlaying,
	onPlayPause,
	getAudioElement,
	isActive,
	duration: initialDuration,
	onSeek,
	event,
	senderMemberEvent,
	roomName,
	onClose,
	className,
	style,
}: PlayerControlsProps) => {
	const progressBarRef = useRef<HTMLDivElement>(null)
	const currentTimeRef = useRef<HTMLSpanElement>(null)
	const durationRef = useRef<HTMLSpanElement>(null)

	// Derive sender info from events
	const senderID = event?.sender
	const senderName = senderID ? getDisplayname(senderID, senderMemberEvent?.content) : null
	const senderAvatarURL = senderID ? getAvatarThumbnailURL(senderID, senderMemberEvent?.content) : null

	const handleTrackInfoClick = useCallback(() => {
		if (event) {
			window.mainScreenContext.setActiveRoom(event.room_id, { openEventID: event.event_id })
		}
	}, [event])

	useAudioAnimation({
		getAudioElement,
		isPlaying,
		isActive,
		onUpdate: (currentTime, duration, progress) => {
			if (progressBarRef.current) {
				progressBarRef.current.style.transform = `translateX(${(progress - 1) * 100}%)`
			}
			if (currentTimeRef.current) {
				currentTimeRef.current.textContent = formatTime(currentTime)
			}
			if (durationRef.current && duration > 0) {
				durationRef.current.textContent = formatTime(duration)
			}
		},
	})

	const handleProgressClick = (e: React.MouseEvent<HTMLDivElement>) => {
		const percent = calculateClickPercent(e)
		const audio = getAudioElement()
		const duration = audio?.duration || initialDuration
		if (duration > 0) {
			onSeek(percent * duration)
			if (progressBarRef.current) {
				progressBarRef.current.style.transform = `translateX(${(percent - 1) * 100}%)`
			}
		}
	}

	return (
		<div className={className} style={style}>
			<button className="play-pause-btn" onClick={onPlayPause} title={isPlaying ? "Pause" : "Play"}>
				{isPlaying ? <PauseIcon /> : <PlayIcon />}
			</button>
			{senderID && senderName && senderAvatarURL && (
				<div className="sender-info" onClick={handleTrackInfoClick}>
					<img
						className="avatar"
						loading="lazy"
						src={senderAvatarURL}
						alt=""
					/>
					<div className="track-info">
						<span className={`sender-name sender-color-${getUserColorIndex(senderID)}`}>
							{senderName}
						</span>
						{roomName && <span className="room-name">{roomName}</span>}
					</div>
				</div>
			)}
			<span ref={currentTimeRef} className="time">{formatTime(0)}</span>
			<div className="progress-container" onClick={handleProgressClick}>
				<div ref={progressBarRef} className="progress-bar" />
			</div>
			<span ref={durationRef} className="time">{formatTime(initialDuration)}</span>
			{onClose && (
				<button className="close-btn" onClick={onClose} title="Close">
					<CloseIcon />
				</button>
			)}
		</div>
	)
}

export default PlayerControls
