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
import { getAvatarThumbnailURL } from "@/api/media.ts"
import { Preferences, getLocalStoragePreferences, getPreferenceProxy } from "@/api/types/preferences"
import { CustomEmojiPack } from "@/util/emoji"
import { NonNullCachedEventDispatcher } from "@/util/eventdispatcher.ts"
import { focused } from "@/util/focus.ts"
import toSearchableString from "@/util/searchablestring.ts"
import { playSound } from "@/util/sound.ts"
import Subscribable, { MultiSubscribable, NoDataSubscribable } from "@/util/subscribable.ts"
import { getDisplayname } from "@/util/validation.ts"
import {
	ContentURI,
	DBRoom,
	EventRowID,
	EventsDecryptedData,
	ImagePackRooms,
	MemDBEvent,
	RoomID,
	RoomStateGUID,
	SendCompleteData,
	SyncCompleteData,
	SyncRoom,
	SyncToDevice,
	TypingEventData,
	UnknownEventContent,
	UserID,
} from "../types"
import StateCache from "./cache.ts"
import { InvitedRoomStore } from "./invitedroom.ts"
import { RoomStateStore } from "./room.ts"
import {
	DirectChatSpace,
	HomeSpace,
	RoomListFilter,
	Space,
	SpaceEdgeStore,
	SpaceOrphansSpace,
	UnreadsSpace,
} from "./space.ts"

export interface RoomListEntry {
	room_id: RoomID
	dm_user_id?: UserID
	sorting_timestamp: number
	preview_event?: MemDBEvent
	name: string
	search_name: string
	avatar?: ContentURI
	unread_messages: number
	unread_notifications: number
	unread_highlights: number
	marked_unread: boolean
	is_invite?: boolean
	favorite_order?: number
	low_priority?: boolean
}

function alphabeticalSort(r1: RoomListEntry, r2: RoomListEntry): number {
	return r2.name.localeCompare(r1.name)
}

function timestampSort(r1: RoomListEntry, r2: RoomListEntry): number {
	return r1.sorting_timestamp - r2.sorting_timestamp
}

function favoriteSort(r1: RoomListEntry, r2: RoomListEntry): number {
	if (r1.favorite_order !== undefined && r2.favorite_order !== undefined) {
		return r2.favorite_order - r1.favorite_order
	} else if (r1.favorite_order !== undefined) {
		return 1
	} else if (r2.favorite_order !== undefined) {
		return -1
	} else {
		return 0
	}
}

function lowPrioritySort(r1: RoomListEntry, r2: RoomListEntry): number {
	if (r1.low_priority && !r2.low_priority) {
		return -1
	} else if (!r1.low_priority && r2.low_priority) {
		return 1
	} else {
		return 0
	}
}

type SortFunc = (r1: RoomListEntry, r2: RoomListEntry) => number

function chainedSort(f1: SortFunc, f2: SortFunc): SortFunc {
	return (r1, r2) => {
		const res = f1(r1, r2)
		if (res !== 0) {
			return res
		}
		return f2(r1, r2)
	}
}

export interface GCSettings {
	interval: number,
	lastOpenedCutoff: number,
}

export interface WidgetListener {
	onTimelineEvent(evt: MemDBEvent): void
	onStateEvent(evt: MemDBEvent): void
	onToDeviceEvent(evt: SyncToDevice): void
	onRoomChange(roomID: RoomID | null): void
}

window.gcSettings ??= {
	// Run garbage collection every 15 minutes.
	interval: 15 * 60 * 1000,
	// Run garbage collection to rooms not opened in the past 30 minutes.
	lastOpenedCutoff: 30 * 60 * 1000,
}

