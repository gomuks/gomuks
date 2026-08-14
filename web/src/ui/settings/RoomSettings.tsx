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
import { use, useState } from "react"
import { MoonLoader } from "react-spinners"
import { getRoomAvatarThumbnailURL, getRoomAvatarURL } from "@/api/media.ts"
import { RoomStateStore, useRoomState } from "@/api/statestore"
import { JoinRule, JoinRulesEventContent, RoomType } from "@/api/types"
import { preferences } from "@/api/types/preferences"
import { useEventAsState } from "@/util/eventdispatcher.ts"
import { getEventLevel } from "@/util/powerlevel.ts"
import { ensureString } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import { getPowerLevels } from "../menu/util.ts"
import { LightboxContext, ModalCloseContext, ModalContext, modals } from "../modal"

interface RoomSettingsProps {
	room: RoomStateStore
}

const historyVisibilityOptions = [
	{ key: "world_readable", name: "Everyone (world readable)" },
	{ key: "shared", name: "All members" },
	{ key: "invited", name: "Members since invited" },
	{ key: "joined", name: "Members since joined" },
]
const validHistoryVisibility = historyVisibilityOptions.map(o => o.key)

const joinRulesOptions = [
	{ key: "public", name: "Public" },
	{ key: "knock_restricted", name: "Knock + Restricted" },
	{ key: "restricted", name: "Restricted" },
	{ key: "knock", name: "Knock" },
	{ key: "invite", name: "Invite-only" },
]
const validJoinRules = joinRulesOptions.map(o => o.key)

function joinRulesAreEqual(a: JoinRulesEventContent, b: JoinRulesEventContent | null): boolean {
	if (!b || a.join_rule !== b.join_rule || a.allow?.length !== b.allow?.length) {
		return false
	}
	return !a.allow?.find((aItem, idx) => {
		const bItem = b.allow![idx]
		return aItem.type !== bItem.type || aItem.room_id !== bItem.room_id
	})
}

function stringifyUnknown(val: unknown): string {
	if (typeof val === "string") {
		return val || "empty string"
	}
	return String(val)
}

