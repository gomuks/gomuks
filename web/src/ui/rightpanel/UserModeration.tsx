// gomuks - A Matrix client written in Go.
// Copyright (C) 2025 Nexus Nicholson
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
import React, { JSX, use, useState } from "react"
import { MoonLoader } from "react-spinners"
import Client from "@/api/client.ts"
import { RoomStateStore, useRoomState } from "@/api/statestore"
import { MemDBEvent, MembershipAction, PowerLevelEventContent } from "@/api/types"
import { getEventLevel, getUserLevel } from "@/util/powerlevel.ts"
import { getPowerLevels } from "../menu/util.ts"
import { BulkRedactModal, ConfirmWithMessageModal, ModalContext } from "../modal"
import StartDMButton from "./StartDMButton.tsx"
import UserIgnoreButton from "./UserIgnoreButton.tsx"
import CheckIcon from "@/icons/check.svg?react"
import DeleteIcon from "@/icons/delete.svg?react"
import BanIcon from "@/icons/gavel.svg?react"
import InviteIcon from "@/icons/person-add.svg?react"
import KickIcon from "@/icons/person-remove.svg?react"
import PowerLevelIcon from "@/icons/shield-person.svg?react"

interface UserModerationProps {
	userID: string;
	client: Client;
	room: RoomStateStore | undefined;
	member: MemDBEvent | null;
}

const powerSlider = !!localStorage.gomuks_power_slider