export class StateStore {
	userID: UserID = ""
	readonly rooms: Map<RoomID, RoomStateStore> = new Map()
	readonly inviteRooms: Map<RoomID, InvitedRoomStore> = new Map()
	readonly roomList = new NonNullCachedEventDispatcher<RoomListEntry[]>([])
	readonly roomListEntries = new Map<RoomID, RoomListEntry>()
	readonly topLevelSpaces = new NonNullCachedEventDispatcher<RoomID[]>([])
	readonly spaceEdges: Map<RoomID, SpaceEdgeStore> = new Map()
	readonly spaceOrphans = new SpaceOrphansSpace(this)
	readonly homeSpace = new HomeSpace()
	readonly directChatsSpace = new DirectChatSpace()
	readonly unreadsSpace = new UnreadsSpace(this)
	readonly pseudoSpaces = [
		this.spaceOrphans,
		this.directChatsSpace,
		this.unreadsSpace,
	] as const
	currentRoomListQuery: string = ""
	currentRoomListFilter: RoomListFilter | null = null
	readonly accountData: Map<string, UnknownEventContent> = new Map()
	readonly accountDataSubs = new MultiSubscribable()
	readonly emojiRoomsSub = new Subscribable()
	readonly preferences = getPreferenceProxy(this)
	#frequentlyUsedEmoji: Map<string, number> | null = null
	#emojiPackKeys: RoomStateGUID[] | null = null
	#watchedRoomEmojiPacks: Record<string, CustomEmojiPack> | null = null
	readonly preferenceSub = new NoDataSubscribable()
	readonly localPreferenceCache: Preferences = getLocalStoragePreferences("global_prefs", this.preferenceSub.notify)
	serverPreferenceCache: Preferences = {}
	switchRoom?: (roomID: RoomID | null) => void
	#activeRoomID: RoomID | null = null
	activeRoomIsPreview: boolean = false
	imageAuthToken?: string
	readonly widgetListeners: Set<WidgetListener> = new Set()
	stateCache?: StateCache
	tmpStateCache?: StateCache
	stateCacheStatus: string = "disabled"
	serverTimestamp?: number

	get activeRoomID(): RoomID | null {
		return this.#activeRoomID
	}

	set activeRoomID(roomID: RoomID | null) {
		this.#activeRoomID = roomID
		this.widgetListeners.forEach(listener => listener.onRoomChange(roomID))
	}

