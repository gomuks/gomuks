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
import { MemDBEvent, RoomID, UnreadType } from "@/api/types"
import reverseMap from "@/util/reversemap.ts"
import ClientContext from "../ClientContext.ts"
import { RoomContext, RoomContextData, createFakeRoomContexts } from "../roomview/roomcontext.ts"
import TimelineEvent from "../timeline/TimelineEvent.tsx"

const BATCH_SIZE = 50

const Notifications = () => {
	const roomCtx = use(RoomContext)
	const client = use(ClientContext)!
	const [events, setEvents] = useState<MemDBEvent[]>([])
	const [loading, setLoading] = useState(true)
	const [error, setError] = useState<string | null>(null)
	const [hasMore, setHasMore] = useState(true)
	const [roomScoped, setRoomScoped] = useState(false)
	const [type, setType] = useState(UnreadType.Highlight)
	const scrollFixRef = useRef<number>(null)
	const viewRef = useRef<HTMLDivElement>(null)
	const roomContexts = useRef<Map<RoomID, RoomContextData>>(new Map())
	useEffect(() => {
		setLoading(true)
		let canceled = false
		client.getMentions({ limit: BATCH_SIZE }).then(
			res => {
				if (!canceled) {
					scrollFixRef.current = 0
					createFakeRoomContexts(client, res, roomContexts.current, roomCtx?.store.roomID)
					setEvents(res)
					setHasMore(res.length >= BATCH_SIZE)
				}
			},
			err => !canceled && setError(`${err}`),
		).finally(() => !canceled && setLoading(false))
		return () => {
			canceled = true
		}
	}, [client, roomCtx?.store.roomID])
	const loadMoreMessages = (reset?: boolean, overrideRoomScoped?: boolean, overrideType?: UnreadType) => {
		if (reset) {
			setEvents([])
			setHasMore(true)
			setError(null)
		}
		setLoading(true)
		client.getMentions({
			maxTS: events.length && !reset ? events[events.length - 1].timestamp - 1 : Date.now(),
			type: overrideType ?? type,
			roomID: (overrideRoomScoped ?? roomScoped) ? roomCtx?.store.roomID : undefined,
			limit: BATCH_SIZE,
		}).then(
			res => {
				scrollFixRef.current = reset ? 0 : viewRef.current?.scrollHeight ?? null
				createFakeRoomContexts(client, res, roomContexts.current, roomCtx?.store.roomID)
				setEvents(evts => evts.concat(res))
				setHasMore(res.length >= BATCH_SIZE)
				setError(null)
			},
			err => setError(`${err}`),
		).finally(() => setLoading(false))
	}
	const reloadWithRoomScope = (roomScoped: boolean) => {
		setRoomScoped(roomScoped)
		loadMoreMessages(true, roomScoped)
	}
	const reloadWithType = (highlightsOnly: boolean) => {
		const newType = highlightsOnly ? UnreadType.Highlight : (UnreadType.Highlight | UnreadType.Notify)
		setType(newType)
		loadMoreMessages(true, undefined, newType)
	}
	useLayoutEffect(() => {
		if (scrollFixRef.current !== null && viewRef.current) {
			viewRef.current.scrollTop += viewRef.current.scrollHeight - scrollFixRef.current
			scrollFixRef.current = null
		}
	}, [events])
	const contentClassNames = [
		"notification-panel-content",
		events.length === 0 ? "empty" : "",
		type === UnreadType.Highlight ? "highlights-only" : "",
	]
	return <>
		<div className="controls">
			<label>
				Highlights only
				<input
					type="checkbox"
					checked={type === UnreadType.Highlight}
					onChange={e => reloadWithType(e.target.checked)}
				/>
			</label>
			<label>
				Current room only
				<input
					type="checkbox"
					checked={roomScoped}
					onChange={e => reloadWithRoomScope(e.target.checked)}
				/>
			</label>
			{error}
		</div>
		<div className={contentClassNames.join(" ")} ref={viewRef}>
			{(hasMore || loading) ? <button className="load-more" onClick={() => loadMoreMessages()} disabled={loading}>
				{loading
					? <>
						<ScaleLoader color="var(--primary-color)"/> Loading
						{events.length > 0 ? " more" : ""} notifications...
					</>
					: "Load more notifications"}
			</button> : <button className="load-more" disabled>No more notifications</button>}
			{reverseMap(events, (evt, i) => {
				const elem = <TimelineEvent
					key={evt.rowid} evt={evt} prevEvt={events[i+1] ?? null} viewType="notifications"
				/>
				if (evt.room_id !== roomCtx?.store.roomID) {
					return <RoomContext value={roomContexts.current.get(evt.room_id)}>{elem}</RoomContext>
				}
				return elem
			})}
		</div>
	</>
}

export default Notifications