const RoomSettings = ({ room }: RoomSettingsProps) => {
	const roomMeta = useEventAsState(room?.meta)
	const client = use(ClientContext)!
	const closeModal = use(ModalCloseContext)
	const openModal = use(ModalContext)

	useRoomState(room, "m.room.power_levels")
	const [pls, ownPL] = getPowerLevels(room, client)
	const canChangeName = ownPL >= getEventLevel(pls, "m.room.name", true)
	const canChangeTopic = ownPL >= getEventLevel(pls, "m.room.topic", true)
	const canChangeHistoryVisibility = ownPL >= getEventLevel(pls, "m.room.history_visibility", true)
	const canChangeJoinRules = ownPL >= getEventLevel(pls, "m.room.join_rules", true)
	const joinRules = useRoomState(room, "m.room.join_rules")?.content as JoinRulesEventContent | null
	const historyVisibility = ensureString(useRoomState(room, "m.room.history_visibility")?.content?.history_visibility)

	const [newName, setNewName] = useState<string>()
	const [newTopic, setNewTopic] = useState<string>()
	const [newHistoryVisibility, setNewHistoryVisibility] = useState<string>()
	const [newJoinRules, setNewJoinRules] = useState<JoinRulesEventContent>()
	const [loading, setLoading] = useState(false)

	const onClickLeave = () => {
		if (room && window.confirm(`Really leave ${room.meta.current.name}?`)) {
			client.rpc.leaveRoom(room.roomID).then(
				() => {
					console.info("Successfully left", room.roomID)
					closeModal()
				},
				err => window.alert(`Failed to leave room: ${err}`),
			)
		}
	}
	const openDevtools = () => {
		if (room) {
			openModal(modals.roomStateExplorer(room))
		}
	}
	const previousRoomID = roomMeta?.creation_content?.predecessor?.room_id
	const openPredecessorRoom = () => {
		window.mainScreenContext.setActiveRoom(previousRoomID!)
		closeModal()
	}
	const undoChanges = () => {
		setNewName(undefined)
		setNewTopic(undefined)
		setNewHistoryVisibility(undefined)
		setNewJoinRules(undefined)
	}
	const saveChanges = () => {
		if (loading) {
			return
		}
		const promises = []
		if (newName !== undefined && newName !== roomMeta.name) {
			promises.push(client.rpc.setState(room.roomID, "m.room.name", "", { name: newName }))
		}
		if (newTopic !== undefined && newTopic !== roomMeta.topic) {
			promises.push(client.rpc.setState(room.roomID, "m.room.topic", "", { topic: newTopic }))
		}
		if (newHistoryVisibility !== undefined && newHistoryVisibility !== historyVisibility) {
			promises.push(client.rpc.setState(room.roomID, "m.room.history_visibility", "", {
				history_visibility: newHistoryVisibility,
			}))
		}
		if (newJoinRules !== undefined && !joinRulesAreEqual(newJoinRules, joinRules)) {
			promises.push(client.rpc.setState(room.roomID, "m.room.join_rules", "", newJoinRules))
		}
		if (promises.length) {
			setLoading(true)
			Promise.all(promises).then(
				() => {},
				err => {
					console.error("Failed to save room info", err)
					window.alert(`Failed to save room info: ${err}`)
				},
			).finally(() => setLoading(false))
		}
	}
	const anythingChanged = (newName !== undefined && newName !== roomMeta.name) ||
		(newTopic !== undefined && newTopic !== roomMeta.topic) ||
		(newHistoryVisibility !== undefined && newHistoryVisibility !== historyVisibility) ||
		(newJoinRules !== undefined && !joinRulesAreEqual(newJoinRules, joinRules))
	const editJoinRule = newJoinRules?.join_rule ?? joinRules?.join_rule
	return <div className="room-details">
		<div className="header">
			<h2>Room info</h2>
			<code className="room-id">{room.roomID}</code>
		</div>
		<div className="general-info">
			<img
				className="avatar large"
				loading="lazy"
				src={getRoomAvatarThumbnailURL(roomMeta)}
				data-full-src={getRoomAvatarURL(roomMeta)}
				onClick={use(LightboxContext)}
				alt=""
			/>
			<input
				className="room-name-input"
				type="text"
				placeholder="Room name"
				value={newName ?? roomMeta.name ?? ""}
				onChange={evt => setNewName(evt.target.value)}
				disabled={!canChangeName}
			/>
			<textarea
				className="room-topic-input"
				placeholder="Room topic"
				value={newTopic ?? roomMeta.topic ?? ""}
				onChange={evt => setNewTopic(evt.target.value)}
				disabled={!canChangeTopic}
			/>
		</div>
		<div className="boxes">
			<div className="history-visibility">
				<h4>History visibility</h4>
				{historyVisibilityOptions.map(({ key, name }) => <label key={key}>
					<input
						type="radio"
						name="history-visibility"
						value={key}
						checked={(newHistoryVisibility ?? historyVisibility) === key}
						disabled={!canChangeHistoryVisibility}
						onChange={evt => setNewHistoryVisibility(evt.target.value)}
					/>
					{name}
				</label>)}
				{!validHistoryVisibility.includes(newHistoryVisibility ?? historyVisibility ?? "") && <label>
					<input type="radio" name="history-visibility" checked disabled />
					<code>{stringifyUnknown(newHistoryVisibility ?? historyVisibility)}</code>
				</label>}
			</div>
			<div className="join-rules">
				<h4>Join rules</h4>
				{joinRulesOptions.map(({ key, name }) => <label key={key}>
					<input
						type="radio"
						name="join-rules"
						value={key}
						checked={editJoinRule === key}
						disabled={!canChangeJoinRules}
						onChange={evt => setNewJoinRules({
							...(newJoinRules ?? joinRules),
							join_rule: evt.target.value as JoinRule,
						})}
					/>
					{name}
				</label>)}
				{!validJoinRules.includes(editJoinRule ?? "") && <label>
					<input type="radio" name="join-rules" checked disabled={!canChangeJoinRules} />
					<code>{stringifyUnknown(editJoinRule)}</code>
				</label>}
			</div>
			{editJoinRule === "restricted" || editJoinRule === "knock_restricted" ? <div className="join-rule-spaces">
				<h4>Allowed spaces</h4>
				{(newJoinRules?.allow ?? joinRules?.allow)?.map(allowItem =>
					allowItem.type === "m.room_membership"
						? <code>{allowItem.room_id}</code>
						: <code>{allowItem.type}</code>,
				)}
			</div> : null}
		</div>
		<div className="save-buttons">
			<button className="cancel-button" disabled={!anythingChanged} onClick={undoChanges}>
				Cancel
			</button>
			<button className="save-button" disabled={!anythingChanged} onClick={saveChanges}>
				{loading ? <MoonLoader size={16} /> : "Save"}
			</button>
		</div>
		<div className="room-buttons">
			<button className="devtools" onClick={openDevtools}>Open devtools</button>
			<select onChange={evt => {
				window.activeRoomContext?.setForceViewType(evt.target.value as RoomType)
				closeModal()
			}} defaultValue="__null__">
				{preferences.room_view_type.allowedValues!.map((val, i) =>
					<option key={i} value={val ?? "__null__"} disabled={i === 0}>
						{i === 0 ? "Override view" : preferences.room_view_type.valueLabels![i]}
					</option>)}
			</select>
			{previousRoomID &&
				<button className="previous-room" onClick={openPredecessorRoom}>
					Open Predecessor Room
				</button>}
		</div>
		<h2>Leave room</h2>
		<button className="leave-room" onClick={onClickLeave}>Leave room</button>
	</div>
}

export default RoomSettings
