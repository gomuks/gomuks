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
import React, { use, useCallback, useRef, useState } from "react"
import { BarLoader } from "react-spinners"
import { getAvatarThumbnailURL } from "@/api/media.ts"
import { RoomListFilter, Space as SpaceStore, SpaceUnreadCounts, usePreference } from "@/api/statestore"
import { useTabs } from "@/api/tabs.ts"
import type { RoomID } from "@/api/types"
import { useEventAsState } from "@/util/eventdispatcher.ts"
import reverseMap from "@/util/reversemap.ts"
import toSearchableString from "@/util/searchablestring.ts"
import ClientContext from "../ClientContext.ts"
import MainScreenContext from "../MainScreenContext.ts"
import { keyToString } from "../keybindings.ts"
import { getRightOpeningModalStyleFromButton } from "../menu/util.ts"
import { ModalContext, modals } from "../modal"
import Entry from "./Entry.tsx"
import FakeSpace from "./FakeSpace.tsx"
import ProfileSwitcher from "./ProfileSwitcher.tsx"
import Space from "./Space.tsx"
import AddCircleIcon from "@/icons/add-circle.svg?react"
import CloseIcon from "@/icons/close.svg?react"
import SearchIcon from "@/icons/search.svg?react"
import "./RoomList.css"

interface RoomListProps {
	activeRoomID: RoomID | null
	space: RoomListFilter | null
}

