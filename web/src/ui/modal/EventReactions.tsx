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
import { use, useCallback, useEffect, useState } from "react"
import { ScaleLoader } from "react-spinners"
import Client from "@/api/client.ts"
import { getAvatarThumbnailURL, getAvatarURL, getMediaURL } from "@/api/media.ts"
import { RoomStateStore, maybeRedactMemberEvent, useRoomMember, useRoomState } from "@/api/statestore"
import { EventID, MemDBEvent, UserID } from "@/api/types"
import { getPowerLevels } from "@/ui/menu/util.ts"
import { LightboxContext } from "@/ui/modal/contexts.ts"
import { getEventLevel } from "@/util/powerlevel.ts"
import { ensureString, getDisplayname } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import { RoomContextData } from "../roomview/roomcontext.ts"
import DeleteIcon from "@/icons/delete.svg?react"
import "./EventReactions.css"

interface EventReactionsProps {
	evt: MemDBEvent
	roomCtx: RoomContextData
}

interface ReactionUser {
	eventIDs: EventID[]
	firstTimestamp: Date
}

interface FullReactionInfo {
	key: string
	emoji?: string
	mxc?: string
	count: number
	users: Map<UserID, ReactionUser>
	shortcode?: string
	firstTimestamp?: number
}

const mapReactionEvents = (reactions: MemDBEvent[]) => {
	const infos: FullReactionInfo[] = []
	const infoMap = new Map<string, FullReactionInfo>()
	const getInfo = (key: string): FullReactionInfo => {
		let info = infoMap.get(key)
		if (!info) {
			info = {
				key,
				users: new Map(),
				count: 0,
			}
			infoMap.set(key, info)
			infos.push(info)
		}
		return info
	}
	for (const evt of reactions) {
		if (evt.redacted_by) {
			continue
		}
		const key = ensureString(evt.content["m.relates_to"]?.key)
		if (!key) {
			continue
		}
		const info = getInfo(key)
		if (key.startsWith("mxc://")) {
			info.mxc = key
		} else {
			info.emoji = key
		}
		if (info.mxc && !info.shortcode && evt.content["com.beeper.reaction.shortcode"]) {
			info.shortcode = ensureString(evt.content["com.beeper.reaction.shortcode"]).replaceAll(":", "")
		}
		if (!info.firstTimestamp || info.firstTimestamp > evt.timestamp) {
			info.firstTimestamp = evt.timestamp
		}
		info.count++
		const item = info.users.get(evt.sender)
		if (!item) {
			info.users.set(evt.sender, {
				eventIDs: [evt.event_id],
				firstTimestamp: newSafeDate(evt.timestamp),
			})
		} else {
			item.eventIDs.push(evt.event_id)
		}
	}
	// infos.sort((a, b) => (a.firstTimestamp ?? 0) - (b.firstTimestamp ?? 0))
	return infos
}

interface ReactionSenderProps {
	client: Client
	userID: UserID
	events: ReactionUser
	room: RoomStateStore
	canDelete: boolean
	removeReactionCache: (userID: UserID, eventID: EventID) => void
}

const fullTimeFormatter = new Intl.DateTimeFormat("en-GB", { dateStyle: "full", timeStyle: "medium" })
const newSafeDate = (val: number) => {
	const date = new Date(val)
	if (isNaN(+date)) {
		return new Date(0)
	}
	return date
}

const ReactionSender = ({ client, userID, events, room, canDelete, removeReactionCache }: ReactionSenderProps) => {
	const memberEvt = useRoomMember(client, room, userID)
	const member = maybeRedactMemberEvent(memberEvt)
	const [deleting, setDeleting] = useState(false)
	const onClickDelete = () => {
		if (userID === client.userID || window.confirm(`Redact reaction from ${userID}?`)) {
			setDeleting(true)
			client.rpc.redactEvent(room.roomID, events.eventIDs[0], "").then(
				() => removeReactionCache(userID, events.eventIDs[0]),
				err => window.alert(`Failed to redact reaction: ${err}`),
			).finally(() => setDeleting(false))
		}
	}
	return <div className="reaction-sender" title={fullTimeFormatter.format(events.firstTimestamp)}>
		{canDelete && <button className="delete-button" onClick={onClickDelete} disabled={deleting}>
			<DeleteIcon />
		</button>}
		<img
			className="small avatar"
			loading="lazy"
			src={getAvatarThumbnailURL(userID, member)}
			data-full-src={getAvatarURL(userID, member)}
			onClick={use(LightboxContext)}
			alt=""
		/>
		{getDisplayname(userID, member)}
		{events.eventIDs.length > 1 ? ` (x${events.eventIDs.length})` : ""}
	</div>
}

