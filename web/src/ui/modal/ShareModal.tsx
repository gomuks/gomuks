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
import { useState } from "react"
import { RoomStateStore } from "@/api/statestore"
import { MemDBEvent } from "@/api/types"
import useEvent from "@/util/useEvent.ts"
import { lessNoisyEncodeURIComponent } from "@/util/validation.ts"
import Toggle from "../util/Toggle.tsx"
import ConfirmModal from "./ConfirmModal.tsx"

export interface ShareModalProps {
	room: RoomStateStore
	evt?: MemDBEvent
}

const emptyArgs = [] as const

export const ShareModal = ({ room, evt }: ShareModalProps) => {
	const [useMatrixTo, setUseMatrixTo] = useState(false)
	const [includeEvent, setIncludeEvent] = useState(!!evt)
	const alias = room.meta.current.canonical_alias
	const [useRoomAlias, setUseRoomAlias] = useState(!evt && !!alias)
	const actuallyUseAlias = alias && useRoomAlias

	let generatedURL = useMatrixTo ? "https://matrix.to/#/" : "matrix:"
	if (useMatrixTo) {
		generatedURL += lessNoisyEncodeURIComponent(actuallyUseAlias ? alias : room.roomID)
	} else if (actuallyUseAlias) {
		generatedURL += "r/" + lessNoisyEncodeURIComponent(`${alias.slice(1)}`)
	} else {
		generatedURL += "roomid/" + lessNoisyEncodeURIComponent(`${room.roomID.slice(1)}`)
	}
	if (evt && includeEvent) {
		if (useMatrixTo) {
			generatedURL += `/${encodeURIComponent(evt.event_id)}`
		} else {
			generatedURL += `/e/${encodeURIComponent(evt.event_id.slice(1))}`
		}
	}
	if (!actuallyUseAlias) {
		generatedURL += "?" + new URLSearchParams(
			room.getViaServers().map(server => ["via", server]),
		).toString()
	}

	const onConfirm = useEvent(() => {
		navigator.clipboard.writeText(generatedURL).catch(
			err => window.alert(`Failed to copy link: ${err}`),
		)
	})

	return <ConfirmModal<readonly never[]>
		evt={evt}
		title={evt ? "Share Message" : "Share Room"}
		confirmButton="Copy to clipboard"
		onConfirm={onConfirm}
		confirmArgs={emptyArgs}
	>
		<div className="toggle-sheet">
			<label htmlFor="use-matrix-to">Use matrix.to link</label>
			<Toggle
				id="use-matrix-to"
				checked={useMatrixTo}
				onChange={evt => setUseMatrixTo(evt.target.checked)}
			/>
			{evt ? <>
				<label htmlFor="share-event">Link to this specific event</label>
				<Toggle
					id="share-event"
					checked={includeEvent}
					onChange={evt => setIncludeEvent(evt.target.checked)}
				/>
			</> : alias ? <>
				<label htmlFor="use-room-alias">Link to room alias</label>
				<Toggle
					id="use-room-alias"
					checked={useRoomAlias}
					onChange={evt => setUseRoomAlias(evt.target.checked)}
				/>
			</> : null}
		</div>
		<div className="output-preview">
			<span className="no-select">Preview: </span><code>{generatedURL}</code>
		</div>
	</ConfirmModal>
}

export default ShareModal
