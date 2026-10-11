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
import { RefCallback, useState } from "react"
import Client from "@/api/client.ts"
import { RoomStateStore, usePreference } from "@/api/statestore"
import type { MediaMessageEventContent } from "@/api/types"
import { LeafletPicker } from "../maps/async.tsx"
import { useMediaContent } from "../timeline/content/useMediaContent.tsx"
import type { ComposerState } from "./MessageComposer.tsx"
import CloseIcon from "@/icons/close.svg?react"
import "./MessageComposer.css"

export interface ComposerMediaProps {
	items: MediaMessageEventContent[]
	setState?: (updater: (current: ComposerState) => Partial<ComposerState>) => void
}

export const ComposerMedia = ({ items, setState }: ComposerMediaProps) => {
	const defaultMaxWidth = 360
	const paddingAndButtonWidth = 16 + 40
	const [maxWidth, setMaxWidth] = useState(defaultMaxWidth)
	const containerRef: RefCallback<HTMLDivElement> = elem => {
		setMaxWidth(Math.min(
			(elem?.getBoundingClientRect().width ?? defaultMaxWidth) - paddingAndButtonWidth,
			defaultMaxWidth,
		))
	}
	return <div className="composer-media" ref={containerRef}>
		{items.map(c => <ComposerMediaItem
			key={c.file?.url ?? c.url}
			content={c}
			setState={setState}
			maxWidth={maxWidth}
			hasMultiple={items.length > 1}
		/>)}
		{setState && <>
			<div className="x-separator" />
			<button
				onClick={() => setState(() => ({ media: null }))}
				title={items.length === 1 ? "Remove media" : "Remove all media items"}
			><CloseIcon/></button>
		</>}
	</div>
}

interface ComposerMediaItemProps {
	content: MediaMessageEventContent
	setState?: (updater: (current: ComposerState) => Partial<ComposerState>) => void
	maxWidth: number
	hasMultiple: boolean
}

const ComposerMediaItem = ({ content, setState, maxWidth, hasMultiple }: ComposerMediaItemProps) => {
	const [mediaContent, containerClass, containerStyle] = useMediaContent(
		content, "m.room.message", { height: 120, width: maxWidth },
	)
	const removeMedia = () => {
		setState!(state => ({ media: state.media?.filter(c => c !== content) }))
	}
	return <div className="composer-media-item">
		<div className={`media-container ${containerClass}`} style={containerStyle}>
			{mediaContent}
		</div>
		{setState && hasMultiple && <button
			onClick={removeMedia}
			title="Remove this media item"
		><CloseIcon/></button>}
	</div>
}

export interface ComposerLocationValue {
	lat: number
	long: number
	prec?: number
}

export interface ComposerLocationProps {
	room: RoomStateStore
	client: Client
	location: ComposerLocationValue
	onChange: (location: ComposerLocationValue) => void
	clearLocation: () => void
}

export const ComposerLocation = ({ client, room, location, onChange, clearLocation }: ComposerLocationProps) => {
	const tileTemplate = usePreference(client.store, room, "leaflet_tile_template")
	return <div className="composer-location">
		<div className="location-container">
			<LeafletPicker tileTemplate={tileTemplate} onChange={onChange} initialLocation={location}/>
		</div>
		<button onClick={clearLocation}><CloseIcon/></button>
	</div>
}
