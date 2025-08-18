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
import { JSX, useEffect, useState } from "react"
import { RoomStateStore } from "@/api/statestore"
import MessageComposer from "../composer/MessageComposer.tsx"
import TypingNotifications from "../composer/TypingNotifications.tsx"
import RightPanel, { RightPanelProps } from "../rightpanel/RightPanel.tsx"
import TimelineView from "../timeline/TimelineView.tsx"
import ErrorBoundary from "../util/ErrorBoundary.tsx"
import { jumpToEvent } from "../util/jumpToEvent.tsx"
import ElementCall from "../widget/ElementCall.tsx"
import RoomViewHeader from "./RoomViewHeader.tsx"
import { RoomContext, RoomContextData } from "./roomcontext.ts"
import "./RoomView.css"

interface RoomViewProps {
	room: RoomStateStore
	rightPanel: RightPanelProps | null
	rightPanelResizeHandle: JSX.Element
}

function getViewForRoomType(roomType: string | undefined): JSX.Element | null {
	switch (roomType) {
	case "m.space":
		return null // TODO <SpaceView />
	case "support.feline.policy.lists.msc.v1":
		return null // TODO <PolicyListEditor />
	case "org.matrix.msc3417.call":
		return <ElementCall />
	default:
		return null
	}
}

const RoomView = ({ room, rightPanelResizeHandle, rightPanel }: RoomViewProps) => {
	const [forceDefaultTimeline, setForceDefaultTimeline] = useState(false)
	const [roomContextData] = useState(() => new RoomContextData(room, setForceDefaultTimeline))
	useEffect(() => {
		if (room.hackyPendingJumpToEventID) {
			jumpToEvent(roomContextData, room.hackyPendingJumpToEventID)
			room.hackyPendingJumpToEventID = null
		}
		window.activeRoomContext = roomContextData
		window.addEventListener("resize", roomContextData.scrollToBottom)
		return () => {
			window.removeEventListener("resize", roomContextData.scrollToBottom)
			if (window.activeRoomContext === roomContextData) {
				window.activeRoomContext = undefined
			}
		}
	}, [room, roomContextData])
	const onClick = (evt: React.MouseEvent<HTMLDivElement>) => {
		if (roomContextData.focusedEventRowID) {
			roomContextData.setFocusedEventRowID(null)
			evt.stopPropagation()
		}
	}
	let view = <>
		<TimelineView/>
		<MessageComposer/>
		<TypingNotifications/>
	</>
	if (!forceDefaultTimeline) {
		view = getViewForRoomType(room.meta.current.creation_content?.type) ?? view
	}
	return <RoomContext value={roomContextData}>
		<div className="room-view" onClick={onClick}>
			<ErrorBoundary thing="room header" wrapperClassName="room-header-error">
				<div className="mobile-event-menu-container" id="mobile-event-menu-container"/>
				<RoomViewHeader room={room}/>
			</ErrorBoundary>
			<ErrorBoundary thing="room timeline" wrapperClassName="room-timeline-error">
				{view}
			</ErrorBoundary>
		</div>
		{rightPanelResizeHandle}
		{rightPanel && <RightPanel {...rightPanel}/>}
	</RoomContext>
}

export default RoomView
