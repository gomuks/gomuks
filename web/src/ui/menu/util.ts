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
import React, { CSSProperties } from "react"
import Client from "@/api/client.ts"
import { RoomStateStore } from "@/api/statestore"
import { MemDBEvent, PowerLevelEventContent } from "@/api/types"
import { getUserLevel } from "@/util/powerlevel.ts"

export const getPending = (evt: MemDBEvent): [pending: boolean, pendingTitle: string | undefined] => {
	const isPending = evt.event_id.startsWith("~")
	const pendingTitle = isPending ? "Can't action messages that haven't been sent yet" : undefined
	return [isPending, pendingTitle]
}

export const getPowerLevels = (room: RoomStateStore, client: Client): [
	pls: PowerLevelEventContent, ownPL: number, createEvent: MemDBEvent
] => {
	const createEvent = room.getStateEvent("m.room.create", "")
	const plEvent = room.getStateEvent("m.room.power_levels", "")
	const pls = (plEvent?.content ?? {}) as PowerLevelEventContent
	const ownPL = getUserLevel(pls, createEvent, client.userID)
	return [pls, ownPL, createEvent!]
}

export const getEncryption = (room: RoomStateStore): boolean =>{
	const encryptionEvent = room.getStateEvent("m.room.encryption", "")
	return encryptionEvent?.content?.algorithm === "m.megolm.v1.aes-sha2"
}

export function getModalStyleFromMouse(
	evt: React.MouseEvent, modalHeight: number, modalWidth = 10 * 16,
): CSSProperties {
	const style: CSSProperties = { left: evt.clientX }
	if (evt.clientX + modalWidth > window.innerWidth) {
		delete style.left
		style.right = "4px"
	}
	if (evt.clientY + modalHeight > window.innerHeight) {
		style.bottom = window.innerHeight - evt.clientY
	} else {
		style.top = evt.clientY
	}
	return style
}

export function getModalStyleFromButton(button: HTMLElement, modalHeight: number): CSSProperties {
	const rect = button.getBoundingClientRect()
	const style: CSSProperties = { right: window.innerWidth - rect.right }
	if (rect.bottom + modalHeight < window.innerHeight) {
		// Show modal below button
		style.top = rect.bottom
	} else if (rect.top > modalHeight) {
		// Show modal above button
		style.bottom = window.innerHeight - rect.top
	} else {
		// Show modal to the left of button
		style.right = window.innerWidth - rect.right + rect.width
		if (modalHeight > window.innerHeight) {
			// Modal is too big for window, top-align
			style.top = "0.5rem"
		} else if (rect.top + modalHeight < window.innerHeight) {
			// Modal fits such that the top is aligned with the button's top
			style.top = rect.top - 4
		} else {
			// Align to bottom of window
			style.bottom = "0.5rem"
		}
	}
	return style
}

export function getRightOpeningModalStyleFromButton(button: HTMLElement, modalHeight: number): CSSProperties {
	const rect = button.getBoundingClientRect()
	const style: CSSProperties = { left: rect.left + rect.width }
	if (rect.top + modalHeight > window.innerHeight) {
		style.top = window.innerHeight - modalHeight
	} else {
		style.top = rect.top
	}
	return style
}