const RoomList = ({ activeRoomID, space }: RoomListProps) => {
	const client = use(ClientContext)!
	const openModal = use(ModalContext)
	const mainScreen = use(MainScreenContext)
	const roomList = useEventAsState(client.store.roomList)
	const spaces = useEventAsState(client.store.topLevelSpaces)
	const initComplete = useEventAsState(client.initComplete)
	const ownProfile = useEventAsState(client.profile)
	const searchInputRef = useRef<HTMLInputElement>(null)
	const [query, directSetQuery] = useState("")
	const [tabs, currentTabID, totalUnreads, switchTab] = useTabs()
	const currentTabIndex = tabs.findIndex(t => t.id === currentTabID)

	const setQuery = (evt: React.ChangeEvent<HTMLInputElement>) => {
		client.store.currentRoomListQuery = toSearchableString(evt.target.value)
		directSetQuery(evt.target.value)
	}
	const openCreateRoom = () => {
		openModal(modals.createRoom())
	}
	const onClickSpace = useCallback((evt: React.MouseEvent<HTMLDivElement>) => {
		const store = client.store.getSpaceStore(evt.currentTarget.getAttribute("data-target-space")!)
		mainScreen.setSpace(store)
	}, [mainScreen, client])
	const onClickSpaceUnread = useCallback((
		evt: React.MouseEvent<HTMLDivElement>, space?: SpaceStore | null,
	) => {
		if (!space) {
			const targetSpace = evt.currentTarget.closest("div.space-entry")?.getAttribute("data-target-space")
			if (!targetSpace) {
				return
			}
			space = client.store.getSpaceStore(targetSpace)
			if (!space) {
				return
			}
		}
		const counts = space.counts.current
		let wantedField: keyof SpaceUnreadCounts
		if (counts.unread_highlights > 0) {
			wantedField = "unread_highlights"
		} else if (counts.unread_notifications > 0) {
			wantedField = "unread_notifications"
		} else if (counts.unread_messages > 0) {
			wantedField = "unread_messages"
		} else {
			return
		}
		for (let i = client.store.roomList.current.length - 1; i >= 0; i--) {
			const entry = client.store.roomList.current[i]
			if (entry[wantedField] > 0 && space.include(entry)) {
				mainScreen.setActiveRoom(entry.room_id, { toSpace: space })
				evt.stopPropagation()
				return
			}
		}
		console.warn(
			"No room found with unreads in space", space.id, "with field", wantedField,
			"- looking for hidden rooms",
		)
		for (const item of client.store.rooms.values()) {
			if (item.meta.current[wantedField] > 0 && space.include({
				...item.meta.current,
				search_name: item.meta.current.name || "",
				name: item.meta.current.name || "",
			})) {
				mainScreen.setActiveRoom(item.roomID, { toSpace: space })
				evt.stopPropagation()
				return
			}
		}
	}, [mainScreen, client])
	const clearQuery = () => {
		client.store.currentRoomListQuery = ""
		directSetQuery("")
		searchInputRef.current?.focus()
	}
	const onKeyDown = (evt: React.KeyboardEvent<HTMLInputElement>) => {
		const key = keyToString(evt)
		if (key === "Escape") {
			clearQuery()
			evt.stopPropagation()
			evt.preventDefault()
		} else if (key === "Enter") {
			const roomList = client.store.getFilteredRoomList()
			mainScreen.setActiveRoom(roomList[roomList.length-1]?.room_id)
			clearQuery()
			evt.stopPropagation()
			evt.preventDefault()
		}
	}

	const showInviteAvatars = usePreference(client.store, null, "show_invite_avatars")
	const roomListFilter = client.store.roomListFilterFunc
	const onWrapperKeyDown = (evt: React.KeyboardEvent<HTMLDivElement>) => {
		if (keyToString(evt) === "Escape" && query !== "") {
			clearQuery()
			evt.stopPropagation()
			evt.preventDefault()
		}
	}
	return <div className="room-list-wrapper" onKeyDown={onWrapperKeyDown}>
		<div className="room-search-wrapper">
			<input
				value={query}
				onChange={setQuery}
				onKeyDown={onKeyDown}
				className="room-search"
				type="text"
				placeholder="Search rooms"
				ref={searchInputRef}
				id="room-search"
			/>
			{query === "" && <button onClick={openCreateRoom} title="Create room">
				<AddCircleIcon/>
			</button>}
			<button onClick={clearQuery} disabled={query === ""}>
				{query !== "" ? <CloseIcon/> : <SearchIcon/>}
			</button>
		</div>
		<div className={`space-bar ${currentTabID ? "has-profiles" : "no-profiles"}`}>
			<FakeSpace space={null} setSpace={mainScreen.setSpace} isActive={space === null} />
			{client.store.pseudoSpaces.map(pseudoSpace => <FakeSpace
				key={pseudoSpace.id}
				space={pseudoSpace}
				setSpace={mainScreen.setSpace}
				onClickUnread={onClickSpaceUnread}
				isActive={space?.id === pseudoSpace.id}
			/>)}
			{spaces.map(roomID => <Space
				key={roomID}
				roomID={roomID}
				client={client}
				onClick={onClickSpace}
				isActive={space?.id === roomID}
				onClickUnread={onClickSpaceUnread}
			/>)}
		</div>
		{currentTabID && <div className="profile-switcher">
			<div
				className="profile-switcher-item"
				onClick={evt => {
					openModal({
						content: <ProfileSwitcher
							tabs={tabs}
							currentTabID={currentTabID}
							switchTab={switchTab}
							style={getRightOpeningModalStyleFromButton(evt.currentTarget, 40 * tabs.length + 4)}
						/>,
					})
				}}
			>
				{totalUnreads > 0 ? <div className="room-entry-unreads floating">
					<div className="unread-count space notified">
						{totalUnreads > 999 ? "99+" : totalUnreads}
					</div>
				</div> : null}
				<img
					src={tabs[currentTabIndex]?.icon ?? getAvatarThumbnailURL(client.userID, ownProfile)}
					className="avatar"
					alt="Profile switcher"
				/>
			</div>
		</div>}
		<div className="room-list">
			{initComplete ? null
				: <BarLoader cssOverride={{ backgroundColor: "unset" }} width="100%" color="var(--primary-color)" />}
			{reverseMap(roomList, room =>
				<Entry
					key={room.room_id}
					isActive={room.room_id === activeRoomID}
					hidden={roomListFilter ? !roomListFilter(room) : false}
					room={room}
					hideAvatar={room.is_invite && !showInviteAvatars}
				/>,
			)}
		</div>
	</div>
}

export default RoomList
