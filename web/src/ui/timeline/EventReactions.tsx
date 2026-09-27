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
import React, { use, useEffect, useState } from "react"
import Client from "@/api/client.ts"
import { getMediaURL } from "@/api/media.ts"
import { RoomStateStore } from "@/api/statestore"
import { EventID } from "@/api/types"
import { getEventLevel } from "@/util/powerlevel.ts"
import ClientContext from "../ClientContext.ts"
import { getPowerLevels } from "../menu/util.ts"

interface EventReactionsProps {
	room: RoomStateStore
	eventID: EventID
	reactions: Record<string, number>
	ownReactions?: Record<string, EventID[]>
}

interface EventReactionProps {
	room: RoomStateStore
	eventID: EventID
	reaction: string
	count: number
	ownEventIDs?: EventID[]
	canReact: boolean
	canUnreact: boolean
	client: Client
}

const EventReaction = ({
	room, eventID, reaction, count, ownEventIDs, canReact, canUnreact, client,
}: EventReactionProps) => {
	const [isLoading, setIsLoading] = useState(false)
	const [locallySent, setLocallySent] = useState<EventID>()
	const [redacted, setRedacted] = useState<EventID[]>()
	let firstOwnID = locallySent ?? ownEventIDs?.[0]
	if (firstOwnID && !locallySent && redacted) {
		firstOwnID = ownEventIDs?.filter(id => !redacted?.includes(id))?.[0]
	}
	const canAct = firstOwnID ? canUnreact : canReact
	useEffect(() => {
		if (locallySent && ownEventIDs?.includes(locallySent)) {
			setLocallySent(undefined)
		}
	}, [locallySent, ownEventIDs])
	const onClick = (mouseEvt: React.MouseEvent) => {
		if (isLoading || !canAct) {
			return
		}
		setIsLoading(true)
		if (firstOwnID) {
			client.rpc.redactEvent(room.roomID, firstOwnID, "").then(() => {
				setLocallySent(undefined)
				setRedacted(redacted => [...(redacted ?? []), firstOwnID])
			}).catch(err => {
				console.error("Failed to remove reaction", err)
				window.alert(`Failed to remove reaction: ${err}`)
			}).finally(() => setIsLoading(false))
		} else {
			client.sendEvent(room.roomID, "m.reaction", {
				"m.relates_to": {
					rel_type: "m.annotation",
					event_id: eventID,
					key: reaction,
				},
			}).then(rowid => {
				const checkEventState = () => {
					const currentEvt = room.eventsByRowID.get(rowid)
					if (
						currentEvt?.event_id?.startsWith("$")
						|| (currentEvt?.send_error && currentEvt.send_error !== "not sent")
					) {
						unsub()
						if (currentEvt.event_id.startsWith("$")) {
							setLocallySent(currentEvt.event_id)
						}
						setIsLoading(false)
					}
				}
				const unsub = room.eventRowIDSubs.getSubscriber(rowid)(checkEventState)
				checkEventState()
			}).catch(err => {
				console.error("Failed to send reaction", err)
				window.alert(`Failed to send reaction: ${err}`)
				setIsLoading(false)
			})
		}
		mouseEvt.stopPropagation()
	}
	const classNames = ["reaction"]
	if (firstOwnID) {
		classNames.push("own-reaction")
	}
	if (isLoading || !canAct) {
		classNames.push("disabled")
	}
	return <div
		key={reaction}
		className={classNames.join(" ")}
		title={reaction}
		onClick={onClick}
	>
		{reaction.startsWith("mxc://")
			? <img className="reaction-emoji" src={getMediaURL(reaction)} alt=""/>
			: <span className="reaction-emoji">{reaction}</span>}
		<span className="reaction-count">{count}</span>
	</div>
}

const EventReactions = ({ room, eventID, reactions, ownReactions }: EventReactionsProps) => {
	const reactionEntries = Object.entries(reactions)
		.filter(([, count]) => count > 0)
		.sort((a, b) => b[1] - a[1])
	if (reactionEntries.length === 0) {
		return null
	}
	const client = use(ClientContext)!
	const [pls, ownPL] = getPowerLevels(room, client)
	const canReact = ownPL >= getEventLevel(pls, "m.reaction")
	const canUnreact = ownPL >= getEventLevel(pls, "m.room.redaction")
	return <div className="event-reactions">
		{reactionEntries.map(([reaction, count]) => <EventReaction
			key={reaction}
			room={room}
			eventID={eventID}
			reaction={reaction}
			count={count}
			ownEventIDs={ownReactions?.[reaction]}
			canReact={canReact}
			canUnreact={canUnreact}
			client={client}
		/>)}
	</div>
}

export default EventReactions
