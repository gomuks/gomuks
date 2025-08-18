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
import { use, useCallback, useEffect, useLayoutEffect, useRef, useState } from "react"
import { ScaleLoader } from "react-spinners"
import { usePreference, useRoomTimeline } from "@/api/statestore"
import { EventRowID } from "@/api/types"
import useFocus from "@/util/focus.ts"
import ClientContext from "../ClientContext.ts"
import { useRoomContext } from "../roomview/roomcontext.ts"
import { renderTimelineList } from "./timelineutil.tsx"
import "./TimelineView.css"

const TimelineView = () => {
	const roomCtx = useRoomContext()
	const room = roomCtx.store
	const timeline = useRoomTimeline(room)
	const client = use(ClientContext)!
	const [isLoadingHistory, setLoadingHistory] = useState(false)
	const [focusedEventRowID, directSetFocusedEventRowID] = useState<EventRowID | null>(null)
	const loadHistory = useCallback(() => {
		setLoadingHistory(true)
		client.loadMoreHistory(room.roomID)
			.catch(err => console.error("Failed to load history", err))
			.finally(() => setLoadingHistory(false))
	}, [client, room])
	const bottomRef = roomCtx.timelineBottomRef
	const topRef = useRef<HTMLDivElement>(null)
	const timelineViewRef = useRef<HTMLDivElement>(null)
	const prevOldestTimelineRow = useRef(0)
	const oldestTimelineRow = timeline[0]?.timeline_rowid
	const oldScrollHeight = useRef(0)
	const focused = useFocus()
	const smallReplies = usePreference(client.store, room, "small_replies")

	// When the user scrolls the timeline manually, remember if they were at the bottom,
	// so that we can keep them at the bottom when new events are added.
	const handleScroll = () => {
		if (!timelineViewRef.current) {
			return
		}
		const timelineView = timelineViewRef.current
		roomCtx.scrolledToBottom = timelineView.scrollTop + timelineView.clientHeight + 1 >= timelineView.scrollHeight
	}
	// Save the scroll height prior to updating the timeline, so that we can adjust the scroll position if needed.
	if (timelineViewRef.current) {
		oldScrollHeight.current = timelineViewRef.current.scrollHeight
	}
	useLayoutEffect(() => {
		const bottomRef = roomCtx.timelineBottomRef
		if (bottomRef.current && roomCtx.scrolledToBottom) {
			// For any timeline changes, if we were at the bottom, scroll to the new bottom
			bottomRef.current.scrollIntoView()
		} else if (timelineViewRef.current && prevOldestTimelineRow.current > (timeline[0]?.timeline_rowid ?? 0)) {
			// When new entries are added to the top of the timeline, scroll down to keep the same position
			timelineViewRef.current.scrollTop += timelineViewRef.current.scrollHeight - oldScrollHeight.current
		}
		prevOldestTimelineRow.current = timeline[0]?.timeline_rowid ?? 0
	}, [client.userID, roomCtx, timeline])
	useEffect(() => {
		roomCtx.directSetFocusedEventRowID = directSetFocusedEventRowID
	}, [roomCtx])
	useEffect(() => {
		const newestEvent = timeline[timeline.length - 1]
		if (
			roomCtx.scrolledToBottom
			&& focused
			&& newestEvent
			&& newestEvent.timeline_rowid > 0
			&& (room.meta.current.marked_unread
				|| (room.readUpToRow < newestEvent.timeline_rowid
					&& newestEvent.sender !== client.userID))
		) {
			room.readUpToRow = newestEvent.timeline_rowid
			room.meta.current.marked_unread = false
			const receiptType = roomCtx.store.preferences.send_read_receipts ? "m.read" : "m.read.private"
			client.rpc.markRead(room.roomID, newestEvent.event_id, receiptType).then(
				() => console.log("Marked read up to", newestEvent.event_id, newestEvent.timeline_rowid),
				err => {
					console.error(`Failed to send read receipt for ${newestEvent.event_id}:`, err)
					room.readUpToRow = -1
				},
			)
		}
	}, [focused, client, roomCtx, room, timeline])
	useEffect(() => {
		const topElem = topRef.current
		if (!topElem || !room.hasMoreHistory) {
			return
		}
		const observer = new IntersectionObserver(entries => {
			if (entries[0]?.isIntersecting && room.paginationRequestedForRow !== prevOldestTimelineRow.current) {
				room.paginationRequestedForRow = prevOldestTimelineRow.current
				loadHistory()
			}
		}, {
			root: topElem.parentElement!.parentElement,
			rootMargin: "0px",
			threshold: 1.0,
		})
		observer.observe(topElem)
		return () => observer.unobserve(topElem)
	}, [room, room.hasMoreHistory, loadHistory, oldestTimelineRow])

	return <div className="timeline-view" onScroll={handleScroll} ref={timelineViewRef}>
		<div className="timeline-edge beginning">
			{room.hasMoreHistory ? <button onClick={loadHistory} disabled={isLoadingHistory}>
				{isLoadingHistory
					? <><ScaleLoader color="var(--primary-color)"/> Loading history...</>
					: "Load more history"}
			</button> : "No more history available in this room"}
		</div>
		<div className="timeline-list">
			<div className="timeline-top-ref" ref={topRef}/>
			{renderTimelineList("timeline", timeline, { smallReplies, focusedEventRowID })}
			<div className="timeline-bottom-ref" ref={bottomRef}/>
		</div>
	</div>
}

export default TimelineView
