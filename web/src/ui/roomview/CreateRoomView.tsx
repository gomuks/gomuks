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
import React, { Fragment, use, useEffect, useState } from "react"
import { RoomStateStore, fakeGomuksSender } from "@/api/statestore"
import { CreateRoomInitialState, RoomNameQuality, RoomPreset, RoomVersion, UserID } from "@/api/types"
import { getEventLevel, preV12 } from "@/util/powerlevel.ts"
import { getServerName } from "@/util/validation"
import ClientContext from "../ClientContext"
import MainScreenContext from "../MainScreenContext"
import { getPowerLevels } from "../menu/util.ts"
import { ModalCloseContext } from "../modal"
import AddIcon from "@/icons/add.svg?react"
import CloseIcon from "@/icons/close.svg?react"
import InviteIcon from "@/icons/person-add.svg?react"
import "./CreateRoomView.css"

interface initialStateEntry {
	type: string
	stateKey: string
	content: string
}

function tryParseJSON(json: string) {
	try {
		return JSON.parse(json)
	} catch {
		return undefined
	}
}

const migrateAliases = "__migrate__"

const CreateRoomView = ({ oldRoom }: { oldRoom?: RoomStateStore }) => {
	const client = use(ClientContext)!
	const closeModal = use(ModalCloseContext)
	const mainScreen = use(MainScreenContext)
	const [loading, setLoading] = useState(false)
	const [error, setError] = useState("")
	const [preset, setPreset] = useState<RoomPreset>("private_chat")
	const [name, setName] = useState("")
	const [topic, setTopic] = useState("")
	const [aliasLocalpart, setAliasLocalpart] = useState("")
	const [invite, setInvite] = useState<UserID[]>([])
	const [isDirect, setIsDirect] = useState(false)
	const [isEncrypted, setIsEncrypted] = useState(true)
	const [initialState, setInitialState] = useState<initialStateEntry[]>([])
	const [roomVersion, setRoomVersion] = useState<RoomVersion | "">("12")
	const [roomID, setRoomID] = useState("")
	const [roomCreateTS, setRoomCreateTS] = useState<number>(0)
	const [creationContent, setCreationContent] = useState<string>("{\n\n}")
	const [powerLevelContentOverride, setPowerLevelContentOverride] = useState<string>("{\n\n}")
	const [sendTombstone, setSendTombstone] = useState(true)

	useEffect(() => {
		if (!oldRoom) {
			return
		}
		const lastEvtID = oldRoom.timelineCache.findLast(evt =>
			evt && evt.sender !== fakeGomuksSender && evt.event_id.startsWith("$"),
		)?.event_id
		setCreationContent(JSON.stringify({
			...oldRoom.getStateEvent("m.room.create", "")?.content,
			predecessor: {
				room_id: oldRoom.roomID,
				event_id: lastEvtID,
			},
			room_version: undefined,
			creator: undefined,
		}, null, 4))
		setName(oldRoom.meta.current.name_quality === RoomNameQuality.Explicit ? oldRoom.meta.current.name ?? "" : "")
		setTopic(oldRoom.meta.current.topic ?? "")
		setIsDirect(!!oldRoom.meta.current.dm_user_id)
		setIsEncrypted(!!oldRoom.meta.current.encryption_event)
		const joinRules = oldRoom.getStateEvent("m.room.join_rules", "")?.content
		const initialState = []
		if (oldRoom.meta.current.dm_user_id) {
			setPreset("trusted_private_chat")
			setInvite([oldRoom.meta.current.dm_user_id])
		} else if (joinRules?.join_rule === "public") {
			setPreset("public_chat")
		} else if (
			(joinRules?.join_rule === "restricted" || joinRules?.join_rule === "knock_restricted")
			&& Array.isArray(joinRules.allow)
		) {
			setPreset("private_chat")
			const newAllow = [...joinRules.allow]
			if (!newAllow.some(allow => allow.type === "fi.mau.spam_checker")) {
				newAllow.push({
					type: "m.room_membership",
					room_id: oldRoom.roomID,
				})
			}
			initialState.push({
				type: "m.room.join_rules",
				stateKey: "",
				content: JSON.stringify({
					join_rule: joinRules.join_rule,
					allow: newAllow,
				}),
			})
		} else {
			setPreset("private_chat")
		}
		// eslint-disable-next-line prefer-const
		let [plContent, ownPL] = getPowerLevels(oldRoom, client)
		setSendTombstone(ownPL >= getEventLevel(plContent, "m.room.tombstone", true))
		if (ownPL >= getEventLevel(plContent, "m.room.canonical_alias", true)) {
			setAliasLocalpart(migrateAliases)
		}
		if (plContent?.users && client.userID in plContent.users) {
			const users = { ...plContent.users }
			delete users[client.userID]
			plContent = { ...plContent, users }
		}
		setPowerLevelContentOverride(JSON.stringify(plContent, null, 4))
		if (oldRoom.meta.current.explicit_avatar && oldRoom.meta.current.avatar) {
			initialState.push({
				type: "m.room.avatar",
				stateKey: "",
				content: JSON.stringify({
					url: oldRoom.meta.current.avatar,
				}),
			})
		}
		setInitialState(initialState)
	}, [oldRoom, client])

	const updatePreset = (newPreset: RoomPreset) => {
		if (newPreset === "public_chat") {
			setIsEncrypted(false)
		}
		setPreset(newPreset)
	}

	const setDMConfig = () => {
		if (invite.length === 0) {
			setInvite([""])
		}
		setIsDirect(true)
		setIsEncrypted(true)
		setPreset("trusted_private_chat")
		setCreationContent(JSON.stringify({
			...tryParseJSON(creationContent),
			type: undefined,
		}, null, 4))
		setPowerLevelContentOverride(JSON.stringify({
			...tryParseJSON(powerLevelContentOverride),
			events_default: undefined,
		}, null, 4))
	}
	const setSpaceConfig = () => {
		setCreationContent(JSON.stringify({
			...tryParseJSON(creationContent),
			type: "m.space",
		}, null, 4))
		setPowerLevelContentOverride(JSON.stringify({
			...tryParseJSON(powerLevelContentOverride),
			events_default: 100,
		}, null, 4))
		setIsDirect(false)
		setIsEncrypted(false)
		setPreset(preset === "public_chat" ? "public_chat" : "private_chat")
	}

	const isRoomV12 = !preV12.has(roomVersion)
	const onSubmit = (evt: React.SubmitEvent<HTMLFormElement>) => {
		if (sendTombstone && oldRoom && !window.confirm("Create room and send a tombstone to the old one?")) {
			return
		}
		let creation_content, power_level_content_override
		try {
			creation_content = JSON.parse(creationContent)
		} catch (err) {
			setError(`Failed to parse creation content: ${err}`)
			return
		}
		try {
			power_level_content_override = JSON.parse(powerLevelContentOverride)
		} catch (err) {
			setError(`Failed to parse power level content override: ${err}`)
			return
		}
		let reqInitialState: CreateRoomInitialState[]
		try {
			reqInitialState = initialState.filter(state => state.type && state.content).map(state => ({
				type: state.type,
				state_key: state.stateKey,
				content: JSON.parse(state.content),
			}))
		} catch (err) {
			setError(`Failed to parse initial state: ${err}`)
			return
		}
		evt.preventDefault()
		setLoading(true)
		setError("")
		if (isEncrypted) {
			reqInitialState.push({
				type: "m.room.encryption",
				content: {
					algorithm: "m.megolm.v1.aes-sha2",
				},
			})
		}
		const room_alias_name = aliasLocalpart && aliasLocalpart !== migrateAliases ? aliasLocalpart : undefined
		client.rpc.createRoom({
			name: name || undefined,
			topic: topic || undefined,
			room_alias_name,
			preset,
			is_direct: isDirect,
			invite: invite.filter(id => !!id),
			initial_state: reqInitialState,
			room_version: roomVersion || undefined,
			creation_content,
			power_level_content_override,
			"fi.mau.room_id": !isRoomV12 || roomCreateTS ? roomID || undefined : undefined,
			"fi.mau.origin_server_ts": roomCreateTS || undefined,
		}).then(resp => {
			closeModal()
			console.log("Created room:", resp.room_id)

			if (aliasLocalpart === migrateAliases && oldRoom) {
				const canon = oldRoom.getStateEvent("m.room.canonical_alias", "")?.content
				client.rpc.getLocalAliases(oldRoom.roomID).then(async aResp => {
					console.log("Migrating aliases:", aResp.aliases)
					await Promise.all(aResp.aliases.map(alias => client.rpc.createAlias(alias, resp.room_id, true)))
					if (canon) {
						console.log("Re-sending canonical alias event")
						await client.rpc.setState(resp.room_id, "m.room.canonical_alias", "", canon)
					}
				}).catch(err => console.error("Failed to migrate local aliases:", err))
			}
			if (sendTombstone && oldRoom) {
				client.rpc.setState(oldRoom.roomID, "m.room.tombstone", "", {
					body: "This room has been replaced",
					replacement_room: resp.room_id,
				}).then(
					() => console.log("Sent tombstone to", oldRoom.roomID),
					err => console.error("Failed to send tombstone to old room:", err),
				)
				const oldPowers = oldRoom.getStateEvent("m.room.power_levels", "")?.content
				if (oldPowers && (oldPowers.events_default ?? 0) < 100) {
					client.rpc.setState(oldRoom.roomID, "m.room.power_levels", "", {
						...oldPowers,
						events: {
							...oldPowers,
							"m.room.message": undefined,
							"m.reaction": undefined,
							"m.room.encrypted": undefined,
							"m.sticker": undefined,
						},
						events_default: 100,
					}).then(
						() => console.log("Sent tombstone to", oldRoom.roomID),
						err => console.error("Failed to send tombstone to old room:", err),
					)
				}
			}

			// FIXME this is a hacky way to work around the room taking time to come down /sync
			setTimeout(() => {
				mainScreen.setActiveRoom(resp.room_id)
			}, 1000)
		}, err => {
			setError(`${err}`.replace(/^Error: /, ""))
			setLoading(false)
		})
	}

	const serverName = getServerName(client.store.userID)
	useEffect(() => {
		if (!isRoomV12 || !roomCreateTS) {
			return
		}
		//eslint-disable-next-line @typescript-eslint/no-explicit-any
		let creationJSON: any
		try {
			creationJSON = JSON.parse(creationContent)
		} catch (err) {
			console.error("Failed to parse creation content:", err)
			return
		}
		creationJSON.room_version = roomVersion
		const timeout = setTimeout(() => {
			client.rpc.calculateRoomID(roomCreateTS, creationJSON).then(
				roomID => setRoomID(roomID),
				err => {
					console.error("Failed to calculate room ID:", err)
					setError(`Failed to calculate room ID: ${err}`)
				},
			)
		}, 500)
		return () => clearTimeout(timeout)
	}, [client, roomVersion, isRoomV12, roomCreateTS, creationContent])

	return <form className="create-room-view" onSubmit={onSubmit}>
		<h2>Create a new room</h2>

		<div className="form-fields">
			<div className="room-templates">
				<span>Apply template:</span>
				<button type="button" onClick={setDMConfig}>DM</button>
				<span>/</span>
				<button type="button" onClick={setSpaceConfig}>Space</button>
			</div>

			<label htmlFor="room-create-name" title="The name of the room">Name</label>
			<input
				id="room-create-name"
				type="text"
				placeholder="Meow room"
				value={name}
				onChange={e => setName(e.target.value)}
			/>
			<label htmlFor="room-create-topic" title="A short description of the room">Topic</label>
			<input
				id="room-create-topic"
				type="text"
				placeholder="A room for meowing"
				value={topic}
				onChange={e => setTopic(e.target.value)}
			/>
			<label htmlFor="room-create-alias" title="The alias for the room">Alias</label>
			<label className="room-alias-container">
				#
				<input
					id="room-create-alias"
					type="text"
					placeholder="meow"
					value={aliasLocalpart}
					onChange={e => setAliasLocalpart(e.target.value)}
				/>
				:{serverName}
			</label>
			<label htmlFor="room-create-encrypted" title="Whether the room is encrypted">
				Encrypted
			</label>
			<input
				id="room-create-encrypted"
				type="checkbox"
				checked={isEncrypted}
				onChange={e => setIsEncrypted(e.target.checked)}
			/>
			{oldRoom && <>
				<label htmlFor="room-create-tombstone" title="Whether to send a tombstone to the old room">
					Tombstone old room
				</label>
				<input
					id="room-create-tombstone"
					type="checkbox"
					checked={sendTombstone}
					onChange={e => setSendTombstone(e.target.checked)}
				/>
			</>}

			<label htmlFor="room-create-preset" title="Preset for join rules and history visibility">Preset</label>
			<select id="room-create-preset" value={preset} onChange={e => updatePreset(e.target.value as RoomPreset)}>
				<option value="public_chat">Public chat</option>
				<option value="private_chat">Private chat</option>
				<option value="trusted_private_chat">Trusted private chat</option>
			</select>
		</div>
		<div className="form-fields item-list" id="room-create-invite">
			<div className="item-list-header">
				Users to invite
				<button
					className="item-list-add"
					type="button"
					onClick={() => setInvite([...invite, ""])}
				><InviteIcon /></button>
			</div>
			{invite.map((id, index) => {
				const onChange = (e: React.ChangeEvent<HTMLInputElement>) =>
					setInvite([...invite.slice(0, index), e.target.value, ...invite.slice(index + 1)])
				const onRemove = () => setInvite([...invite.slice(0, index), ...invite.slice(index + 1)])
				return <Fragment key={index}>
					<input
						className="item-list-item"
						type="text"
						placeholder={`@user:${serverName}`}
						value={id}
						onChange={onChange}
					/>
					<button
						className="item-list-remove"
						type="button"
						onClick={onRemove}
					><CloseIcon /></button>
				</Fragment>
			})}
		</div>
		<details>
			<summary>Advanced options</summary>
			<div className="form-fields">
				<label htmlFor="room-create-is-direct" title="Whether the room is a direct chat">
					Direct chat
				</label>
				<input
					id="room-create-is-direct"
					type="checkbox"
					checked={isDirect}
					onChange={e => setIsDirect(e.target.checked)}
				/>
				<label
					htmlFor="room-create-version"
					title="The version of the room to create. If unset, the server will decide"
				>
					Room version
				</label>
				<input
					id="room-create-version"
					type="text"
					placeholder="server default"
					value={roomVersion}
					onChange={e => setRoomVersion(e.target.value as RoomVersion)}
				/>
				{isRoomV12 ? <>
					<label
						htmlFor="room-create-ts"
						title="Custom room creation timestamp. Only works if supported by the server."
					>
						Create timestamp
					</label>
					<input
						id="room-create-ts"
						type="number"
						placeholder={Date.now().toString()}
						value={roomCreateTS ? roomCreateTS.toString() : ""}
						onChange={e => setRoomCreateTS(+e.target.value)}
					/>
				</> : null}
				{!isRoomV12 || roomCreateTS ? <>
					<label htmlFor="room-create-id" title="Custom room ID. Only works if supported by the server.">
						Room ID
					</label>
					<input
						id="room-create-id"
						type="text"
						placeholder={`!meow:${serverName}`}
						value={roomID}
						onChange={e => setRoomID(e.target.value)}
						disabled={isRoomV12}
					/>
				</> : null}
				<label htmlFor="room-create-power-level-override" title="Override power levels in the room">
					Power level override
				</label>
				<textarea
					id="room-create-power-level-override"
					value={powerLevelContentOverride}
					onChange={e => setPowerLevelContentOverride(e.target.value)}
					rows={5}
				/>
				<label htmlFor="room-create-creation-content" title="Override the creation content of the room">
					Creation content
				</label>
				<textarea
					id="room-create-creation-content"
					value={creationContent}
					onChange={e => setCreationContent(e.target.value)}
					rows={3}
				/>
			</div>
			<div className="form-fields item-list state-event-list" id="room-create-initial-state">
				<div className="item-list-header">
					Initial state
					<button
						className="item-list-state-add"
						type="button"
						onClick={() => setInitialState([
							...initialState,
							{ type: "", stateKey: "", content: "{\n\n}" },
						])}
					><AddIcon /></button>
				</div>
				{initialState.map((state, index) => {
					const onChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => setInitialState([
						...initialState.slice(0, index),
						{
							...initialState[index],
							[e.target.dataset.field!]: e.target.value,
						},
						...initialState.slice(index + 1),
					])
					const onRemove = () => setInitialState([
						...initialState.slice(0, index),
						...initialState.slice(index + 1),
					])
					return <Fragment key={index}>
						<div className="item-list-item state-event-form">
							<input
								className="state-event-type"
								type="text"
								data-field="type"
								placeholder="Event type"
								value={state.type}
								onChange={onChange}
							/>
							<input
								className="state-event-key"
								type="text"
								data-field="stateKey"
								placeholder="State key"
								value={state.stateKey}
								onChange={onChange}
							/>
							<textarea
								className="state-event-content"
								data-field="content"
								placeholder="Event content"
								value={state.content}
								onChange={onChange}
								rows={3}
							/>
						</div>
						<button
							className="item-list-remove"
							type="button"
							onClick={onRemove}
						><CloseIcon /></button>
					</Fragment>
				})}
			</div>
		</details>

		<button type="submit" disabled={loading}>Create</button>
		{error && <div className="error">{error}</div>}
	</form>
}

export default CreateRoomView
