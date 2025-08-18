// gomuks - A Matrix client written in Go.
// Copyright (C) 2024 Nexus Nicholson
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
import React, { JSX, use } from "react"
import { TombstoneEventContent } from "@/api/types"
import { ensureString, getDisplayname, getServerName, isRoomID } from "@/util/validation.ts"
import MainScreenContext from "../../MainScreenContext.ts"
import EventContentProps from "./props.ts"

const RoomTombstoneBody = ({ event, sender }: EventContentProps) => {
	const mainScreen = use(MainScreenContext)!
	const content = event.content as TombstoneEventContent
	const body = ensureString(content.body)
	const replacementRoom = ensureString(content.replacement_room)
	const end = body ? ` with the message: ${body}` : "."
	const via = getServerName(event.sender)
	const onClick = (e: React.MouseEvent<HTMLAnchorElement, MouseEvent>) => {
		e.preventDefault()
		mainScreen.setActiveRoom(replacementRoom, {
			previewMeta: {
				via: [via],
			},
		})
	}
	let description: JSX.Element
	if (isRoomID(replacementRoom)) {
		description = <span>
			replaced this room with <a
				onClick={onClick}
				href={`matrix:roomid/${replacementRoom.slice(1)}?via=${via}`}
				className="hicli-matrix-uri"
			>
				{replacementRoom}
			</a>{end}
		</span>
	} else {
		description = <span>shut down this room{end}</span>
	}
	return <div className="room-tombstone-body">
		{getDisplayname(event.sender, sender?.content)} {description}
	</div>
}

export default RoomTombstoneBody
