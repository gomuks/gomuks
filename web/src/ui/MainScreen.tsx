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
import equal from "fast-deep-equal"
import { JSX, use, useEffect, useMemo, useReducer, useRef, useState } from "react"
import { SyncLoader } from "react-spinners"
import Client from "@/api/client.ts"
import { RoomListFilter, RoomStateStore } from "@/api/statestore"
import type { RoomID } from "@/api/types"
import { useEventAsState } from "@/util/eventdispatcher.ts"
import { ensureString, ensureStringArray, parseMatrixURI } from "@/util/validation.ts"
import ClientContext from "./ClientContext.ts"
import MainScreenContext, { MainScreenContextFields } from "./MainScreenContext.ts"
import StylePreferences from "./StylePreferences.tsx"
import Keybindings from "./keybindings.ts"
import { ModalContext, ModalWrapper, NestableModalContext } from "./modal"
import RightPanel, { RightPanelProps } from "./rightpanel/RightPanel.tsx"
import RoomList from "./roomlist/RoomList.tsx"
import RoomPreview, { RoomPreviewProps } from "./roomview/RoomPreview.tsx"
import RoomView from "./roomview/RoomView.tsx"
import { useResizeHandle } from "./util/useResizeHandle.tsx"
import "./MainScreen.css"

class ContextFields implements MainScreenContextFields {
	public keybindings: Keybindings
	private rightPanelStack: RightPanelProps[] = []

	constructor(
		private directSetRightPanel: (props: RightPanelProps | null) => void,
		private directSetActiveRoom: (room: RoomStateStore | RoomPreviewProps | null) => void,
		private directSetSpace: (space: RoomListFilter | null) => void,
		private client: Client,
	) {
		this.keybindings = new Keybindings(client.store, this)
		client.store.switchRoom = this.setActiveRoom
	}

	get currentRightPanel(): RightPanelProps | null {
		return this.rightPanelStack.length ? this.rightPanelStack[this.rightPanelStack.length - 1] : null
	}

	setRightPanel = (props: RightPanelProps | null, pushState = true) => {
		if ((props?.type !== "user") && !this.client.store.activeRoomID) {
			props = null
		}
		const isEqual = equal(this.currentRightPanel, props)
		if (isEqual && !pushState) {
			return
		}
		if (isEqual || props === null) {
			const length = this.rightPanelStack.length
			this.rightPanelStack = []
			this.directSetRightPanel(null)
			if (length && pushState) {
				history.go(-length)
			}
		} else {
			this.directSetRightPanel(props)
			for (let i = this.rightPanelStack.length - 1; i >= 0; i--) {
				if (equal(this.rightPanelStack[i], props)) {
					this.rightPanelStack = this.rightPanelStack.slice(0, i + 1)
					if (pushState) {
						history.go(i - this.rightPanelStack.length)
					}
					return
				}
			} // else:
			this.rightPanelStack.push(props)
			if (pushState) {
				history.pushState({ ...(history.state ?? {}), right_panel: props }, "")
			}
		}
	}

	setActiveRoom = (
		roomID: RoomID | null,
		previewMeta?: Partial<RoomPreviewProps>,
		toSpace?: RoomListFilter,
		pushState = true,
	) => {
		console.log("Switching to room", roomID)
		if (roomID) {
			const room = this.client.store.rooms.get(roomID)
			if (room) {
				this.#setActiveRoom(room, toSpace, pushState)
			} else {
				this.#setPreviewRoom(roomID, pushState, previewMeta)
			}
		} else {
			this.#closeActiveRoom(pushState)
		}
	}

	setSpace = (space: RoomListFilter | null, pushState = true) => {
		if (space === this.client.store.currentRoomListFilter) {
			return
		}
		console.log("Switching to space", space?.id)
		this.directSetSpace(space)
		this.client.store.currentRoomListFilter = space
		if (pushState) {
			if (this.client.store.activeRoomID && space) {
				const entry = this.client.store.roomListEntries.get(this.client.store.activeRoomID)
				if (entry && !space.include(entry)) {
					this.setActiveRoom(null)
				}
			}
			history.replaceState({ ...(history.state || {}), space_id: space?.id }, "")
		}
	}

