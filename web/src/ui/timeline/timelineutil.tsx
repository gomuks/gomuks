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
import { JSX } from "react"
import { MemDBEvent } from "@/api/types"
import TimelineEvent, { TimelineEventViewType } from "./TimelineEvent.tsx"

interface renderTimelineListParams {
	smallReplies?: boolean
	focusedEventRowID?: number | null
	prevEventOverride?: MemDBEvent
}

export function renderTimelineList(
	viewType: TimelineEventViewType,
	timeline: (MemDBEvent | null)[],
	{ smallReplies, focusedEventRowID, prevEventOverride }: renderTimelineListParams,
): (JSX.Element | null)[] {
	let prevEvt: MemDBEvent | null = prevEventOverride ?? null
	return timeline.map(entry => {
		if (!entry) {
			return null
		}
		const thisEvt = <TimelineEvent
			key={entry.rowid}
			evt={entry}
			prevEvt={prevEvt}
			smallReplies={smallReplies}
			isFocused={focusedEventRowID === entry.rowid}
			viewType={viewType}
		/>
		prevEvt = entry
		return thisEvt
	})
}
