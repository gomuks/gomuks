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
import { JSX, use, useCallback, useEffect, useState } from "react"
import { RoomStateStore, StateStore, useAccountData, useRoomAccountData, useRoomState } from "@/api/statestore"
import { MemDBEvent, UnknownEventContent } from "@/api/types"
import ClientContext from "../ClientContext.ts"
import JSONView from "../util/JSONView"
import "./RoomStateExplorer.css"

interface StateExplorerProps {
	room: RoomStateStore
}

enum EventKind {
	None,
	Message,
	State,
	AccountData,
	RoomAccountData,
	Profile,
}

function kindName(kind: EventKind): string {
	switch (kind) {
	case EventKind.None:
		return ""
	case EventKind.Message:
		return "Message"
	case EventKind.State:
		return "Room State"
	case EventKind.AccountData:
		return "Account Data"
	case EventKind.RoomAccountData:
		return "Room Account Data"
	case EventKind.Profile:
		return "Profile"
	}
}

type DoneCallback = (kind: EventKind, type: string, stateKey: string | undefined, content: unknown | undefined) => void

interface BaseEventViewProps {
	type?: string
	stateKey?: string
	onBack: () => void
	onDone?: DoneCallback
}

interface EventViewProps extends BaseEventViewProps {
	kind: EventKind
	event: MemDBEvent | UnknownEventContent | null
	room?: RoomStateStore
	nestedContent?: boolean
}

interface RoomEventViewProps extends BaseEventViewProps {
	room: RoomStateStore
}

interface GlobalEventViewProps extends BaseEventViewProps {
	store: StateStore
}

interface StateKeyListProps {
	room: RoomStateStore
	type: string
	onSelectStateKey: (stateKey: string) => void
	onBack: () => void
}

const StateEventView = ({
	room, type, stateKey, onBack, onDone,
}: RoomEventViewProps) => {
	const event = useRoomState(room, type, stateKey)
	return <EventView
		room={room}
		kind={EventKind.State}
		type={type}
		stateKey={stateKey}
		event={event}
		nestedContent={true}
		onBack={onBack}
		onDone={onDone}
	/>
}

const RoomAccountDataEventView = ({
	room, type, onBack, onDone,
}: RoomEventViewProps) => {
	const content = useRoomAccountData(room, type)
	return <EventView
		room={room} kind={EventKind.RoomAccountData} type={type} event={content} onBack={onBack} onDone={onDone}
	/>
}

const GlobalAccountDataEventView = ({
	store, type, onBack, onDone,
}: GlobalEventViewProps) => {
	const content = useAccountData(store, type)
	return <EventView
		kind={EventKind.AccountData} type={type} event={content} onBack={onBack} onDone={onDone}
	/>
}

const NewMessageEventView = ({ room, onBack, onDone }: RoomEventViewProps) => {
	return <EventView
		kind={EventKind.Message} event={null} room={room} onBack={onBack} onDone={onDone}
	/>
}