	#roomListFilterFunc = (entry: RoomListEntry) => {
		if (this.currentRoomListQuery && !entry.search_name.includes(this.currentRoomListQuery)) {
			return false
		} else if (this.currentRoomListFilter && !this.currentRoomListFilter.include(entry)) {
			return false
		}
		return true
	}

	get anyStateCache(): StateCache | undefined {
		return this.stateCache ?? this.tmpStateCache
	}

	closeCache() {
		this.anyStateCache?.close()
		this.stateCache = undefined
		this.tmpStateCache = undefined
		this.stateCacheStatus = "closed"
	}

	deleteCache() {
		this.closeCache()
		this.stateCacheStatus = "deleted"
		return StateCache.delete()
	}

	loadCache() {
		const cache = new StateCache()
		this.tmpStateCache = cache
		return cache.load().then(data => {
			if (!data) {
				console.info("No state cache found")
				this.stateCacheStatus = "no data"
			} else {
				console.info(
					"Applying state cache from", new Date(data.server_timestamp!),
					"with", Object.keys(data.rooms ?? {}).length, "rooms",
				)
				this.applySync(data)
				this.stateCacheStatus = "enabled"
			}
			this.stateCache = cache
			this.tmpStateCache = undefined
		}, err => {
			console.error("Failed to load state cache", err)
			this.stateCacheStatus = `failed: ${err}`
			cache.close()
		})
	}

	getSpaceByID(spaceID: string | undefined): RoomListFilter | null {
		if (!spaceID) {
			return null
		}
		const realSpace = this.spaceEdges.get(spaceID)
		if (realSpace) {
			return realSpace
		}
		for (const pseudoSpace of this.pseudoSpaces) {
			if (pseudoSpace.id === spaceID) {
				return pseudoSpace
			}
		}
		console.warn("Failed to find space", spaceID)
		return null
	}

	findMatchingSpace(room: RoomListEntry): Space | null {
		if (this.spaceOrphans.include(room)) {
			return this.spaceOrphans
		}
		for (const spaceID of this.topLevelSpaces.current) {
			const space = this.spaceEdges.get(spaceID)
			if (space?.include(room)) {
				return space
			}
		}
		if (this.directChatsSpace.include(room)) {
			return this.directChatsSpace
		}
		return null
	}

	get roomListFilterFunc(): ((entry: RoomListEntry) => boolean) | null {
		if (!this.currentRoomListFilter && !this.currentRoomListQuery) {
			return null
		}
		return this.#roomListFilterFunc
	}

	getFilteredRoomList(): RoomListEntry[] {
		const fn = this.roomListFilterFunc
		return fn ? this.roomList.current.filter(fn) : this.roomList.current
	}

	#showTypeInRoomList(meta: DBRoom): boolean {
		const cc = meta.creation_content
		switch (cc?.type ?? "") {
		default:
			// The room is not a normal room
			return false
		case "":
		case "support.feline.policy.lists.msc.v1":
		case "org.matrix.msc3417.call":
		case "fi.mau.msc2545.image_pack":
			return true
		}
	}

	#isTombstoned(meta: DBRoom): boolean {
		const replacementRoom = meta.tombstone?.replacement_room
		if (
			replacementRoom
			&& this.rooms.get(replacementRoom)?.meta.current.creation_content?.predecessor?.room_id
			=== meta.room_id
		) {
			// The room is tombstoned and the replacement room is valid.
			return true
		}
		// Otherwise don't hide the room.
		return false
	}

	#roomListEntryChanged(entry: SyncRoom, oldEntry: RoomStateStore): boolean {
		if (!entry.meta) {
			return false
		}
		return entry.meta.sorting_timestamp !== oldEntry.meta.current.sorting_timestamp ||
			entry.meta.unread_messages !== oldEntry.meta.current.unread_messages ||
			entry.meta.unread_notifications !== oldEntry.meta.current.unread_notifications ||
			entry.meta.unread_highlights !== oldEntry.meta.current.unread_highlights ||
			entry.meta.marked_unread !== oldEntry.meta.current.marked_unread ||
			entry.meta.preview_event_rowid !== oldEntry.meta.current.preview_event_rowid ||
			entry.meta.name !== oldEntry.meta.current.name ||
			entry.meta.avatar !== oldEntry.meta.current.avatar ||
			entry.meta.dm_user_id !== oldEntry.meta.current.dm_user_id ||
			(entry.events ?? []).findIndex(evt => evt.rowid === entry.meta!.preview_event_rowid) !== -1 ||
			(!!entry.account_data && "m.tag" in entry.account_data)
	}

	#makeRoomListEntry(entry: SyncRoom, room?: RoomStateStore): RoomListEntry | null {
		if (!entry.meta) {
			return null
		}
		if (!room) {
			room = this.rooms.get(entry.meta.room_id)
		}
		const isTombstoned = this.#isTombstoned(entry.meta)
		const showInRoomList = this.#showTypeInRoomList(entry.meta)
		const hidden = isTombstoned || !showInRoomList
		if (room) {
			room.tombstoned = isTombstoned
			room.hidden = hidden
		}
		if (hidden) {
			return null
		}
		const preview_event = room?.eventsByRowID.get(entry.meta.preview_event_rowid)
		const name = entry.meta.name ?? "Unnamed room"
		const tags = room?.accountData.get("m.tag")?.tags
		const favoriteTag = tags?.["m.favourite"]
		const lowPriority = !!tags?.["m.lowpriority"]
		return {
			room_id: entry.meta.room_id,
			dm_user_id: entry.meta.dm_user_id,
			sorting_timestamp: entry.meta.sorting_timestamp,
			preview_event,
			name,
			search_name: toSearchableString(name),
			avatar: entry.meta.avatar,
			unread_messages: lowPriority && this.preferences.mute_low_priority ? 0 : entry.meta.unread_messages,
			unread_notifications: entry.meta.unread_notifications,
			unread_highlights: entry.meta.unread_highlights,
			marked_unread: entry.meta.marked_unread,
			favorite_order: favoriteTag ? (+favoriteTag.order || 0) : undefined,
			low_priority: lowPriority,
		}
	}

	#applyUnreadModification(meta: RoomListEntry | null, oldMeta: RoomListEntry | undefined | null) {
		const someMeta = meta ?? oldMeta
		if (!someMeta) {
			return
		}
		this.homeSpace.applyUnreads(meta, oldMeta)
		if (this.directChatsSpace.include(someMeta)) {
			this.directChatsSpace.applyUnreads(meta, oldMeta)
		} else if (oldMeta && this.directChatsSpace.include(oldMeta)) {
			this.directChatsSpace.applyUnreads(null, oldMeta)
		}
		if (this.spaceOrphans.include(someMeta)) {
			this.spaceOrphans.applyUnreads(meta, oldMeta)
			return
		} else if (oldMeta && this.spaceOrphans.include(oldMeta)) {
			this.spaceOrphans.applyUnreads(null, oldMeta)
		}
		for (const space of this.spaceEdges.values()) {
			if (space.include(someMeta)) {
				space.applyUnreads(meta, oldMeta)
			}
		}
	}

	applySync(sync: SyncCompleteData) {
		let prevActiveRoom: RoomID | null = null
		if (sync.clear_state && this.rooms.size > 0) {
			console.info("Clearing state store as sync told to reset and there are rooms in the store")
			prevActiveRoom = this.activeRoomID
			this.clear()
		} else if (sync.catchup) {
			console.info("Received catchup sync, doing garbage collection on all rooms")
			for (const room of this.rooms.values()) {
				room.doGarbageCollection()
			}
		}
		const resyncRoomList = this.roomList.current.length === 0
		const changedRoomListEntries = new Map<RoomID, RoomListEntry | null>()
		if (sync.to_device?.length && this.widgetListeners.size > 0) {
			for (const listener of this.widgetListeners) {
				sync.to_device.forEach(listener.onToDeviceEvent)
			}
		}
		for (const data of sync.invited_rooms ?? []) {
			const room = new InvitedRoomStore(data, this)
			this.inviteRooms.set(room.room_id, room)
			if (!resyncRoomList) {
				changedRoomListEntries.set(room.room_id, room)
				this.#applyUnreadModification(room, this.roomListEntries.get(room.room_id))
				this.roomListEntries.set(room.room_id, room)
			}
			if (this.activeRoomID === room.room_id) {
				this.switchRoom?.(room.room_id)
			}
			this.stateCache?.setInvitedRoom(data)
		}
		const hasInvites = this.inviteRooms.size > 0
		for (const [roomID, data] of Object.entries(sync.rooms ?? {})) {
			let isNewRoom = false
			let room = this.rooms.get(roomID)
			if (!room) {
				if (!data.meta) {
					console.warn("Received room sync data without meta for unknown room", roomID)
					continue
				}
				room = new RoomStateStore(data.meta, this)
				this.rooms.set(roomID, room)
				if (hasInvites) {
					this.inviteRooms.delete(roomID)
				}
				isNewRoom = true
			}
			const roomListEntryChanged = !resyncRoomList && (isNewRoom || this.#roomListEntryChanged(data, room))
			room.applySync(data, isNewRoom)
			if (roomListEntryChanged) {
				const entry = this.#makeRoomListEntry(data, room)
				changedRoomListEntries.set(roomID, entry)
				this.#applyUnreadModification(entry, this.roomListEntries.get(roomID))
				if (entry) {
					this.roomListEntries.set(roomID, entry)
				} else {
					this.roomListEntries.delete(roomID)
				}
			}
			if (!resyncRoomList) {
				// When we join a valid replacement room, hide the tombstoned room.
				const predecessorID = data.meta?.creation_content?.predecessor?.room_id
				if (isNewRoom && typeof predecessorID === "string") {
					const predecessorRoom = this.rooms.get(predecessorID)
					if (predecessorRoom?.meta.current.tombstone?.replacement_room === roomID) {
						changedRoomListEntries.set(predecessorID, null)
						this.#applyUnreadModification(null, this.roomListEntries.get(predecessorID))
						predecessorRoom.tombstoned = true
						predecessorRoom.hidden = true
					}
				}
			}

			if (
				window.Notification?.permission === "granted"
				&& !focused.current
				&& data.notifications
				&& !this.localPreferenceCache.web_push
			) {
				for (const notification of data.notifications) {
					this.showNotification(room, notification.event_rowid, notification.sound)
				}
			}
			if (this.activeRoomID === roomID && this.activeRoomIsPreview) {
				this.switchRoom?.(roomID)
			}
		}
		for (const ad of Object.values(sync.account_data ?? {})) {
			if (ad.type === "io.element.recent_emoji") {
				this.#frequentlyUsedEmoji = null
			} else if (ad.type === "fi.mau.gomuks.preferences") {
				this.serverPreferenceCache = ad.content
				this.preferenceSub.notify()
			}
			this.accountData.set(ad.type, ad.content)
			this.accountDataSubs.notify(ad.type)
			this.stateCache?.setAccountData(ad)
		}
		for (const roomID of sync.left_rooms ?? []) {
			if (this.activeRoomID === roomID) {
				this.switchRoom?.(null)
			}
			this.rooms.delete(roomID)
			changedRoomListEntries.set(roomID, null)
			this.#applyUnreadModification(null, this.roomListEntries.get(roomID))
			this.stateCache?.deleteRoom(roomID)
		}

		let sortFunc: SortFunc = timestampSort
		if (this.preferences.alphabetical_order) {
			sortFunc = alphabeticalSort
		}
		if (this.preferences.pin_favorites) {
			sortFunc = chainedSort(favoriteSort, sortFunc)
		}
		if (this.preferences.pin_low_priority) {
			sortFunc = chainedSort(lowPrioritySort, sortFunc)
		}

		if (sync.space_edges) {
			// Ensure all space stores exist first
			for (const spaceID of Object.keys(sync.space_edges)) {
				this.getSpaceStore(spaceID, true)
			}
			for (const [spaceID, children] of Object.entries(sync.space_edges ?? {})) {
				this.getSpaceStore(spaceID, true).children = children
				this.stateCache?.setSpaceEdges(spaceID, children)
			}
		}
		if (sync.top_level_spaces) {
			this.topLevelSpaces.emit(sync.top_level_spaces)
			this.spaceOrphans.children = sync.top_level_spaces.map(child_id => ({ child_id }))
			this.stateCache?.setTopLevelSpaces(sync.top_level_spaces)
		}

		let updatedRoomList: RoomListEntry[] | undefined
		if (resyncRoomList) {
			updatedRoomList = this.inviteRooms.values().toArray()
			updatedRoomList = updatedRoomList.concat(Object.values(sync.rooms ?? {})
				.map(entry => this.#makeRoomListEntry(entry))
				.filter(entry => entry !== null))
			updatedRoomList.sort(sortFunc)
			for (const entry of updatedRoomList) {
				this.#applyUnreadModification(entry, undefined)
				this.roomListEntries.set(entry.room_id, entry)
			}
		} else if (changedRoomListEntries.size > 0) {
			updatedRoomList = this.roomList.current.filter(entry => !changedRoomListEntries.has(entry.room_id))
			for (const entry of changedRoomListEntries.values()) {
				if (!entry) {
					continue
				}
				if (sortFunc === timestampSort) {
					if (updatedRoomList.length === 0 || entry.sorting_timestamp >=
						updatedRoomList[updatedRoomList.length - 1].sorting_timestamp) {
						updatedRoomList.push(entry)
					} else if (entry.sorting_timestamp <= 0 ||
						entry.sorting_timestamp < updatedRoomList[0]?.sorting_timestamp) {
						updatedRoomList.unshift(entry)
					} else {
						const indexToPushAt = updatedRoomList.findLastIndex(val =>
							val.sorting_timestamp <= entry.sorting_timestamp)
						updatedRoomList.splice(indexToPushAt + 1, 0, entry)
					}
				} else {
					const indexToPushAt = updatedRoomList.findLastIndex(val => sortFunc(val, entry) <= 0)
					updatedRoomList.splice(indexToPushAt + 1, 0, entry)
				}
			}
		}
		if (updatedRoomList) {
			this.roomList.emit(updatedRoomList)
		}
		if (prevActiveRoom) {
			// TODO this will fail if the room is not in the top 100 recent rooms
			this.switchRoom?.(prevActiveRoom)
		}
		if (sync.server_timestamp) {
			this.stateCache?.setServerTimestamp(sync.server_timestamp)
			this.serverTimestamp = sync.server_timestamp
		}
	}

	invalidateEmojiPackKeyCache() {
		this.#emojiPackKeys = null
		this.#watchedRoomEmojiPacks = null
	}

	invalidateEmojiPacksCache() {
		this.#watchedRoomEmojiPacks = null
		this.emojiRoomsSub.notify()
	}

	getEmojiPackKeys(bothKeys: boolean = true): RoomStateGUID[] {
		if (this.#emojiPackKeys === null) {
			const emoteRooms = (
				this.accountData.get("m.image_pack.rooms")
				?? this.accountData.get("im.ponies.emote_rooms")
			) as ImagePackRooms | undefined
			try {
				const emojiPacks: RoomStateGUID[] = []
				for (const [roomID, packs] of Object.entries(emoteRooms?.rooms ?? {})) {
					for (const pack of Object.keys(packs)) {
						if (bothKeys) {
							emojiPacks.push({ room_id: roomID, type: "im.ponies.room_emotes", state_key: pack })
						}
						emojiPacks.push({ room_id: roomID, type: "m.room.image_pack", state_key: pack })
					}
				}
				this.#emojiPackKeys = emojiPacks
			} catch (err) {
				console.warn("Failed to parse emote rooms data", err, emoteRooms)
				this.#emojiPackKeys = []
			}
		}
		return this.#emojiPackKeys
	}

	getRoomEmojiPacks() {
		if (this.#watchedRoomEmojiPacks === null) {
			this.#watchedRoomEmojiPacks = Object.fromEntries(
				this.getEmojiPackKeys(false)
					.map(key => {
						const room = this.rooms.get(key.room_id)
						if (!room) {
							console.warn("Failed to find room for emoji pack", key)
							return null
						}
						const pack = room.getEmojiPack(key.state_key)
						if (!pack) {
							console.warn("Failed to find pack", key)
							return null
						}
						return [pack.id, pack]
					})
					.filter(pack => !!pack),
			)
		}
		return this.#watchedRoomEmojiPacks ?? {}
	}

	getSpaceStore(spaceID: RoomID, force: true): SpaceEdgeStore
	getSpaceStore(spaceID: RoomID): SpaceEdgeStore | null
	getSpaceStore(spaceID: RoomID, force?: true): SpaceEdgeStore | null {
		let store = this.spaceEdges.get(spaceID)
		if (!store) {
			if (!force && this.rooms.get(spaceID)?.meta.current.creation_content?.type !== "m.space") {
				return null
			}
			store = new SpaceEdgeStore(spaceID, this)
			this.spaceEdges.set(spaceID, store)
		}
		return store
	}

	get frequentlyUsedEmoji(): Map<string, number> {
		if (this.#frequentlyUsedEmoji === null) {
			const emojiData = this.accountData.get("io.element.recent_emoji")
			try {
				const recentList = emojiData?.recent_emoji as [string, number][] | undefined
				this.#frequentlyUsedEmoji = new Map(recentList?.toSorted(
					([, count1], [, count2]) => count2 - count1,
				))
			} catch (err) {
				console.warn("Failed to parse recent emoji data", err, emojiData?.recent_emoji)
				this.#frequentlyUsedEmoji = new Map()
			}
		}
		return this.#frequentlyUsedEmoji
	}

	showNotification(room: RoomStateStore, rowid: EventRowID, sound: boolean) {
		const evt = room.eventsByRowID.get(rowid)
		if (!evt || typeof evt.content.body !== "string") {
			return
		}
		if (sound) {
			playSound(room.preferences.notification_sound, room.preferences.notification_sound_volume)
		}
		if (window.gomuksDesktop?.getDisableNotifications()) {
			// Notifications are sent by the main process
			return
		}
		let body = evt.content.body
		if (body.length > 400) {
			body = body.slice(0, 350) + " […]"
		}
		const memberEvt = room.getStateEvent("m.room.member", evt.sender)
		const icon = `${getAvatarThumbnailURL(evt.sender, memberEvt?.content)}&image_auth=${this.imageAuthToken}`
		const roomName = room.meta.current.name ?? "Unnamed room"
		const senderName = getDisplayname(evt.sender, memberEvt?.content)
		const title = senderName === roomName ? senderName : `${senderName} (${roomName})`
		const notif = new Notification(title, {
			body,
			icon,
			badge: "gomuks.png",
			// timestamp: evt.timestamp,
			// image: ...,
			silent: !sound,
			tag: rowid.toString(),
		})
		room.openNotifications.set(rowid, notif)
		notif.onclose = () => room.openNotifications.delete(rowid)
		notif.onclick = () => this.onClickNotification(room.roomID)
	}

	onClickNotification(roomID: RoomID) {
		if (this.switchRoom) {
			this.switchRoom(roomID)
		}
	}

	applySendComplete(data: SendCompleteData) {
		const room = this.rooms.get(data.event.room_id)
		if (!room) {
			// TODO log or something?
			return
		}
		room.applySendComplete(data.event)
	}

	applyDecrypted(decrypted: EventsDecryptedData) {
		const room = this.rooms.get(decrypted.room_id)
		if (!room) {
			// TODO log or something?
			return
		}
		room.applyDecrypted(decrypted)
		if (decrypted.preview_event_rowid) {
			const idx = this.roomList.current.findIndex(entry => entry.room_id === decrypted.room_id)
			if (idx !== -1) {
				const updatedRoomList = [...this.roomList.current]
				updatedRoomList[idx] = {
					...updatedRoomList[idx],
					preview_event: room.eventsByRowID.get(decrypted.preview_event_rowid),
				}
				this.roomList.emit(updatedRoomList)
			}
		}
	}

	applyTyping(typing: TypingEventData) {
		const room = this.rooms.get(typing.room_id)
		if (!room) {
			// TODO log or something?
			return
		}
		room.applyTyping(typing.user_ids)
	}

	clearTyping() {
		for (const room of this.rooms.values()) {
			room.clearTyping()
		}
	}

	doGarbageCollection() {
		const maxLastOpened = Date.now() - window.gcSettings.lastOpenedCutoff
		let deletedEvents = 0
		let deletedState = 0
		for (const room of this.rooms.values()) {
			if (room.roomID === this.activeRoomID || room.lastOpened > maxLastOpened || room.activeWidgets.size) {
				continue
			}
			const [de, ds] = room.doGarbageCollection()
			deletedEvents += de
			deletedState += ds
		}
		return { deletedEvents, deletedState } as const
	}

	clear() {
		this.rooms.clear()
		this.inviteRooms.clear()
		this.spaceEdges.clear()
		this.homeSpace.clearUnreads()
		this.pseudoSpaces.forEach(space => space.clearUnreads())
		this.roomList.emit([])
		this.topLevelSpaces.emit([])
		this.accountData.clear()
		this.currentRoomListQuery = ""
		this.currentRoomListFilter = null
		this.#frequentlyUsedEmoji = null
		this.#emojiPackKeys = null
		this.#watchedRoomEmojiPacks = null
		this.serverPreferenceCache = {}
		this.activeRoomID = null
	}
}
