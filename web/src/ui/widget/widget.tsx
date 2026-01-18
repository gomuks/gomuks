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
import {
	ClientWidgetApi,
	IModalWidgetCloseRequest,
	IStickyActionRequest,
	IWidget,
	Widget as WrappedWidget,
} from "matrix-widget-api"
import { memo } from "react"
import type Client from "@/api/client"
import type { RoomStateStore, WidgetListener } from "@/api/statestore"
import type { MemDBEvent, RoomID, SyncToDevice } from "@/api/types"
import { getDisplayname } from "@/util/validation"
import PermissionPrompt from "./PermissionPrompt"
import { memDBEventToIRoomEvent } from "./util"
import GomuksWidgetDriver from "./widgetDriver"
import "./Widget.css"

export interface WidgetProps {
	info: IWidget
	room: RoomStateStore
	client: Client
	noPermissionPrompt?: boolean
	onClose?: () => void
}

// TODO remove this after widgets start using a parameter for it
const addLegacyParams = (url: string, widgetID: string) => {
	const urlObj = new URL(url)
	urlObj.searchParams.set("parentUrl", window.location.href)
	urlObj.searchParams.set("widgetId", widgetID)
	return urlObj.toString()
}

class WidgetListenerImpl implements WidgetListener {
	constructor(private api: ClientWidgetApi) {}

	onTimelineEvent = (evt: MemDBEvent) => {
		this.api.feedEvent(memDBEventToIRoomEvent(evt))
			.catch(err => console.error("Failed to feed event", memDBEventToIRoomEvent(evt), err))
	}

	onStateEvent = (evt: MemDBEvent) => {
		this.api.feedStateUpdate(memDBEventToIRoomEvent(evt))
			.catch(err => console.error("Failed to feed state update", memDBEventToIRoomEvent(evt), err))
	}

	onRoomChange = (roomID: RoomID | null) => {
		this.api.setViewedRoomId(roomID)
	}

	onToDeviceEvent = (evt: SyncToDevice) => {
		this.api.feedToDevice({
			sender: evt.sender,
			type: evt.type,
			content: evt.content,
		}, evt.encrypted).catch(err => console.error("Failed to feed to-device event", evt, err))
	}
}

const openPermissionPrompt = (requested: Set<string>): Promise<Set<string>> => {
	return new Promise(resolve => {
		window.openModal({
			content: <PermissionPrompt
				capabilities={requested}
				onConfirm={resolve}
			/>,
			dimmed: true,
			boxed: true,
			noDismiss: true,
			innerBoxClass: "permission-prompt",
		})
	})
}

const noopPermissions = (requested: Set<string>): Promise<Set<string>> =>  Promise.resolve(requested)

interface ExistingWidget {
	onMount: (into: HTMLDivElement, onClose?: () => void) => void
	onUnmount: (from: HTMLDivElement) => void
	onClose?: () => void
}

const existingWidgets = new Map<string, ExistingWidget>()

const makeWidgetPopoutContainer = (name: string, onClose: () => void) => {
	const popoutParent = document.createElement("div")
	popoutParent.className = "widget-popout-iframe-container"
	const widgetName = document.createElement("h3")
	widgetName.textContent = name
	widgetName.className = "widget-popout-title"
	widgetName.onpointerdown = (evt: PointerEvent) => {
		if (!evt.isPrimary || (evt.pointerType === "mouse" && evt.button !== 0)) {
			return
		}
		const rect = popoutParent.getBoundingClientRect()
		const documentRect = document.body.getBoundingClientRect()
		const offsetX = evt.clientX - rect.left
		const offsetY = evt.clientY - rect.top
		const pointermove = (evt: PointerEvent) => {
			// Let the popup go slightly outside the sides and bottom of the screen, but not the top
			const halfWidth = rect.width / 2
			const halfHeight = rect.height / 2
			const newX = Math.min(Math.max(evt.clientX - offsetX, -halfWidth), documentRect.width - halfWidth)
			const newY = Math.min(Math.max(evt.clientY - offsetY, 0), documentRect.height - halfHeight)
			popoutParent.style.left = `${newX}px`
			popoutParent.style.top = `${newY}px`
		}
		const pointerup = () => {
			widgetName.removeEventListener("pointerup", pointerup)
			widgetName.addEventListener("pointercancel", pointerup)
			widgetName.removeEventListener("pointermove", pointermove)
			widgetName.style.cursor = "grab"
			widgetName.releasePointerCapture(evt.pointerId)
		}
		widgetName.addEventListener("pointerup", pointerup)
		widgetName.addEventListener("pointercancel", pointerup)
		widgetName.addEventListener("pointermove", pointermove)
		widgetName.setPointerCapture(evt.pointerId)
		widgetName.style.cursor = "grabbing"
	}
	const closeBtn = document.createElement("button")
	closeBtn.className = "widget-popout-close-button"
	closeBtn.title = "Close widget"
	closeBtn.innerText = "✕"
	closeBtn.onclick = onClose
	popoutParent.appendChild(widgetName)
	popoutParent.appendChild(closeBtn)
	document.body.appendChild(popoutParent)
	return popoutParent
}

