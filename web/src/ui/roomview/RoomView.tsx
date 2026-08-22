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
import { JSX, Suspense, lazy, useEffect, useState } from "react"
import { GridLoader } from "react-spinners"
import { RoomStateStore, usePreference } from "@/api/statestore"
import { RoomType } from "@/api/types"
import MessageComposer from "../composer/MessageComposer.tsx"
import TypingNotifications from "../composer/TypingNotifications.tsx"
import RightPanel, { RightPanelProps } from "../rightpanel/RightPanel.tsx"
import TimelineView from "../timeline/TimelineView.tsx"
import ErrorBoundary from "../util/ErrorBoundary.tsx"
import { jumpToEvent } from "../util/jumpToEvent.tsx"
import ElementCall from "../widget/ElementCall.tsx"
import RoomViewHeader from "./RoomViewHeader.tsx"
import SpaceView from "./SpaceView.tsx"
import { RoomContext, RoomContextData } from "./roomcontext.ts"
import "./RoomView.css"

interface RoomViewProps {
	room: RoomStateStore
	rightPanel: RightPanelProps | null
	rightPanelResizeHandle: JSX.Element
}

const ImagePackView = lazy(() => import("./ImagePackView.tsx"))

function getViewForRoomType(roomType: RoomType | undefined): JSX.Element | null {
	switch (roomType) {
	case "m.space":
		return <SpaceView />
	case "support.feline.policy.lists.msc.v1":
		return null // TODO <PolicyListEditor />
	case "org.matrix.msc3417.call":
		return <ElementCall />
	case "fi.mau.msc2545.image_pack":
		return <Suspense fallback={<div style={{ display: "flex", justifyContent: "center", marginTop: "2rem" }}>
			<GridLoader color="var(--primary-color)" size={20} />
		</div>}>
			<ImagePackView />
		</Suspense>
	default:
		return null
	}
}

const RoomView = ({ room, rightPanelResizeHandle, rightPanel }: RoomViewProps) => {
	const [forceViewType, setForceViewType] = useState<RoomType | null>(null)
	const settingsViewType = usePreference(null, room, "room_view_type")
	const [roomContextData] = useState(() => new RoomContextData(room, setForceViewType))
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
	const hasRightPanel = !!rightPanel
	useEffect(() => {
		roomContextData.scrollToBottom()
	}, [roomContextData, hasRightPanel])
	const viewType = forceViewType ?? settingsViewType ?? room.meta.current.creation_content?.type
	const view = getViewForRoomType(viewType) ?? <>
		<TimelineView/>
		<MessageComposer/>
		<TypingNotifications/>
	</>
	return <RoomContext value={roomContextData}>
		<div className="room-view">
			<ErrorBoundary thing="room header" wrapperClassName="room-header-error">
				<RoomViewHeader room={room}/>
			</ErrorBoundary>
			<ErrorBoundary thing="room timeline" wrapperClassName="room-timeline-error">
				{view}
			</ErrorBoundary>
		</div>
		{rightPanelResizeHandle}
		{hasRightPanel ? <RightPanel {...rightPanel}/> : null}
	</RoomContext>
}

export default RoomView
