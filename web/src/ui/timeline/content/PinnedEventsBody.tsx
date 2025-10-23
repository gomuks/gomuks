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
import { JSX, MouseEvent } from "react"
import { EventID, PinnedEventsContent, RoomID } from "@/api/types"
import { RoomContextData, useRoomContext } from "@/ui/roomview/roomcontext.ts"
import { jumpToEvent, jumpToVisibleEvent } from "@/ui/util/jumpToEvent.tsx"
import { listDiff } from "@/util/diff.ts"
import { ensureTypedArray, getDisplayname, isEventID } from "@/util/validation.ts"
import EventContentProps from "./props.ts"

function renderPinChanges(
	roomID: RoomID,
	roomCtx: RoomContextData,
	content: PinnedEventsContent,
	prevContent?: PinnedEventsContent,
): JSX.Element {
	const [added, removed] = listDiff(
		ensureTypedArray(content.pinned ?? [], isEventID),
		ensureTypedArray(prevContent?.pinned ?? [], isEventID),
	)
	const jumpToOnClick = (event_id: EventID) => (evt: MouseEvent<HTMLAnchorElement>) => {
		evt.preventDefault()
		evt.stopPropagation()
		if (!jumpToVisibleEvent(event_id, evt.currentTarget.closest(".timeline-list"))) {
			jumpToEvent(roomCtx, event_id)
		}
	}
	const encode = (value: string) => encodeURIComponent(value).replace("%3A", ":")
	const uri = (e: EventID) => `matrix:roomid/${encode(roomID.slice(1))}/e/${encode(e.slice(1))}`

	const renderEventLink = (event_id: EventID) => (
		<a key={event_id} href={uri(event_id)} onClick={jumpToOnClick(event_id)}>
			{event_id}
		</a>
	)

	const joinElements = (elements: JSX.Element[]) => {
		if (elements.length === 0) {
			return null
		}
		if (elements.length === 1) {
			return elements[0]
		}
		if (elements.length === 2) {
			return <>{elements[0]} and {elements[1]}</>
		}
		return <>
			{elements.slice(0, -1).map((el, i) => (
				<span key={i}>{el}{i < elements.length - 2 ? ", " : ""}</span>
			))}
			{" and "}
			{elements[elements.length - 1]}
		</>
	}

	if (added.length) {
		const addedLinks = added.map(renderEventLink)
		if (removed.length) {
			const removedLinks = removed.map(renderEventLink)
			return <>pinned {joinElements(addedLinks)} and unpinned {joinElements(removedLinks)}</>
		}
		return <>pinned {joinElements(addedLinks)}</>
	} else if (removed.length) {
		const removedLinks = removed.map(renderEventLink)
		return <>unpinned {joinElements(removedLinks)}</>
	} else {
		return <>sent a no-op pin event</>
	}
}

const PinnedEventsBody = ({ room, event, sender }: EventContentProps) => {
	const roomCtx = useRoomContext()
	const content = event.content as PinnedEventsContent
	const prevContent = event.unsigned.prev_content as PinnedEventsContent | undefined
	return <div className="pinned-events-body">
		{getDisplayname(event.sender, sender?.content)} {renderPinChanges(room.roomID, roomCtx, content, prevContent)}
	</div>
}

export default PinnedEventsBody
