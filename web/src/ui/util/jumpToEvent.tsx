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
import { EventID } from "@/api/types"
import { RoomContextData } from "../roomview/roomcontext.ts"
import EventContextModal from "../timeline/EventContextModal.tsx"

export const jumpToEvent = (roomCtx: RoomContextData, evtID: EventID, allowRetry: boolean = true) => {
	if (jumpToVisibleEvent(evtID)) {
		console.info("Jumped to event", evtID, "in visible timeline")
	} else if (roomCtx.store.timeline.length === 0 && allowRetry) {
		// Hacky sleep to let the timeline load maybe?
		console.info("Waiting for timeline to load before jumping to event", evtID)
		setTimeout(() => jumpToEvent(roomCtx, evtID, false), 500)
	} else {
		console.info("Using event context modal to jump to event", evtID)
		openEventContextModal(roomCtx, evtID)
	}
}

export const openEventContextModal = (roomCtx: RoomContextData, evtID: EventID) => {
	if (roomCtx.threadParentRoom) {
		roomCtx = roomCtx.threadParentRoom
	}
	window.openNestableModal({
		dimmed: true,
		boxed: true,
		boxClass: "event-context-modal",
		content: <EventContextModal roomCtx={roomCtx} eventID={evtID} key={evtID} />,
	})
}

export const jumpToVisibleEvent = (evtID: EventID, parent?: Element | null): boolean => {
	if (!evtID) {
		return false
	}
	const targetEvt = (parent ?? document).querySelector(
		`div[data-event-id="${CSS.escape(evtID)}"]`,
	)
	if (!targetEvt) {
		return false
	}
	targetEvt.scrollIntoView({
		block: "center",
	})
	targetEvt.classList.add("jump-highlight")
	setTimeout(() => {
		targetEvt.classList.add("jump-highlight-fadeout")
		targetEvt.classList.remove("jump-highlight")
		setTimeout(() => {
			targetEvt.classList.remove("jump-highlight-fadeout")
		}, 1500)
	}, 3000)
	return true
}
