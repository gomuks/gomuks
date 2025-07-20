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
import React, { use } from "react"
import Client from "@/api/client.ts"
import { getRoomAvatarThumbnailURL } from "@/api/media.ts"
import type { RoomID } from "@/api/types"
import { useEventAsState } from "@/util/eventdispatcher.ts"
import { SpaceMenu, getModalStyleFromMouse } from "../menu"
import { ModalContext } from "../modal"
import UnreadCount from "./UnreadCount.tsx"
import "./RoomList.css"

export interface SpaceProps {
	roomID: RoomID
	client: Client
	onClick: (evt: React.MouseEvent<HTMLDivElement>) => void
	onClickUnread: (evt: React.MouseEvent<HTMLDivElement>) => void
	isActive: boolean
}

const Space = ({
	roomID, client, onClick, isActive, onClickUnread,
}: SpaceProps) => {
	const unreads = useEventAsState(client.store.spaceEdges.get(roomID)?.counts)
	const roomStore = client.store.rooms.get(roomID)
	const room = useEventAsState(roomStore?.meta)
	const openModal = use(ModalContext)
	if (!room || !roomStore) {
		return
	}
	const onContextMenu = (evt: React.MouseEvent<HTMLDivElement>) => {
		const realRoom = client.store.rooms.get(room.room_id)
		const edgeStore = client.store.spaceEdges.get(room.room_id)
		if (!realRoom || !edgeStore) {
			// TODO implement separate menu for invite rooms
			console.error("Room state store not found for", room.room_id)
			return
		}
		openModal({
			content: <SpaceMenu
				room={roomStore}
				space={edgeStore}
				style={getModalStyleFromMouse(evt, SpaceMenu.height + edgeStore.childSpaces.size * 40)}
			/>,
		})
		evt.preventDefault()
	}
	return <div
		className={`space-entry ${isActive ? "active" : ""}`}
		onClick={onClick}
		onContextMenu={onContextMenu}
		data-target-space={roomID}
	>
		<UnreadCount counts={unreads} space={true} onClick={onClickUnread} />
		<img src={getRoomAvatarThumbnailURL(room)} alt={room.name} title={room.name} className="avatar" />
	</div>
}

export default Space
