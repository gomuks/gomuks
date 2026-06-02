// gomuks - A Matrix client written in Go.
// Copyright (C) 2026 Tulir Asokan
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
import { use, useRef, useState } from "react"
import { ScaleLoader } from "react-spinners"
import { LocalSearchParams, MemDBEvent, RoomID, UserID } from "@/api/types"
import ClientContext from "../ClientContext.ts"
import { RoomContext, RoomContextData } from "../roomview/roomcontext.ts"
import TimelineEvent from "../timeline/TimelineEvent.tsx"

const BATCH_SIZE = 50

type MutableParams =
	"search_term"
	| "raw_like"
	| "room_ids"
	| "senders"
	| "max_timestamp"
	| "min_timestamp"
	| "sort_by_time"
	| "include_redacted"

const MessageSearch = () => {
	const roomCtx = use(RoomContext)
	const client = use(ClientContext)!
	const [events, setEvents] = useState<MemDBEvent[]>([])
	const [nextBatch, setNextBatch] = useState<string | undefined>()
	const [loading, setLoading] = useState(false)
	const [error, setError] = useState<string | null>(null)
	const [searchTerm, setSearchTerm] = useState("")
	const [rawLike, setRawLike] = useState("")
	const [local, setLocal] = useState(!!roomCtx?.store.meta.current.encryption_event)
	const [sortByTime, setSortByTime] = useState(false)
	const [includeRedacted, setIncludeRedacted] = useState(true)
	const [minDate, setMinDate] = useState("")
	const [maxDate, setMaxDate] = useState("")
	const [minTimestamp, setMinTimestamp] = useState<number | undefined>(undefined)
	const [maxTimestamp, setMaxTimestamp] = useState<number | undefined>(undefined)
	const [roomIDs, setRoomIDs] = useState<RoomID[]>(() => roomCtx ? [roomCtx.store.roomID] : [])
	const [senders, setSenders] = useState<UserID[]>([])
	const cancelLoad = useRef<(() => void) | null>(null)
	const loadDebounce = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
	const roomContexts = useRef<Map<RoomID, RoomContextData>>(new Map())

	const makeParams = (overrides: Partial<LocalSearchParams> = {}): LocalSearchParams => ({
		limit: BATCH_SIZE,
		search_term: searchTerm,
		raw_like: rawLike,
		room_ids: roomIDs,
		senders: senders,
		max_timestamp: maxTimestamp,
		min_timestamp: minTimestamp,
		include_redacted: includeRedacted,
		sort_by_time: sortByTime,
		...overrides,
	})
	const clearResults = () => {
		setEvents([])
		setNextBatch(undefined)
	}
	const loadImmediate = (local: boolean, params: LocalSearchParams, reset: boolean = true) => {
		if (!params.search_term && !params.raw_like) {
			clearResults()
			return
		}
		if (!reset) {
			if (!nextBatch) {
				return
			}
			params.next_batch = nextBatch
		} else {
			clearResults()
		}
		clearTimeout(loadDebounce?.current)
		loadDebounce.current = undefined
		cancelLoad.current?.()
		setLoading(true)

		let canceled = false
		let cancelFn: (() => void) | null = null
		const promise = client.search(local, params)
		promise.then(
			([res, nextBatch]) => {
				if (!canceled) {
					for (const evt of res) {
						if (!roomContexts.current.has(evt.room_id)) {
							const room = client.store.rooms.get(evt.room_id)
							if (room) {
								const ctx = new RoomContextData(room)
								// TODO make appropriate things read this flag and jump to the real room view
								ctx.isFake = true
								roomContexts.current.set(evt.room_id, ctx)
							}
						}
					}
					if (reset) {
						setEvents(res)
					} else {
						setEvents(evts => evts.concat(res))
					}
					setNextBatch(nextBatch)
					setError(null)
				}
			},
			err => {
				if (!canceled) {
					setError(`${err}`)
					if (reset) {
						clearResults()
					}
				}
			},
		).finally(() => {
			cancelFn = null
			if (!canceled) {
				setLoading(false)
			}
		})
		cancelFn = () => promise.cancel("")
		cancelLoad.current = () => {
			canceled = true
			cancelFn?.()
			cancelFn = null
		}
	}
	const loadMore = () => {
		loadImmediate(local, makeParams(), false)
	}
	const loadDebounced = (params: LocalSearchParams) => {
		clearTimeout(loadDebounce?.current)
		cancelLoad.current?.()
		loadDebounce.current = setTimeout(() => loadImmediate(local, params), 500)
	}
	const setAndReload = <K extends MutableParams>(
		key: K,
		value: NonNullable<LocalSearchParams[K]>,
		debounced: boolean = false,
	) => {
		const setter = {
			search_term: setSearchTerm,
			raw_like: setRawLike,
			room_ids: setRoomIDs,
			senders: setSenders,
			max_timestamp: setMaxTimestamp,
			min_timestamp: setMinTimestamp,
			sort_by_time: setSortByTime,
			include_redacted: setIncludeRedacted,
		}[key] as (value: NonNullable<LocalSearchParams[K]>) => void
		setter(value)
		if (debounced) {
			loadDebounced(makeParams({ [key]: value }))
		} else {
			loadImmediate(local, makeParams({ [key]: value }))
		}
	}

	const contentClassNames = [
		"search-panel-content",
		events.length === 0 ? "empty" : "",
	]
	return <>
		<div className="search-controls">
			<input
				type="search"
				placeholder="Search term"
				value={searchTerm}
				autoFocus
				onChange={e => setAndReload("search_term", e.target.value, true)}
			/>
			<details>
				<summary>Options</summary>

				<label>
					Current room only
					<input
						type="checkbox"
						checked={roomIDs.length === 1 && roomIDs[0] === roomCtx?.store.roomID}
						disabled={!roomCtx}
						onChange={e =>
							setAndReload("room_ids", e.currentTarget.checked ? [roomCtx!.store.roomID] : [])}
					/>
				</label>
				<label>
					Sort by time
					<input
						type="checkbox"
						checked={sortByTime}
						onChange={e => setAndReload("sort_by_time", e.currentTarget.checked)}
					/>
				</label>
				<label>
					Search local database
					<input
						type="checkbox"
						checked={local}
						onChange={e => {
							setLocal(e.currentTarget.checked)
							loadImmediate(e.currentTarget.checked, makeParams())
						}}
					/>
				</label>
				{local && <>
					<label>
						Include redacted events
						<input
							type="checkbox"
							checked={includeRedacted}
							disabled={!local}
							onChange={e => setAndReload("include_redacted", e.currentTarget.checked)}
						/>
					</label>
					<input
						type="search"
						className="raw-like-input"
						placeholder="Raw LIKE query"
						value={rawLike}
						onChange={e => setAndReload("raw_like", e.currentTarget.value, true)}
					/>
					<label>
						After
						<input
							type="date"
							value={minDate}
							onChange={e => {
								setMinDate(e.currentTarget.value)
								setAndReload("min_timestamp", +new Date(e.currentTarget.value + " 00:00:00"))
							}}
						/>
					</label>
					<label>
						Before
						<input
							type="date"
							value={maxDate}
							onChange={e => {
								setMaxDate(e.currentTarget.value)
								setAndReload("max_timestamp", +new Date(e.currentTarget.value + " 23:59:59"))
							}}
						/>
					</label>
				</>}
			</details>
			{error ? <div className="error">
				{error}
			</div> : null}
		</div>
		<div className={contentClassNames.join(" ")}>
			{events.map((evt, i) => {
				// TODO separate event view type for search?
				const elem = <TimelineEvent
					key={evt.rowid} evt={evt} prevEvt={events[i-1] ?? null} viewType="notifications"
				/>
				if (evt.room_id !== roomCtx?.store.roomID) {
					return <RoomContext value={roomContexts.current.get(evt.room_id)}>{elem}</RoomContext>
				}
				return elem
			})}
			{(nextBatch || loading) ? <button className="load-more" onClick={loadMore} disabled={loading}>
				{loading
					? <><ScaleLoader color="var(--primary-color)"/> Searching...</>
					: "Load more results"}
			</button> : <button className="load-more" disabled>
				{events.length ? "No more results" : searchTerm ? "No results" : ""}
			</button>}
		</div>
	</>
}

export default MessageSearch
