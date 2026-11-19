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
import { JSX, useState } from "react"
import { MemberEventContent } from "@/api/types"
import { getUserLevel } from "@/util/powerlevel.ts"
import { ensureString, getDisplayname } from "@/util/validation.ts"
import { getPowerLevels } from "../../menu/util.ts"
import EventContentProps from "./props.ts"
import DeleteIcon from "../../../icons/delete.svg?react"

const BlurredText = ({ text }: { text: string }) => {
	const [show, setShow] = useState(false)
	return <span
		style={{ filter: show ? "unset" : undefined }}
		className="blurred-text"
		onClick={() => setShow(!show)}
	>{text}</span>
}

const RedactedBody = ({ event, sender, room }: EventContentProps) => {
	let sentBy = ""
	let forReason: JSX.Element | string | undefined
	if (event.redacted_by) {
		// TODO store redaction sender in database instead of fetching the entire redaction event
		const redactedByEvent = room.eventsByID.get(event.redacted_by)
		if (redactedByEvent && redactedByEvent.sender !== event.sender) {
			const redacterProfile = room.getStateEvent("m.room.member", redactedByEvent.sender)
			sentBy = `by ${getDisplayname(redactedByEvent.sender, redacterProfile?.content)}`
		}
		let reason = ensureString(redactedByEvent?.content?.reason)
		if (reason) {
			if (reason.length > 120) {
				reason = reason.slice(0, 100) + "[…]"
			}
			const [pls, , createEvent] = getPowerLevels(room)
			if (getUserLevel(pls, createEvent, redactedByEvent!.sender) > (pls.redact ?? 50)) {
				forReason = `for ${reason}`
			} else {
				forReason = <>for&nbsp;<BlurredText text={reason} /></>
			}
		}
	} else {
		const senderContent = sender?.content as MemberEventContent | undefined
		if (senderContent?.["org.matrix.msc4293.redact_events"]) {
			sentBy = "via ban event"
		}
	}
	return <div className="redacted-body">
		<DeleteIcon/> <div>Message deleted {sentBy} {forReason}</div>
	</div>
}

export default RedactedBody