	#setPreviewRoom(roomID: RoomID, pushState: boolean, meta?: Partial<RoomPreviewProps>) {
		const invite = this.client.store.inviteRooms.get(roomID)
		this.#closeActiveRoom(false)
		this.directSetActiveRoom({ roomID, ...(meta ?? {}), invite })
		this.client.store.activeRoomID = roomID
		this.client.store.activeRoomIsPreview = true
		if (pushState) {
			history.pushState({
				room_id: roomID,
				source_via: meta?.via,
				source_alias: meta?.alias,
				space_id: history.state?.space_id,
			}, "")
		}
	}

	#getWindowTitle(room?: RoomStateStore, name?: string) {
		if (!room) {
			return this.client.store.preferences.window_title
		}
		return room.preferences.room_window_title.replace("$room", name!)
	}

	#setActiveRoom(room: RoomStateStore, space: RoomListFilter | undefined | null, pushState: boolean) {
		window.activeRoom = room
		this.directSetActiveRoom(room)
		this.directSetRightPanel(null)
		if (!space && this.client.store.currentRoomListFilter) {
			const roomListEntry = this.client.store.roomListEntries.get(room.roomID)
			if (roomListEntry && !this.client.store.currentRoomListFilter.include(roomListEntry)) {
				space = this.client.store.findMatchingSpace(roomListEntry)
			}
		}
		if (space && space !== this.client.store.currentRoomListFilter) {
			console.log("Switching to space", space?.id)
			this.directSetSpace(space)
			this.client.store.currentRoomListFilter = space
		}
		this.rightPanelStack = []
		this.client.store.activeRoomID = room.roomID
		this.client.store.activeRoomIsPreview = false
		this.keybindings.activeRoom = room
		room.lastOpened = Date.now()
		if (!room.stateLoaded) {
			this.client.loadRoomState(room.roomID)
				.catch(err => console.error("Failed to load room state", err))
		}
		document
			.querySelector(`div.room-entry[data-room-id="${CSS.escape(room.roomID)}"]`)
			?.scrollIntoView({ block: "nearest" })
		if (pushState) {
			history.pushState({ room_id: room.roomID, space_id: space?.id ?? history.state?.space_id }, "")
		}
		let roomNameForTitle = room.meta.current.name
		if (roomNameForTitle && roomNameForTitle.length > 48) {
			roomNameForTitle = roomNameForTitle.slice(0, 45) + "…"
		}
		document.title = this.#getWindowTitle(room, roomNameForTitle)
	}

	#closeActiveRoom(pushState: boolean) {
		window.activeRoom = null
		this.directSetActiveRoom(null)
		this.directSetRightPanel(null)
		this.rightPanelStack = []
		this.client.store.activeRoomID = null
		this.client.store.activeRoomIsPreview = false
		this.keybindings.activeRoom = null
		if (pushState) {
			history.pushState({ space_id: history.state?.space_id }, "")
		}
		document.title = this.#getWindowTitle()
	}

	clickRoom = (evt: React.MouseEvent) => {
		const roomID = evt.currentTarget.getAttribute("data-room-id")
		if (roomID) {
			this.setActiveRoom(roomID)
		} else {
			console.warn("No room ID :(", evt.currentTarget)
		}
	}

	clickRightPanelOpener = (evt: React.MouseEvent) => {
		evt.preventDefault()
		const type = evt.currentTarget.getAttribute("data-target-panel")
		if (type === "pinned-messages" || type === "members" || type === "widgets") {
			this.setRightPanel({ type })
		} else if (type === "user") {
			this.setRightPanel({ type, userID: evt.currentTarget.getAttribute("data-target-user")! })
		} else {
			throw new Error(`Invalid right panel type ${type}`)
		}
	}

	clearActiveRoom = () => this.setActiveRoom(null)
	closeRightPanel = () => this.setRightPanel(null)
}

const SYNC_ERROR_HIDE_DELAY = 30 * 1000