const makeWidget = ({ room, info, client, noPermissionPrompt }: WidgetProps) => {
	const fullWidgetID = `widget;${room.roomID};${info.id}`
	let widgetContainer = existingWidgets.get(fullWidgetID)
	if (widgetContainer) {
		return widgetContainer
	}
	const wrappedWidget = new WrappedWidget(info)
	const driver = new GomuksWidgetDriver(client, room, noPermissionPrompt ? noopPermissions : openPermissionPrompt)
	const widgetURL = addLegacyParams(wrappedWidget.getCompleteUrl({
		widgetRoomId: room.roomID,
		currentUserId: client.userID,
		deviceId: client.state.current?.is_logged_in ? client.state.current.device_id : "",
		userDisplayName: getDisplayname(client.userID, room.getStateEvent("m.room.member", client.userID)?.content),
		clientId: "fi.mau.gomuks",
		clientTheme: window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light",
		clientLanguage: navigator.language,
	}), wrappedWidget.id)

	const iframe = document.createElement("iframe")
	iframe.src = widgetURL
	iframe.className = "widget-iframe"
	iframe.allow = "microphone; camera; fullscreen; encrypted-media; display-capture; screen-wake-lock;"
	iframe.id = fullWidgetID

	let deleted = false
	let alwaysOnScreen = false
	let popoutParent: HTMLDivElement | null = null
	let clientAPI: ClientWidgetApi
	let removeListener: (() => void)

	const deleteWidget = () => {
		if (deleted) {
			return
		}
		existingWidgets.delete(fullWidgetID)
		deleted = true
		console.info("Deleting widget", fullWidgetID)
		removeListener?.()
		clientAPI?.stop()
		clientAPI?.removeAllListeners()
		iframe.remove()
		if (popoutParent) {
			popoutParent.remove()
			popoutParent = null
		}
	}
	const onSetup = () => {
		console.info("Setting up widget API for", iframe, fullWidgetID)
		clientAPI = new ClientWidgetApi(wrappedWidget, iframe, driver)
		clientAPI.setViewedRoomId(room.roomID)
		removeListener = client.addWidgetListener(new WidgetListenerImpl(clientAPI))

		clientAPI.on("ready", () => console.info("Widget is ready", fullWidgetID))
		// Suppress unnecessary events to avoid errors
		const noopReply = (evt: CustomEvent) => {
			evt.preventDefault()
			clientAPI.transport.reply(evt.detail, {})
		}
		clientAPI.on("action:io.element.join", noopReply)
		clientAPI.on("action:im.vector.hangup", noopReply)
		clientAPI.on("action:io.element.device_mute", noopReply)
		clientAPI.on("action:io.element.tile_layout", noopReply)
		clientAPI.on("action:io.element.spotlight_layout", noopReply)
		clientAPI.on("action:io.element.close", (evt: CustomEvent<IModalWidgetCloseRequest>) => {
			noopReply(evt)
			if (widgetContainer?.onClose && !deleted) {
				widgetContainer.onClose()
			}
			deleteWidget()
		})
		clientAPI.on("action:set_always_on_screen", (evt: CustomEvent<IStickyActionRequest>) => {
			if (!HTMLDivElement.prototype.moveBefore) {
				throw new Error("Browser does not support moving iframes without resetting")
			}
			alwaysOnScreen = evt.detail.data.value
			noopReply(evt)
		})
	}
	const onMount = (into: HTMLDivElement, onClose?: () => void) => {
		if (deleted) {
			return
		}
		console.info("Mounting widget", fullWidgetID, "into", into)
		widgetContainer!.onClose = onClose
		if (iframe.parentElement && into.moveBefore) {
			into.moveBefore(iframe, null)
		} else {
			into.appendChild(iframe)
		}
		if (!clientAPI) {
			onSetup()
		}
		if (popoutParent) {
			popoutParent.remove()
			popoutParent = null
		}
	}
	const onUnmount = (from: HTMLDivElement) => {
		if (iframe.parentElement !== from || deleted) {
			return
		} else if (!alwaysOnScreen || !HTMLDivElement.prototype.moveBefore) {
			deleteWidget()
		} else if (!popoutParent) {
			console.info("Popping out widget", fullWidgetID)
			popoutParent = makeWidgetPopoutContainer(info.name || "Unnamed widget", deleteWidget)
			popoutParent.moveBefore!(iframe, null)
		} else {
			console.error("Unexpected widget onUnmount call for already popped-out widget")
		}
	}

	widgetContainer = { onMount, onUnmount }
	existingWidgets.set(fullWidgetID, widgetContainer)
	return widgetContainer!
}

const ReactWidget = (props: WidgetProps) => {
	return <div className="widget-container" ref={(elem) => {
		if (!elem) {
			return
		}
		const { onMount, onUnmount } = makeWidget(props)
		onMount(elem, props.onClose)
		return () => onUnmount(elem)
	}} />
}

export default memo(ReactWidget)
