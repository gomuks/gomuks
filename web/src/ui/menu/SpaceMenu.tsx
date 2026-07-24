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
import React, { CSSProperties, use, useState } from "react"
import Client from "@/api/client.ts"
import { getRoomAvatarThumbnailURL } from "@/api/media.ts"
import { RoomStateStore, SpaceEdgeStore } from "@/api/statestore"
import { useEventAsState } from "@/util/eventdispatcher.ts"
import ClientContext from "../ClientContext.ts"
import MainScreenContext from "../MainScreenContext.ts"
import { ModalCloseContext, ModalContext, modals } from "../modal"
import { getRightOpeningModalStyleFromButton } from "./util.ts"
import ChatIcon from "@/icons/chat.svg?react"
import SettingsIcon from "@/icons/settings.svg?react"
import ShareIcon from "@/icons/share.svg?react"
import "./RoomMenu.css"

interface ChildSpaceProps {
	client: Client
	child: SpaceEdgeStore
	onClick: (evt: React.MouseEvent<HTMLDivElement>) => void
}

const ChildSpace = ({ client, child, onClick }: ChildSpaceProps) => {
	const [openStyle, setOpenStyle] = useState<CSSProperties | null>(null)
	const [focused, setFocused] = useState(false)
	const [mouseOver, setMouseOver] = useState(false)
	const closeModal = use(ModalCloseContext)
	const room = client.store.rooms.get(child.id)
	const roomMeta = useEventAsState(room?.meta)
	if (!room || !roomMeta) {
		return null
	}
	const onMouseEnter = (evt: React.MouseEvent<HTMLDivElement>) => {
		setOpenStyle(getRightOpeningModalStyleFromButton(
			evt.currentTarget, SpaceMenu.height + child.childSpaces.size * 40,
		))
		setMouseOver(true)
	}
	const onFocus = (evt: React.FocusEvent<HTMLDivElement>) => {
		setOpenStyle(getRightOpeningModalStyleFromButton(
			evt.currentTarget, SpaceMenu.height + child.childSpaces.size * 40,
		))
		setFocused(true)
	}
	return <div
		className="context-menu-item space-list-child"
		data-target-space={child.id}
		onMouseEnter={onMouseEnter}
		onFocus={onFocus}
		onMouseLeave={() => setMouseOver(false)}
		onBlur={() => setFocused(false)}
		onClick={evt => {
			closeModal()
			onClick(evt)
		}}
		tabIndex={0}
	>
		<img
			loading="lazy"
			className="avatar room-avatar"
			src={getRoomAvatarThumbnailURL(roomMeta)}
			alt=""
		/>
		<div className="room-name">{roomMeta.name}</div>
		{openStyle && (focused || mouseOver) ?
			<SpaceMenu room={room} space={child} style={openStyle} onClick={onClick} />
			: null}
	</div>
}

interface SpaceMenuProps {
	room: RoomStateStore
	space: SpaceEdgeStore
	style: CSSProperties
	onClick: (evt: React.MouseEvent<HTMLDivElement>) => void
}

export const SpaceMenu = ({ room, space, style, onClick }: SpaceMenuProps) => {
	const openModal = use(ModalContext)
	const closeModal = use(ModalCloseContext)
	const mainScreen = use(MainScreenContext)!
	const client = use(ClientContext)!
	const openSettings = () => {
		closeModal()
		window.openNestableModal(modals.settings(room))
	}
	const openTimeline = () => {
		closeModal()
		mainScreen.setActiveRoom(room.roomID)
	}
	const onClickShare = () => {
		openModal(modals.shareRoom(room))
	}

	return <div className="context-menu space-list-menu" style={style}>
		<button className="context-menu-item" onClick={openSettings}><SettingsIcon /> Settings</button>
		<button onClick={onClickShare}><ShareIcon /> Share</button>
		<button className="context-menu-item" onClick={openTimeline}><ChatIcon /> View timeline</button>
		{space.childSpaces.values()
			.map(child => <ChildSpace client={client} child={child} key={child.id} onClick={onClick} />)
			.toArray()}
	</div>
}

SpaceMenu.height = 2 * 40