const handleURLHash = (client: Client, context: MainScreenContextFields, hashOnly = false) => {
	if (!location.hash.startsWith("#/uri/")) {
		if (hashOnly) {
			return null
		}
		if (location.search) {
			const currentETag = (
				document.querySelector("meta[name=gomuks-frontend-etag]") as HTMLMetaElement
			)?.content
			const newURL = new URL(location.href)
			const updateTo = newURL.searchParams.get("updateTo")
			if (updateTo === currentETag) {
				console.info("Update to etag", updateTo, "successful")
			} else {
				console.warn("Update to etag", updateTo, "failed, got", currentETag)
			}
			const state = JSON.parse(newURL.searchParams.get("state") || "{}")
			newURL.search = ""
			// Set an extra empty state to ensure back button goes to room list instead of reloading the page.
			history.replaceState({}, "", newURL.toString())
			history.pushState(state, "")
			return state
		}
		return history.state
	}

	const decodedURI = decodeURIComponent(location.hash.slice("#/uri/".length))
	const uri = parseMatrixURI(decodedURI)
	if (!uri) {
		console.error("Invalid matrix URI", decodedURI)
		return hashOnly ? null : history.state
	}
	console.log("Handling URI", uri)
	const newURL = new URL(location.href)
	newURL.hash = ""
	newURL.search = ""
	if (uri.identifier.startsWith("@")) {
		const newState = {
			right_panel: {
				type: "user",
				userID: uri.identifier,
			},
		}
		history.replaceState(newState, "", newURL.toString())
		return newState
	} else if (uri.identifier.startsWith("!")) {
		const newState = { room_id: uri.identifier, source_via: uri.params.getAll("via") }
		history.replaceState(newState, "", newURL.toString())
		return newState
	} else if (uri.identifier.startsWith("#")) {
		history.replaceState(history.state, "", newURL.toString())
		// TODO loading indicator or something for this?
		client.rpc.resolveAlias(uri.identifier).then(
			res => {
				context.setActiveRoom(res.room_id, {
					alias: uri.identifier,
					via: res.servers.slice(0, 3),
				})
			},
			err => window.alert(`Failed to resolve room alias ${uri.identifier}: ${err}`),
		)
		return null
	} else {
		console.error("Invalid matrix URI", uri)
		history.replaceState(history.state, "", newURL.toString())
	}
	return hashOnly ? null : history.state
}

type ActiveRoomType = [RoomStateStore | RoomPreviewProps | null, RoomStateStore | RoomPreviewProps | null]

const activeRoomReducer = (
	prev: ActiveRoomType,
	active: RoomStateStore | RoomPreviewProps | "clear-animation" | null,
): ActiveRoomType => {
	if (active === "clear-animation") {
		return prev[1] === null ? [null, null] : prev
	} else if (window.innerWidth > 720 || window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
		return [null, active]
	} else {
		return [prev[1], active]
	}
}

