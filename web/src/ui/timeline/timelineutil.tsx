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
import { JSX } from "react"
import { MemDBEvent, MemberEventContent } from "@/api/types"
import { Preferences } from "@/api/types/preferences"
import TimelineEvent, { TimelineEventViewType } from "./TimelineEvent.tsx"
import { HiddenEvent, getBodyType } from "./content"

interface renderTimelineListParams {
	focusedEventRowID?: number | null
	prevEventOverride?: MemDBEvent
}

function isHiddenEvent(entry: MemDBEvent): boolean {
	switch (entry.type) {
	case "m.room.server_acl":
		// TODO treat the "all servers are banned" ACL as visible?
		return true
	case "m.room.member": {
		const prevContent = entry.unsigned.prev_content as MemberEventContent
		return prevContent
			&& prevContent.membership === entry.content.membership
			&& prevContent.displayname === entry.content.displayname
			&& prevContent.avatar_url === entry.content.avatar_url
	}
	}
	return getBodyType(entry, Boolean(entry.redacted_by && !entry.viewing_redacted)) === HiddenEvent
}

const unredactableEventTypes = new Set([
	"m.room.power_levels",
	"m.room.create",
	"m.room.member",
])

function shouldHide(entry: MemDBEvent, prefs: Preferences): boolean {
	if (entry.type === "m.room.member") {
		if (!prefs.show_membership_events) {
			return true
		} else if (!prefs.show_profile_changes
			&& (entry.unsigned.prev_content as MemberEventContent)?.membership === entry.content.membership) {
			return true
		}
	}
	if (entry.redacted_by && !prefs.show_redacted_events && !unredactableEventTypes.has(entry.type)) {
		return true
	}
	if ((!prefs.show_hidden_events || entry.sender === "@github:maunium.net") && isHiddenEvent(entry)) {
		return true
	}
	return false
}

export function renderTimelineList(
	viewType: TimelineEventViewType,
	timeline: (MemDBEvent | null)[],
	prefs: Preferences,
	{ focusedEventRowID, prevEventOverride }: renderTimelineListParams = {},
): (JSX.Element | null)[] {
	let prevEvt: MemDBEvent | null = prevEventOverride ?? null
	let receiptMergeIdx: number | null = null
	const flattenedPrefs = {
		small_replies: prefs.small_replies,
		small_threads: viewType !== "thread" && prefs.small_threads,
		display_read_receipts: prefs.display_read_receipts,
		show_membership_events: prefs.show_membership_events,
		show_profile_changes: prefs.show_profile_changes,
		show_redacted_events: prefs.show_redacted_events,
		show_hidden_events: prefs.show_hidden_events,
	}
	return timeline.map(entry => {
		if (!entry) {
			return null
		} else if (shouldHide(entry, flattenedPrefs)) {
			if (prevEvt && viewType === "timeline" && flattenedPrefs.display_read_receipts) {
				// Completely pointless optimization to avoid recreating the receipt_flattening array on every render
				if (!prevEvt.receipt_flattening) {
					prevEvt.receipt_flattening = [entry.event_id]
					receiptMergeIdx = null
				} else if (receiptMergeIdx === null) {
					prevEvt.receipt_flattening.push(entry.event_id)
				} else if (prevEvt.receipt_flattening[receiptMergeIdx] === entry.event_id) {
					receiptMergeIdx += 1
				} else {
					prevEvt.receipt_flattening = prevEvt.receipt_flattening.slice(0, receiptMergeIdx)
					prevEvt.receipt_flattening.push(entry.event_id)
					receiptMergeIdx = null
				}
			}
			return null
		}
		if (
			prevEvt?.receipt_flattening
			&& receiptMergeIdx !== null
			&& prevEvt.receipt_flattening.length > receiptMergeIdx
		) {
			prevEvt.receipt_flattening = prevEvt.receipt_flattening.slice(0, receiptMergeIdx)
		}
		const thisEvt = <TimelineEvent
			key={entry.rowid}
			evt={entry}
			prevEvt={prevEvt}
			smallReplies={flattenedPrefs.small_replies}
			smallThreads={flattenedPrefs.small_threads}
			isFocused={focusedEventRowID === entry.rowid}
			viewType={viewType}
		/>
		prevEvt = entry
		receiptMergeIdx = 0
		return thisEvt
	})
}
