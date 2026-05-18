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
import React, { use, useState } from "react"
import { ScaleLoader } from "react-spinners"
import { RoomStateStore } from "@/api/statestore"
import { HarmCategory, MemDBEvent, TopLevelHarm, UserID, subHarms, topLevelHarms } from "@/api/types"
import ClientContext from "@/ui/ClientContext.ts"
import { ModalCloseContext } from "@/ui/modal/contexts.ts"
import TimelineEvent from "@/ui/timeline/TimelineEvent.tsx"
import { isMobileDevice } from "@/util/ismobile.ts"
import { ensureString, getServerName, isUserID } from "@/util/validation.ts"
import "./ReportModal.css"

interface ReportModalProps {
	evt: MemDBEvent
	room: RoomStateStore
}

interface CommunityReportingSettings {
	community_id: string
	via_user_id: UserID
}

function getCommunityReporting(room: RoomStateStore): CommunityReportingSettings | null {
	const content = room.getStateEvent("org.matrix.msc4468.reporting", "")?.content
	if (!content) {
		return null
	}
	const communityID = ensureString(content.community_id)
	const viaUserID = ensureString(content.via_user_id)
	if (communityID && isUserID(viaUserID)) {
		return {
			community_id: communityID,
			via_user_id: viaUserID,
		}
	}
	return null
}

const useHarmPicker = () => {
	const [topLevel, setTopLevel] = useState<TopLevelHarm>("org.matrix.msc4456.spam")
	const [sub, setSub] = useState("")
	const onSelectTopLevel = (evt: React.ChangeEvent<HTMLSelectElement>) => {
		setTopLevel(evt.currentTarget.value as TopLevelHarm)
		setSub("")
	}
	const onSelectSub = (evt: React.ChangeEvent<HTMLSelectElement>) => {
		setSub(evt.currentTarget.value)
	}
	const content = <div className="harm-picker">
		<label>Category</label>
		<select value={topLevel} onChange={onSelectTopLevel}>
			{Object.entries(topLevelHarms).map(([id, name]) =>
				<option key={id} value={id}>{name}</option>)}
		</select>
		<label>Type</label>
		<select value={sub} onChange={onSelectSub}>
			<option value="">General / Other</option>
			{Object.entries(subHarms[topLevel] ?? {}).map(([id, name]) =>
				<option key={id} value={id}>{name}</option>)}
		</select>
	</div>
	return [content, sub ? `${topLevel}.${sub}` as HarmCategory : topLevel] as const
}

const ReportModal = ({ evt, room }: ReportModalProps) => {
	const client = use(ClientContext)!
	const closeModal = use(ModalCloseContext)
	const [confirming, setConfirming] = useState(false)
	const [reportToOwnServer, setReportToOwnServer] = useState(true)
	const [reportToCommunity, setReportToCommunity] = useState(false)
	const [reportToTargetServer, setReportToTargetServer] = useState(false)
	const [reason, setReason] = useState("")
	const communityReporting = getCommunityReporting(room)
	const ownServer = getServerName(client.userID)
	const targetServer = getServerName(evt.sender)
	const reportingToOtherServerImplemented = false
	const [harmPicker, harm] = useHarmPicker()

	const sendReport = (e: React.SubmitEvent) => {
		e.preventDefault()
		setConfirming(true)
		client.rpc.reportEvent({
			room_id: evt.room_id,
			event_id: evt.event_id,
			reason,
			harm,
			dont_report_to_own_server: !reportToOwnServer,
			report_to_community: reportToCommunity ? communityReporting?.community_id : undefined,
			report_via_user_id: reportToCommunity ? communityReporting?.via_user_id : undefined,
			report_to_other_server: reportToTargetServer ? targetServer : undefined,
		}).catch(err => window.alert(`Failed to send report: ${err}`)).finally(closeModal)
	}

	return <form className="report-modal confirm-message-modal" onSubmit={sendReport}>
		<h3>Report Message</h3>
		{evt ? <div className="timeline-event-container">
			<TimelineEvent evt={evt} prevEvt={null} disableMenu={true} viewType="confirm"/>
		</div> : null}
		<div className="confirm-description">
			Report this message?
		</div>
		<div className="report-destinations">
			<label>
				<input
					type="checkbox"
					checked={reportToOwnServer}
					onChange={evt => setReportToOwnServer(evt.target.checked)}
				/>
				to your server admins ({ownServer})
			</label>
			<label>
				<input
					type="checkbox"
					checked={ownServer === targetServer ? reportToOwnServer : reportToTargetServer}
					onChange={evt => setReportToTargetServer(evt.target.checked)}
					disabled={ownServer === targetServer || !reportingToOtherServerImplemented}
				/>
				to the target's server admins ({targetServer})
			</label>
			<label>
				<input
					type="checkbox"
					checked={reportToCommunity}
					onChange={evt => setReportToCommunity(evt.target.checked)}
					disabled={!communityReporting}
				/>
				to the room moderators ({communityReporting?.via_user_id ?? "not available"})
			</label>
		</div>
		{harmPicker}
		<input
			autoFocus={!isMobileDevice}
			value={reason}
			type="text"
			placeholder="Reason for report"
			onChange={evt => setReason(evt.target.value)}
		/>
		<div className="confirm-buttons">
			{confirming ? <>
				<ScaleLoader barCount={8} color="var(--primary-color)"/>
			</> : <>
				<button type="button" onClick={closeModal}>Cancel</button>
				<button type="submit">Send report</button>
			</>}
		</div>
	</form>
}

export default ReportModal