const EventView = ({
	room, kind, event, nestedContent, type, stateKey, onBack, onDone,
}: EventViewProps) => {
	const isNewEvent = type === undefined
	const [editingContent, setEditingContent] = useState<string | null>(
		isNewEvent
			? kind === EventKind.Profile ? "" : "{\n\n}"
			: null,
	)
	const [newType, setNewType] = useState<string>(type || "")
	const [newStateKey, setNewStateKey] = useState<string>(stateKey || "")
	const [disableEncryption, setDisableEncryption] = useState<boolean>(false)
	const client = use(ClientContext)!

	const sendEdit = () => {
		let parsedContent
		try {
			parsedContent = JSON.parse(editingContent!)
		} catch (err) {
			window.alert(`Failed to parse JSON: ${err}`)
			return
		}
		let resp: Promise<unknown>
		if (kind === EventKind.State) {
			resp = client.rpc.setState(
				room!.roomID,
				newType,
				newStateKey,
				parsedContent,
			)
		} else if (kind === EventKind.Message) {
			resp = client.sendEvent(room!.roomID, newType, parsedContent, disableEncryption)
		} else if (kind === EventKind.AccountData || kind === EventKind.RoomAccountData) {
			resp = client.rpc.setAccountData(
				newType,
				parsedContent,
				kind === EventKind.RoomAccountData ? room!.roomID : undefined,
			)
		} else if (kind === EventKind.Profile) {
			resp = client.rpc.setProfileField(newType, parsedContent)
		} else {
			throw new Error("Invalid event kind for editing")
		}
		resp.then(() => {
			console.log("Sent updated event", kind, room?.roomID, type, stateKey)
			setEditingContent(null)
			if (isNewEvent || kind === EventKind.Profile) {
				onDone?.(kind, newType, newStateKey, parsedContent)
			}
		}, err => {
			console.error("Failed to send updated event", err)
			window.alert(`Failed to send updated event: ${err}`)
		})
	}
	const doDelete = () => {
		client.rpc.setProfileField(newType, undefined).then(
			() => {
				console.log("Deleted profile field", newType)
				onDone?.(kind, newType, newStateKey, undefined)
				onBack()
			},
			err => {
				console.error("Failed to delete profile field", err)
				window.alert(`Failed to delete profile field: ${err}`)
			},
		)
	}
	const stopEdit = () => setEditingContent(null)
	const startEdit = () => setEditingContent(
		kind === EventKind.Profile
			? JSON.stringify(event, null, 4)
			: JSON.stringify((nestedContent ? event?.content : event) || {}, null, 4),
	)

	return (
		<div className="state-explorer state-event-view">
			<div className="state-header">
				{isNewEvent
					? <h3>New {kindName(kind).toLowerCase()} event</h3>
					: <h3>
						{kindName(kind)}: <code>{type}</code> {kind === EventKind.State ? <>
							({stateKey ? <code>{stateKey}</code> : "no state key"})
						</> : null}
					</h3>}
				{editingContent !== null ? <div className="new-event-type">
					<input
						autoFocus
						type="text"
						value={newType}
						onChange={evt => setNewType(evt.target.value)}
						placeholder="Event type"
					/>
					{kind === EventKind.State ? <input
						type="text"
						value={newStateKey}
						onChange={evt => setNewStateKey(evt.target.value)}
						placeholder="State key"
					/> : null}
				</div> : null}
			</div>
			<div className="state-event-content">
				{editingContent !== null
					? <textarea rows={10} value={editingContent} onChange={evt => setEditingContent(evt.target.value)}/>
					: <JSONView data={event}/>
				}
			</div>
			<div className="nav-buttons">
				{editingContent !== null ? <>
					<button onClick={isNewEvent ? onBack : stopEdit}>Back</button>
					<div className="spacer"/>
					{kind === EventKind.Message && room!.meta.current.encryption_event ? <label>
						<input
							type="checkbox"
							checked={disableEncryption}
							onChange={evt => setDisableEncryption(evt.target.checked)}
						/>
						Disable encryption
					</label> : null}
					<button onClick={sendEdit}>Send</button>
				</> : <>
					<button onClick={onBack}>Back</button>
					<div className="spacer"/>
					{kind === EventKind.Profile ? <button onClick={doDelete}>Delete</button> : null}
					<button onClick={startEdit}>Edit</button>
				</>}
			</div>
		</div>
	)
}