const UserModeration = ({ userID, client, member, room }: UserModerationProps) => {
	const openModal = use(ModalContext)
	const [redactRemaining, setRedactRemaining] = useState<number>(0)
	const [modifiedPL, setModifiedPL] = useState<number | null>(null)
	const [powerLoading, setPowerLoading] = useState<boolean>(false)
	useRoomState(room, "m.room.power_levels")
	if (!room) {
		return makeNonRoomUserActions(client, userID)
	}
	const [pls, ownPL, createEvent] = getPowerLevels(room, client)
	const otherUserPL = getUserLevel(pls, createEvent, userID)
	const hasPL = (action: "invite" | "kick" | "ban" | "redact") => {
		return ownPL >= (pls[action] ?? (action === "invite" ? 0 : 50))
			&& (action === "redact" || action === "invite" || ownPL > otherUserPL)
	}

	const runAction = (action: MembershipAction) => {
		const callback = (reason: string, redact?: boolean) =>
			client.rpc.setMembership(room.roomID, userID, action, reason, action == "ban" && redact)
		let content: JSX.Element
		if (action == "ban") {
			const [eligibleEventsCount, nonStateEventsCount, redactCallback] = makeRecentMessageRedactor(callback)
			content = <BulkRedactModal
				userID={userID}
				evtCount={eligibleEventsCount}
				nonStateEvtCount={nonStateEventsCount}
				isBanModal={true}
				onConfirm={redactCallback}
			/>
		} else {
			const titleCasedAction = action.charAt(0).toUpperCase() + action.slice(1)
			content = <ConfirmWithMessageModal
				title={`${titleCasedAction} user`}
				description={<>Are you sure you want to {action} <code>{userID}</code>?</>}
				placeholder="Reason (optional)"
				confirmButton={titleCasedAction}
				onConfirm={callback}
			/>
		}
		return () => {
			openModal({
				dimmed: true,
				boxed: true,
				content,
			})
		}
	}
	const calculateRedactions = () => {
		return room.timelineCache.filter((evt): evt is MemDBEvent =>
			evt !== null && evt.room_id == room.roomID && evt.sender === userID && !evt.redacted_by)
	}
	const makeRecentMessageRedactor = (banCallback?: (reason: string, redact?: boolean) => Promise<unknown>) => {
		const eligibleEvents = calculateRedactions()
		const nonStateEvents = eligibleEvents.filter(evt => evt.state_key === undefined)
		const callback = async (doRedact: boolean, preserveState: boolean, reason: string) => {
			await banCallback?.(reason, doRedact)
			if (!doRedact) {
				return
			}
			const targetEvents = preserveState ? nonStateEvents : eligibleEvents
			let toRedact = targetEvents.length
			setRedactRemaining(toRedact)
			for (const evt of targetEvents) {
				try {
					await client.rpc.redactEvent(evt.room_id, evt.event_id, reason)
					toRedact--
					setRedactRemaining(toRedact)
					console.debug(`Redacted ${evt.event_id} (${toRedact} remaining)`)
				} catch (e) {
					console.error(`Failed to redact ${evt.event_id}:`, e)
					throw e
				}
			}
			return true
		}
		return [eligibleEvents.length, nonStateEvents.length, callback] as const
	}
	const openRedactRecentModal = () => {
		const [eligibleEventsCount, nonStateEventsCount, callback] = makeRecentMessageRedactor()
		openModal({
			dimmed: true,
			boxed: true,
			content: <BulkRedactModal
				userID={userID}
				evtCount={eligibleEventsCount}
				nonStateEvtCount={nonStateEventsCount}
				isBanModal={false}
				onConfirm={callback}
			/>,
		})
	}
	const onClickSavePL = () => {
		if (modifiedPL === null) {
			return
		}
		const powerCopy: PowerLevelEventContent = { ...pls }
		if (modifiedPL === (pls.users_default ?? 0)) {
			if (powerCopy.users) {
				delete powerCopy.users[userID]
			}
		} else {
			powerCopy.users = {
				...(pls.users ?? {}),
				[userID]: modifiedPL,
			}
		}
		setPowerLoading(true)
		client.rpc.setState(room.roomID, "m.room.power_levels", "", powerCopy).then(
			() => console.info("Successfully set power level of", userID, "to", modifiedPL),
			err => window.alert(`Failed to set power level: ${err}`),
		).finally(() => setPowerLoading(false))
	}
	const onChangePL = (evt: React.ChangeEvent<HTMLInputElement>) => {
		let newPL: number | null = evt.target.valueAsNumber
		if (isNaN(newPL)) {
			return
		} else if (newPL > ownPL) {
			newPL = ownPL
		} else if (newPL > Number.MAX_SAFE_INTEGER) {
			newPL = Number.MAX_SAFE_INTEGER
		} else if (newPL < Number.MIN_SAFE_INTEGER) {
			newPL = Number.MIN_SAFE_INTEGER
		}
		setModifiedPL(newPL)
	}
	const onPLKeyDown = (evt: React.KeyboardEvent<HTMLInputElement>) => {
		if (evt.key === "Escape") {
			setModifiedPL(null)
			evt.currentTarget.blur()
			evt.stopPropagation()
		}
	}

	const membership = member?.content.membership || "leave"
	const isCreator = otherUserPL === Infinity
	const hasPLPL = ownPL >= getEventLevel(pls, "m.room.power_levels", true)
		&& !isCreator
		&& (ownPL > otherUserPL || pls.users?.[userID] === undefined || userID === client.userID)
	return <div className="user-moderation">
		<h4>Actions</h4>
		<div className="moderation-action">
			<PowerLevelIcon />
			<input
				type={isCreator ? "text" : powerSlider ? "range" : "number"}
				value={isCreator ? "Infinity" : modifiedPL ?? otherUserPL}
				max={Math.min(ownPL, Number.MAX_SAFE_INTEGER)}
				min={Number.MIN_SAFE_INTEGER}
				disabled={!hasPLPL || powerLoading}
				onChange={onChangePL}
				onMouseUp={powerSlider ? onClickSavePL : undefined}
				onKeyDown={onPLKeyDown}
				title="Power level"
			/>
			{modifiedPL !== null && modifiedPL !== otherUserPL && <button
				disabled={!hasPLPL || powerLoading}
				onClick={onClickSavePL}
				title="Save power level"
			>{powerLoading ? <MoonLoader size={16} /> : <CheckIcon />}</button>}
		</div>
		{room.meta.current.dm_user_id !== userID && client.userID !== userID ?
			<StartDMButton userID={userID} client={client} /> : null}
		{(["knock", "leave"].includes(membership) || !member) && hasPL("invite") && (
			<button className="moderation-action positive" onClick={runAction("invite")}>
				<InviteIcon />
				<span>{membership === "knock" ? "Accept join request" : "Invite"}</span>
			</button>
		)}
		{["knock", "invite", "join"].includes(membership) && hasPL("kick") && (
			<button className="moderation-action dangerous" onClick={runAction("kick")}>
				<KickIcon />
				<span>{
					membership === "join"
						? "Kick"
						: membership === "invite"
							? "Revoke invitation"
							: "Reject join request"
				}</span>
			</button>
		)}
		{hasPL("ban") && (membership === "ban" ? (
			<button className="moderation-action positive" onClick={runAction("unban")}>
				<BanIcon />
				<span>Unban</span>
			</button>
		) : (
			<button className="moderation-action dangerous" onClick={runAction("ban")}>
				<BanIcon />
				<span>Ban</span>
			</button>
		))}
		{ownPL >= getEventLevel(pls, "m.room.redaction") && (hasPL("redact") || userID === client.userID) && (
			<button
				className="moderation-action dangerous"
				onClick={openRedactRecentModal}
				disabled={redactRemaining > 0}
			>
				<DeleteIcon />
				<span>{redactRemaining > 0 ? `${redactRemaining} remaining`: "Redact recent messages"}</span>
			</button>
		)}
		<UserIgnoreButton userID={userID} client={client} />
	</div>
}

const makeNonRoomUserActions = (client: Client, userID: string) => {
	return <div className="user-moderation">
		<h4>Actions</h4>
		<StartDMButton userID={userID} client={client} />
	</div>
}

export default UserModeration
