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
import { CSSProperties, JSX, Ref, use } from "react"
import { RoomListEntry, RoomStateStore, useAccountData } from "@/api/statestore"
import { RoomID } from "@/api/types"
import { useEventAsState } from "@/util/eventdispatcher.ts"
import ClientContext from "../ClientContext.ts"
import { ModalCloseContext, ModalContext, modals } from "../modal"
import DoorOpenIcon from "@/icons/door-open.svg?react"
import FavoriteIcon from "@/icons/favorite.svg?react"
import MarkReadIcon from "@/icons/mark-read.svg?react"
import MarkUnreadIcon from "@/icons/mark-unread.svg?react"
import NotificationsOffIcon from "@/icons/notifications-off.svg?react"
import NotificationsIcon from "@/icons/notifications.svg?react"
import SetLowPriorityIcon from "@/icons/set-low-priority.svg?react"
import SettingsIcon from "@/icons/settings.svg?react"
import ShareIcon from "@/icons/share.svg?react"
import UnfavoriteIcon from "@/icons/unfavorite.svg?react"
import UnsetLowPriorityIcon from "@/icons/unset-low-priority.svg?react"
import "./RoomMenu.css"

interface RoomMenuProps {
	room: RoomStateStore
	entry: RoomListEntry
	style: CSSProperties
	ref?: Ref<HTMLDivElement>
}

const hasNotifyingActions = (actions: unknown) => {
	return Array.isArray(actions) && actions.length > 0 && actions.includes("notify")
}

const MuteButton = ({ roomID }: { roomID: RoomID }) => {
	const client = use(ClientContext)!
	const closeModal = use(ModalCloseContext)
	const roomRules = useAccountData(client.store, "m.push_rules")?.global?.room
	const pushRule = Array.isArray(roomRules) ? roomRules.find(rule => rule?.rule_id === roomID) : null
	const muted = pushRule?.enabled === true && !hasNotifyingActions(pushRule.actions)
	const toggleMute = () => {
		client.rpc.muteRoom(roomID, !muted).catch(err => {
			console.error("Failed to mute room", err)
			window.alert(`Failed to ${muted ? "unmute" : "mute"} room: ${err}`)
		})
		closeModal()
	}
	return <button onClick={toggleMute}>
		{muted ? <NotificationsIcon/> : <NotificationsOffIcon/>}
		{muted ? "Unmute" : "Mute"}
	</button>
}

interface TagButtonProps {
	room: RoomStateStore
	tag: string
	taggedElem: JSX.Element
	untaggedElem: JSX.Element
}

const TagButton = ({ room, tag, taggedElem, untaggedElem }: TagButtonProps) => {
	const client = use(ClientContext)!
	const closeModal = use(ModalCloseContext)
	const tags = room?.accountData.get("m.tag")?.tags ?? {}
	const tagged = !!tags[tag]
	const toggleTag = () => {
		const newTags = {
			...tags,
			[tag]: tagged ? undefined : {},
		}
		client.rpc.setAccountData("m.tag", { tags: newTags }, room.roomID).catch(err => {
			console.error("Failed to tag room", err)
			window.alert(`Failed to ${tagged ? "untag" : "tag"} room: ${err}`)
		})
		closeModal()
	}
	return <button onClick={toggleTag}>
		{tagged ? taggedElem : untaggedElem}
	</button>
}

const MarkReadButton = ({ room }: { room: RoomStateStore }) => {
	const meta = useEventAsState(room.meta)
	const client = use(ClientContext)!
	const closeModal = use(ModalCloseContext)
	const read = !meta.marked_unread && meta.unread_messages === 0
	const markRead = () => {
		const evt = room.eventsByRowID.get(
			room.timeline[room.timeline.length-1]?.event_rowid ?? meta.preview_event_rowid,
		)
		if (!evt) {
			window.alert("Can't mark room as read: last event not found in cache")
			return
		}
		const rrType = room.preferences.send_read_receipts ? "m.read" : "m.read.private"
		client.rpc.markRead(room.roomID, evt.event_id, rrType).catch(err => {
			console.error("Failed to mark room as read", err)
			window.alert(`Failed to mark room as read: ${err}`)
		})
		closeModal()
	}
	const markUnread = () => {
		client.rpc.setAccountData("m.marked_unread", { unread: true }, room.roomID).catch(err => {
			console.error("Failed to mark room as unread", err)
			window.alert(`Failed to mark room as unread: ${err}`)
		})
		closeModal()
	}
	return <button onClick={read ? markUnread : markRead}>
		{read ? <MarkUnreadIcon/> : <MarkReadIcon/>}
		Mark {read ? "unread" : "read"}
	</button>
}

export const RoomMenu = ({ room, style, ref }: RoomMenuProps) => {
	const openModal = use(ModalContext)
	const closeModal = use(ModalCloseContext)
	const client = use(ClientContext)!
	const openSettings = () => {
		closeModal()
		window.openNestableModal(modals.settings(room))
	}
	const leaveRoom = () => {
		if (!window.confirm(`Really leave ${room.meta.current.name}?`)) {
			return
		}
		client.rpc.leaveRoom(room.roomID).catch(err => {
			console.error("Failed to leave room", err)
			window.alert(`Failed to leave room: ${err}`)
		})
		closeModal()
	}
	const onClickShare = () => {
		openModal(modals.shareRoom(room))
	}
	const showLowPriority = client.store.preferences.pin_low_priority || client.store.preferences.mute_low_priority
	const showFavorite = client.store.preferences.pin_favorites
	return <div className="context-menu room-list-menu" style={style} ref={ref}>
		<MarkReadButton room={room} />
		<MuteButton roomID={room.roomID}/>
		{showFavorite ? <TagButton
			room={room}
			tag="m.favourite"
			taggedElem={<><UnfavoriteIcon /> Unfavorite</>}
			untaggedElem={<><FavoriteIcon /> Favorite</>}
		/> : null}
		{showLowPriority ? <TagButton
			room={room}
			tag="m.lowpriority"
			taggedElem={<><UnsetLowPriorityIcon /> Unset low priority</>}
			untaggedElem={<><SetLowPriorityIcon /> Set low priority</>}
		/> : null}
		<button onClick={onClickShare}><ShareIcon /> Share</button>
		<button onClick={openSettings}><SettingsIcon /> Settings</button>
		<button onClick={leaveRoom}><DoorOpenIcon /> Leave room</button>
	</div>
}

RoomMenu.height = 7 * 40