const StateKeyList = ({ room, type, onSelectStateKey, onBack }: StateKeyListProps) => {
	const [filter, setFilter] = useState("")
	const stateMap = room.state.get(type)
	return (
		<div className="state-explorer state-key-list">
			<div className="state-header">
				<h3>State keys under <code>{type}</code></h3>
				<input
					type="search"
					className="search-field"
					placeholder="Filter state keys"
					value={filter}
					onChange={evt => setFilter(evt.target.value)}
				/>
				{type === "m.room.member" && !room.fullMembersLoaded ? "Warning: member list hasn't been loaded" : null}
			</div>
			<div className="state-button-list">
				{Array.from(stateMap?.keys().map(stateKey => stateKey.includes(filter) && (
					<button key={stateKey} onClick={() => onSelectStateKey(stateKey)}>
						{stateKey ? <code>{stateKey}</code> : "<empty>"}
					</button>
				)).filter(x => !!x) ?? [])}
			</div>
			<div className="nav-buttons">
				<button onClick={onBack}>Back</button>
			</div>
		</div>
	)
}

export const StateExplorer = ({ room }: StateExplorerProps) => {
	const [viewKind, setViewKind] = useState<EventKind>(EventKind.State)
	const [creatingNew, setCreatingNew] = useState<EventKind | null>(null)
	const [selectedType, setSelectedType] = useState<string | null>(null)
	const [selectedStateKey, setSelectedStateKey] = useState<string | null>(null)
	const [loadingState, setLoadingState] = useState(false)
	const [resettingTimeline, setResettingTimeline] = useState(false)
	const [profile, setProfile] = useState<UnknownEventContent | null>(null)
	const client = use(ClientContext)!

	useEffect(() => {
		if (!profile && viewKind == EventKind.Profile) {
			client.rpc.getProfile(client.userID).then(
				setProfile,
				err => {
					console.error("Failed to load profile", err)
					window.alert(`Failed to load profile: ${err}`)
				},
			)
		}
	}, [viewKind, client, profile])

	const handleTypeSelect = (type: string) => {
		if (viewKind !== EventKind.State) {
			setSelectedType(type)
			setSelectedStateKey(null)
			return
		}
		const stateKeysMap = room.state.get(type)
		if (!stateKeysMap) {
			return
		}

		const stateKeys = Array.from(stateKeysMap.keys())
		if (stateKeys.length === 1 && stateKeys[0] === "") {
			// If there's only one state event with an empty key, view it directly
			setSelectedType(type)
			setSelectedStateKey("")
		} else {
			// Otherwise show the list of state keys
			setSelectedType(type)
			setSelectedStateKey(null)
		}
	}

	const handleBack = useCallback(() => {
		if (creatingNew) {
			setCreatingNew(null)
		} else if (selectedStateKey !== null && selectedType !== null) {
			setSelectedStateKey(null)
			if (viewKind !== EventKind.State) {
				setSelectedType(null)
			} else {
				const stateKeysMap = room.state.get(selectedType)
				if (stateKeysMap?.size === 1 && stateKeysMap.has("")) {
					setSelectedType(null)
				}
			}
		} else if (selectedType !== null) {
			setSelectedType(null)
		}
	}, [viewKind, selectedType, selectedStateKey, creatingNew, room])
	const handleNewEventDone: DoneCallback = useCallback((kind, type, stateKey, content) => {
		setCreatingNew(null)
		setSelectedType(type)
		setSelectedStateKey(stateKey ?? null)
		if (kind === EventKind.Profile) {
			setProfile(prev => {
				const upd = {
					...prev,
					[type]: content,
				}
				if (content === undefined) {
					delete upd[type]
				}
				return upd
			})
		}
	}, [])

	switch (creatingNew) {
	case EventKind.State:
		return <StateEventView room={room} onBack={handleBack} onDone={handleNewEventDone} />
	case EventKind.Message:
		return <NewMessageEventView room={room} onBack={handleBack} onDone={handleBack} />
	case EventKind.AccountData:
		return <GlobalAccountDataEventView store={client.store} onBack={handleBack} onDone={handleNewEventDone} />
	case EventKind.RoomAccountData:
		return <RoomAccountDataEventView room={room} onBack={handleBack} onDone={handleNewEventDone} />
	case EventKind.Profile:
		return <EventView kind={EventKind.Profile} event={null} onBack={handleBack} onDone={handleNewEventDone} />
	}
	if (selectedType !== null) {
		switch (viewKind) {
		case EventKind.State:
			if (selectedStateKey === null) {
				return <StateKeyList
					room={room} type={selectedType} onSelectStateKey={setSelectedStateKey} onBack={handleBack}
				/>
			}
			return <StateEventView room={room} type={selectedType} stateKey={selectedStateKey!} onBack={handleBack} />
		case EventKind.AccountData:
			return <GlobalAccountDataEventView store={client.store} type={selectedType} onBack={handleBack} />
		case EventKind.RoomAccountData:
			return <RoomAccountDataEventView room={room} type={selectedType} onBack={handleBack} />
		case EventKind.Profile:
			return <EventView
				kind={EventKind.Profile} type={selectedType} event={profile![selectedType]} onBack={handleBack}
				onDone={handleNewEventDone}
			/>
		default:
			return <div>Invalid view kind</div>
		}
	}
	const loadRoomState = () => {
		setLoadingState(true)
		client.loadRoomState(room.roomID, {
			omitMembers: false,
			refetch: room.stateLoaded && room.fullMembersLoaded,
		}).then(
			() => {
				console.log("Room state loaded from devtools", room.roomID)
			},
			err => {
				console.error("Failed to fetch room state", err)
				window.alert(`Failed to fetch room state: ${err}`)
			},
		).finally(() => setLoadingState(false))
	}
	const resetTimeline = () => {
		setResettingTimeline(true)
		client.resetTimeline(room.roomID)
			.finally(() => setResettingTimeline(false))
	}
	let stateKeys: MapIterator<string> | string[]
	let navButtons: JSX.Element
	switch (viewKind) {
	case EventKind.State:
		stateKeys = room.state.keys()
		navButtons = <>
			<button onClick={loadRoomState} disabled={loadingState}>
				{room.stateLoaded
					? room.fullMembersLoaded
						? "Resync full room state"
						: "Load room members"
					: "Load room state and members"}
			</button>
			<button onClick={resetTimeline} disabled={resettingTimeline}>
				Reset timeline
			</button>
			<div className="spacer"/>
			<button onClick={() => setCreatingNew(EventKind.Message)}>Send new message event</button>
			<button onClick={() => setCreatingNew(EventKind.State)}>Send new state event</button>
		</>
		break
	case EventKind.AccountData:
		stateKeys = client.store.accountData.keys()
		navButtons = <>
			<div className="spacer"/>
			<button onClick={() => setCreatingNew(EventKind.AccountData)}>Send new account data event</button>
		</>
		break
	case EventKind.RoomAccountData:
		stateKeys = room.accountData.keys()
		navButtons = <>
			<div className="spacer"/>
			<button onClick={() => setCreatingNew(EventKind.RoomAccountData)}>Send new room account data event</button>
		</>
		break
	case EventKind.Profile:
		// TODO loading indicator when null
		stateKeys = profile ? Object.keys(profile) : []
		navButtons = <>
			<div className="spacer"/>
			<button onClick={() => setCreatingNew(EventKind.Profile)}>Add new profile field</button>
		</>
		break
	default:
		return <div className="state-explorer">Invalid view kind</div>
	}
	const kinds = [EventKind.State, EventKind.AccountData, EventKind.RoomAccountData, EventKind.Profile]
	return <div className="state-explorer">
		<div className="title-bar">
			{kinds.map(kind =>
				<button key={kind} disabled={viewKind === kind} onClick={() => setViewKind(kind)}>
					{kindName(kind)}
				</button>,
			)}
		</div>
		<div className="state-button-list">
			{Array.from(stateKeys.map(type => (
				<button key={type} onClick={() => handleTypeSelect(type)}>
					<code>{type}</code>
				</button>
			)) ?? [])}
		</div>
		<div className="nav-buttons">
			{navButtons}
		</div>
	</div>
}

export default StateExplorer
