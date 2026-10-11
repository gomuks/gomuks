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
import { GalleryMessageEventContent, MediaMessageEventContent } from "@/api/types"
import { ensureString } from "@/util/validation.ts"
import MediaMessageBody from "./MediaMessageBody.tsx"
import TextMessageBody from "./TextMessageBody.tsx"
import EventContentProps from "./props.ts"

function checkGalleryMsgType(msgtype: unknown): boolean {
	return msgtype === "m.image" || msgtype === "m.video" || msgtype === "m.audio" || msgtype === "m.file"
}

function checkGalleryItem(item: unknown): boolean {
	if (!item || typeof item !== "object" || Array.isArray(item)) {
		return false
	}
	if ("msgtype" in item) {
		return checkGalleryMsgType(item.msgtype)
	} else if ("itemtype" in item) {
		return checkGalleryMsgType(item.itemtype)
	}
	return false
}

const GalleryMessageBody = ({ event, room, sender }: EventContentProps) => {
	const content = event.content as GalleryMessageEventContent
	let caption = null
	if (content.body) {
		caption = <TextMessageBody event={event} room={room} sender={sender} />
	}
	let items = content.items ?? content.itemtypes
	if (!Array.isArray(items)) {
		items = []
	}

	return <>
		<div className="gallery-items">
			{items.map((item, i) => checkGalleryItem(item) ? <MediaMessageBody
				room={room} event={event} sender={sender} content={item as MediaMessageEventContent} key={i}
			/> : <code>{`{ "msgtype": "${ensureString(item?.msgtype || item?.itemtype)}" }`}</code>)}
		</div>
		{caption}
	</>
}

export default GalleryMessageBody
