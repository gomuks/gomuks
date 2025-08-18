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
import { use, useEffect, useLayoutEffect, useRef, useState } from "react"
import { ScaleLoader } from "react-spinners"
import { usePreference } from "@/api/statestore"
import { EventID, MemDBEvent } from "@/api/types"
import ClientContext from "../ClientContext.ts"
import { RoomContext, RoomContextData } from "../roomview/roomcontext.ts"
import { jumpToVisibleEvent } from "../util/jumpToEvent.tsx"
import { renderTimelineList } from "./timelineutil.tsx"
import "./EventContextModal.css"

export interface EventContextModalProps {
	roomCtx: RoomContextData
	eventID: EventID
}

const EventContextModal = ({ roomCtx, eventID }: EventContextModalProps) => {
	const client = use(ClientContext)!
	const room = roomCtx.store
	const [error, setError] = useState("loading")
	const [start, setStart] = useState("")
	const [startLoading, setStartLoading] = useState(false)
	const [end, setEnd] = useState("")
	const [endLoading, setEndLoading] = useState(false)
	const [timeline, setTimeline] = useState<MemDBEvent[]>([])
	const didHighlight = useRef<EventID | null>(null)
	const viewRef = useRef<HTMLDivElement>(null)
	const scrollFixRef = useRef<number>(null)
	const smallReplies = usePreference(client.store, room, "small_replies")
	useEffect(() => {
		setError("loading")
		setStartLoading(false)
		setEndLoading(false)
		client.rpc.getEventContext(room.roomID, eventID).then(res => {
			setEnd(res.end)
			setStart(res.start)
			setTimeline([
				...res.before.reverse().map(evt => room.applyEvent(evt)),
				room.applyEvent(res.event),
				...res.after.map(evt => room.applyEvent(evt)),
			])
			setError("")
			console.log("Event context loaded for", eventID, "in room", room.roomID)
		}, err => setError(`${err}`))
	}, [client, room, eventID])
	useLayoutEffect(() => {
		if (scrollFixRef.current && viewRef.current?.parentElement) {
			const scrollable = viewRef.current.parentElement
			scrollable.scrollTo({
				top: scrollable.scrollTop + (scrollable.scrollHeight - scrollFixRef.current),
				behavior: "instant",
			})
			scrollFixRef.current = null
		}
		if (didHighlight.current === eventID) {
			return
		}
		if (viewRef.current && jumpToVisibleEvent(eventID, viewRef.current)) {
			didHighlight.current = eventID
		}
	}, [eventID, timeline])

	const loadStart = () => {
		setStartLoading(true)
		client.rpc.paginateManual(room.roomID, start, "b").then(
			res => {
				scrollFixRef.current = viewRef.current?.parentElement?.scrollHeight ?? null
				setStart(res.next_batch)
				setTimeline([
					...res.events.reverse().map(evt => room.applyEvent(evt)),
					...timeline,
				])
			},
			err => window.alert(`Failed to load older messages: ${err}`),
		).finally(() => setStartLoading(false))
	}
	const loadEnd = () => {
		setEndLoading(true)
		client.rpc.paginateManual(room.roomID, end, "f").then(
			res => {
				setEnd(res.next_batch)
				setTimeline([
					...timeline,
					...res.events.map(evt => room.applyEvent(evt)),
				])
			},
			err => window.alert(`Failed to load newer messages: ${err}`),
		).finally(() => setEndLoading(false))
	}

	if (error === "loading") {
		return <div className="event-context-loading">
			<ScaleLoader color="var(--primary-color)" width="1rem" height="3rem" /> Loading event context...
		</div>
	} else if (error) {
		return <div className="event-context-loading error">
			Failed to load event context: {error}
		</div>
	}
	const content = <div className="timeline-view" ref={viewRef}>
		<div className="timeline-edge">
			{start ? <button onClick={loadStart} disabled={startLoading}>
				{startLoading
					? <><ScaleLoader color="var(--primary-color)"/> Loading older messages...</>
					: "Load older messages"}
			</button> : "No older messages available in this room"}
		</div>
		<div className="timeline-list">
			{renderTimelineList("context", timeline, { smallReplies })}
		</div>
		<div className="timeline-edge">
			{end ? <button onClick={loadEnd} disabled={endLoading}>
				{endLoading
					? <><ScaleLoader color="var(--primary-color)"/> Loading newer messages...</>
					: "Load newer messages"}
			</button> : "No newer messages available in this room"}
		</div>
	</div>
	return <RoomContext value={roomCtx}>{content}</RoomContext>
}

export default EventContextModal