const MainScreen = () => {
	const [[prevActiveRoom, activeRoom], directSetActiveRoom] = useReducer(activeRoomReducer, [null, null])
	const [space, directSetSpace] = useState<RoomListFilter | null>(null)
	const skipNextTransitionRef = useRef(false)
	const [rightPanel, directSetRightPanel] = useState<RightPanelProps | null>(null)
	const client = use(ClientContext)!
	const syncStatus = useEventAsState(client.syncStatus)
	const context = useMemo(
		() => new ContextFields(directSetRightPanel, directSetActiveRoom, directSetSpace, client),
		[client],
	)
	useEffect(() => {
		window.mainScreenContext = context
		const listener = (evt: Pick<PopStateEvent, "state" | "hasUAVisualTransition">) => {
			skipNextTransitionRef.current = evt.hasUAVisualTransition
			const roomID = evt.state?.room_id ?? null
			const spaceID = evt.state?.space_id ?? undefined
			if (spaceID !== client.store.currentRoomListFilter?.id) {
				context.setSpace(client.store.getSpaceByID(spaceID), false)
			}
			if (roomID !== client.store.activeRoomID) {
				context.setActiveRoom(roomID, {
					alias: ensureString(evt.state?.source_alias) || undefined,
					via: ensureStringArray(evt.state?.source_via),
				}, undefined, false)
			}
			context.setRightPanel(evt.state?.right_panel ?? null, false)
		}
		const hashListener = () => {
			const state = handleURLHash(client, context, true)
			if (state !== null) {
				listener({ state, hasUAVisualTransition: false })
			}
		}
		window.addEventListener("hashchange", hashListener)
		window.addEventListener("popstate", listener)
		const initHandle = () => {
			const state = handleURLHash(client, context)
			listener({ state } as PopStateEvent)
		}
		let cancel = () => {}
		if (client.initComplete.current) {
			initHandle()
		} else {
			cancel = client.initComplete.once(initHandle)
		}
		return () => {
			window.removeEventListener("popstate", listener)
			window.removeEventListener("hashchange", hashListener)
			cancel()
		}
	}, [context, client])
	useEffect(() => context.keybindings.listen(), [context])
	const [roomListWidth, resizeHandle1] = useResizeHandle(
		350, 96, Math.min(900, window.innerWidth * 0.4),
		"roomListWidth", { className: "room-list-resizer" },
	)
	const [rightPanelWidth, resizeHandle2] = useResizeHandle(
		300, 100, Math.min(900, window.innerWidth * 0.4),
		"rightPanelWidth", { className: "right-panel-resizer", inverted: true },
	)
	const extraStyle = {
		["--room-list-width" as string]: `${roomListWidth}px`,
		["--right-panel-width" as string]: `${rightPanelWidth}px`,
	}
	if (skipNextTransitionRef.current) {
		extraStyle["transition"] = "none"
		skipNextTransitionRef.current = false
	}
	const classNames = ["matrix-main"]
	if (activeRoom) {
		classNames.push("room-selected")
	}
	if (rightPanel) {
		classNames.push("right-panel-open")
	}
	let syncLoader: JSX.Element | null = null
	if (syncStatus.type === "waiting") {
		syncLoader = <div className="sync-status waiting">
			<SyncLoader color="var(--primary-color)"/>
			Waiting for first sync...
		</div>
	} else if (
		syncStatus.type === "erroring"
		&& (syncStatus.error_count > 2 || (syncStatus.last_sync ?? 0) + SYNC_ERROR_HIDE_DELAY < Date.now())
	) {
		syncLoader = <div className="sync-status errored" title={syncStatus.error}>
			<SyncLoader color="var(--error-color)"/>
			Sync is failing
		</div>
	} else if (syncStatus.type === "permanently-failed") {
		syncLoader = <div className="sync-status errored" title={syncStatus.error}>
			Sync failed permanently
		</div>
	}
	const activeRealRoom = activeRoom instanceof RoomStateStore ? activeRoom : null
	const renderedRoom = activeRoom ?? prevActiveRoom
	useEffect(() => {
		if (prevActiveRoom !== null && activeRoom === null) {
			// Note: this timeout must match the one in MainScreen.css
			const timeout = setTimeout(() => directSetActiveRoom("clear-animation"), 300)
			return () => clearTimeout(timeout)
		}
	}, [activeRoom, prevActiveRoom])
	const mainContent = <main className={classNames.join(" ")} style={extraStyle}>
		<RoomList activeRoomID={activeRoom?.roomID ?? null} space={space}/>
		{resizeHandle1}
		{renderedRoom
			? renderedRoom instanceof RoomStateStore
				? <RoomView
					key={renderedRoom.roomID}
					room={renderedRoom}
					rightPanel={rightPanel}
					rightPanelResizeHandle={resizeHandle2}
				/>
				: <RoomPreview {...renderedRoom} />
			: rightPanel && <>
				<div className="room-view placeholder"/>
				{resizeHandle2}
				{rightPanel && <RightPanel {...rightPanel}/>}
			</>}
	</main>
	return <MainScreenContext value={context}>
		<ModalWrapper ContextType={ModalContext} historyStateKey="modal">
			<ModalWrapper ContextType={NestableModalContext} historyStateKey="nestable_modal">
				<StylePreferences client={client} activeRoom={activeRealRoom}/>
				{mainContent}
				{syncLoader}
			</ModalWrapper>
		</ModalWrapper>
	</MainScreenContext>
}

export default MainScreen