interface ReactionItemProps {
	client: Client
	info: FullReactionInfo
	room: RoomStateStore
	canRedactOwn: boolean
	canRedactOthers: boolean
	removeReactionCache: (userID: UserID, eventID: EventID) => void
}

const seg = new Intl.Segmenter()

const moreThanOneGrapheme = (str?: string) => {
	if (!str || str.length < 2) {
		return false
	}
	const segments = seg.segment(str)
	let count = 0
	// eslint-disable-next-line @typescript-eslint/no-unused-vars
	for (const _ of segments) {
		count++
		if (count > 1) {
			return true
		}
	}
	return false
}

const ReactionItem = ({
	client, info, room, canRedactOwn, canRedactOthers, removeReactionCache,
}: ReactionItemProps) => {
	const colonShortcode = `:${info.shortcode}:`
	return <>
		<div className="emoji">
			{info.mxc ? <img
				className="custom-emoji"
				src={getMediaURL(info.mxc)}
				alt={colonShortcode}
				title={colonShortcode}
				onClick={use(LightboxContext)}
			/> : <span className={`unicode-emoji ${moreThanOneGrapheme(info.emoji) ? "long" : ""}`}>{info.emoji}</span>}
		</div>
		<div className="senders">
			{info.users.entries().map(([userID, events]) =>
				<ReactionSender
					client={client}
					key={userID}
					events={events}
					userID={userID}
					room={room}
					removeReactionCache={removeReactionCache}
					canDelete={client.userID === userID ? canRedactOwn : canRedactOthers}
				/>)}
		</div>
	</>
}

const EventReactions = ({ evt, roomCtx }: EventReactionsProps) => {
	const client = use(ClientContext)!
	const [reactions, setReactions] = useState<FullReactionInfo[]>([])
	const [error, setError] = useState("")
	const [loading, setLoading] = useState(true)

	useRoomState(roomCtx.store, "m.room.power_levels", "")
	const [pls, ownPL] = getPowerLevels(roomCtx.store, client)
	const canRedactOwn = ownPL >= getEventLevel(pls, "m.room.redaction")
	const canRedactOthers = canRedactOwn && ownPL >= (pls?.redact ?? 50)

	useEffect(() => {
		setLoading(true)
		setError("")
		setReactions([])
		client.getRelatedEvents(roomCtx.store, evt.event_id, "m.annotation").then(
			reactions => setReactions(mapReactionEvents(reactions)),
			err => {
				console.error("Failed to get event reactions", err)
				setError(`${err}`)
			},
		).finally(() => setLoading(false))
	}, [client, roomCtx, evt])
	const removeReactionCache = useCallback((userID: UserID, eventID: EventID) => {
		setReactions(prev => prev.map(info => {
			const user = info.users.get(userID)
			if (user && user.eventIDs.includes(eventID)) {
				user.eventIDs = user.eventIDs.filter(id => id !== eventID)
				if (user.eventIDs.length === 0) {
					info.users.delete(userID)
				}
				info.count--
				return { ...info }
			}
			return info
		}))
	}, [])

	if (loading) {
		return <div><ScaleLoader color="var(--primary-color)" /></div>
	} else if (error) {
		return <div>Failed to load :( {error}</div>
	}
	return reactions.map(info => <ReactionItem
		key={info.key}
		info={info}
		client={client}
		room={roomCtx.store}
		canRedactOwn={canRedactOwn}
		canRedactOthers={canRedactOthers}
		removeReactionCache={removeReactionCache}
	/>)
}

export default EventReactions
