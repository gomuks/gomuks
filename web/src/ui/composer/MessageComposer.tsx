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
import React, {
	CSSProperties,
	JSX,
	use,
	useCallback,
	useEffect,
	useLayoutEffect,
	useReducer,
	useRef,
	useState,
} from "react"
import { useRoomEvent, useRoomState } from "@/api/statestore"
import {
	BotArgumentValue,
	EventID,
	MediaEncodingOptions,
	MediaMessageEventContent,
	MemDBEvent,
	Mentions,
	MessageEventContent,
	PowerLevelEventContent,
	RelatesTo,
	RoomID,
	URLPreview as URLPreviewType,
	WrappedBotCommand,
	stringToCommandArgs,
} from "@/api/types"
import { isFakeCommand } from "@/api/types/fakecommands.ts"
import { PartialEmoji, emojiToMarkdown } from "@/util/emoji"
import { useEventAsState } from "@/util/eventdispatcher.ts"
import { isMobileDevice } from "@/util/ismobile.ts"
import { escapeMarkdown } from "@/util/markdown.ts"
import { getEventLevel, getUserLevel } from "@/util/powerlevel.ts"
import { getRelatesTo, getServerName, isEventID } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import MainScreenContext from "../MainScreenContext.ts"
import EmojiPicker from "../emojipicker/EmojiPicker.tsx"
import GIFPicker from "../emojipicker/GIFPicker.tsx"
import StickerPicker from "../emojipicker/StickerPicker.tsx"
import { keyToString } from "../keybindings.ts"
import { ModalContext, modals } from "../modal"
import { useRoomContext } from "../roomview/roomcontext.ts"
import { ReplyBody } from "../timeline/ReplyBody.tsx"
import URLPreview from "../urlpreview/URLPreview.tsx"
import ErrorBoundary from "../util/ErrorBoundary.tsx"
import type { AutocompleteQuery } from "./Autocompleter.tsx"
import CommandInput from "./CommandInput.tsx"
import { ComposerLocation, ComposerLocationValue, ComposerMedia } from "./ComposerMedia.tsx"
import {
	canAutocompleteCommand,
	charToAutocompleteType,
	emojiQueryRegex,
	getAutocompleter,
	startsWithSingleSlash,
} from "./getAutocompleter.ts"
import { interceptCommand } from "./localcommands.ts"
import AttachIcon from "@/icons/attach.svg?react"
import CloseIcon from "@/icons/close.svg?react"
import EmojiIcon from "@/icons/emoji-categories/smileys-emotion.svg?react"
import GIFIcon from "@/icons/gif.svg?react"
import LocationIcon from "@/icons/location.svg?react"
import MicIcon from "@/icons/mic.svg?react"
import MoreIcon from "@/icons/more.svg?react"
import SendIcon from "@/icons/send.svg?react"
import StickerIcon from "@/icons/sticker.svg?react"
import "./MessageComposer.css"

export interface CommandState {
	spec: WrappedBotCommand
	inputArgs: Record<string, BotArgumentValue>
}

export interface ComposerState {
	text: string
	media: MediaMessageEventContent | null
	location: ComposerLocationValue | null
	command: CommandState | null
	previews: URLPreviewType[]
	loadingPreviews: string[]
	possiblePreviews: string[]
	replyTo: EventID | null
	mentionRoom: boolean
	silentReply: boolean
	explicitReplyInThread: boolean
	startNewThread: boolean
	uninited?: boolean
}

const MAX_TEXTAREA_ROWS = 10

const emptyComposer: ComposerState = {
	text: "",
	media: null,
	location: null,
	command: null,
	previews: [],
	loadingPreviews: [],
	possiblePreviews: [],
	replyTo: null,
	mentionRoom: true,
	silentReply: false,
	explicitReplyInThread: false,
	startNewThread: false,
}
const uninitedComposer: ComposerState = { ...emptyComposer, uninited: true }
const composerReducer = (
	state: ComposerState,
	action: Partial<ComposerState> | ((current: ComposerState) => Partial<ComposerState>),
) => ({
	...state,
	...(typeof action === "function" ? action(state) : action),
	uninited: undefined,
})

const draftStore = {
	makeDraftKey(roomID: RoomID, threadID?: EventID): string {
		if (threadID) {
			return `draft-${roomID}-${threadID}`
		}
		return `draft-${roomID}`
	},
	get: (roomID: RoomID, threadID?: EventID): ComposerState | null => {
		const data = localStorage.getItem(draftStore.makeDraftKey(roomID, threadID))
		if (!data) {
			return null
		}
		try {
			const parsed = JSON.parse(data)
			parsed.loadingPreviews = []
			return parsed
		} catch {
			return null
		}
	},
	set: (roomID: RoomID, data: ComposerState, threadID?: EventID) =>
		localStorage.setItem(draftStore.makeDraftKey(roomID, threadID), JSON.stringify(data)),
	clear: (roomID: RoomID, threadID?: EventID) =>
		localStorage.removeItem(draftStore.makeDraftKey(roomID, threadID)),
}

type CaretEvent<T> = React.MouseEvent<T> | React.KeyboardEvent<T> | React.ChangeEvent<T>

const MessageComposer = () => {
	const roomCtx = useRoomContext()
	const room = roomCtx.store
	const roomMeta = useEventAsState(room.meta)
	const isEncrypted = !!roomMeta.encryption_event
	const client = use(ClientContext)!
	const mainScreen = use(MainScreenContext)!
	const openModal = use(ModalContext)
	const [autocomplete, setAutocomplete] = useState<AutocompleteQuery | null>(null)
	const [state, setState] = useReducer(composerReducer, uninitedComposer)
	const [editing, rawSetEditing] = useState<MemDBEvent | null>(null)
	const [loadingMedia, setLoadingMedia] = useState<number | null>(null)
	const [ignorePermissions, setIgnorePermissions] = useState(false)
	const cancelMediaUpload = useRef(() => {})
	const fileInput = useRef<HTMLInputElement>(null)
	const textInput = useRef<HTMLTextAreaElement>(null)
	const composerRef = useRef<HTMLDivElement>(null)
	const textRows = useRef(1)
	const typingSentAt = useRef(0)
	const replyToEvt = useRoomEvent(room, state.replyTo)
	const tombstoneEvent = useRoomState(room, "m.room.tombstone", "")
	const createEvent = useRoomState(room, "m.room.create", "")
	const pls = useRoomState(room, "m.room.power_levels", "")?.content as PowerLevelEventContent | undefined
	roomCtx.insertText = useCallback((text: string) => {
		textInput.current?.focus()
		document.execCommand("insertText", false, text)
	}, [])
	roomCtx.setReplyTo = useCallback((evt: EventID | null) => {
		setState({ replyTo: evt, silentReply: false, explicitReplyInThread: false, startNewThread: false })
		textInput.current?.focus()
	}, [])
	const setSilentReply = useCallback((newVal: boolean | React.MouseEvent) => {
		if (typeof newVal === "boolean") {
			setState({ silentReply: newVal })
		} else {
			newVal.stopPropagation()
			setState(state => ({ silentReply: !state.silentReply }))
		}
	}, [])
	const setExplicitReplyInThread = useCallback((newVal: boolean | React.MouseEvent) => {
		if (typeof newVal === "boolean") {
			setState({ explicitReplyInThread: newVal })
		} else {
			newVal.stopPropagation()
			setState(state => ({ explicitReplyInThread: !state.explicitReplyInThread }))
		}
	}, [])
	const setStartNewThread = useCallback((newVal: boolean | React.MouseEvent) => {
		if (typeof newVal === "boolean") {
			setState({ startNewThread: newVal })
		} else {
			newVal.stopPropagation()
			setState(state => ({ startNewThread: !state.startNewThread }))
		}
	}, [])
	roomCtx.setEditing = useCallback((evt: MemDBEvent | null) => {
		if (evt === null) {
			rawSetEditing(null)
			setState(draftStore.get(room.roomID, roomCtx.threadRoot) ?? emptyComposer)
			return
		}
		const evtContent = evt.content as MessageEventContent
		const mediaMsgTypes = ["m.sticker", "m.image", "m.audio", "m.video", "m.file"]
		if (evt.type === "m.sticker") {
			evtContent.msgtype = "m.sticker"
		}
		const isMedia = mediaMsgTypes.includes(evtContent.msgtype)
			&& Boolean(evt.content?.url || evt.content?.file?.url)
		rawSetEditing(evt)
		const textIsEditable = (evt.content.filename && evt.content.filename !== evt.content.body)
			|| evt.type === "m.sticker"
			|| !isMedia
		setState({
			media: isMedia ? evtContent as MediaMessageEventContent : null,
			text: textIsEditable
				? (evt.local_content?.edit_source ?? evtContent.body ?? "")
				: "",
			replyTo: null,
			silentReply: false,
			explicitReplyInThread: false,
			startNewThread: false,
			command: null,
			previews:
				evt.content["m.url_previews"] ??
				evt.content["com.beeper.linkpreviews"] ??
				[],
		})
		textInput.current?.focus()
	}, [room.roomID, roomCtx.threadRoot])
	const canSend = Boolean(state.text || state.media || state.location)
	const onClickSend = (evt: React.FormEvent) => {
		evt.preventDefault()
		if (!canSend || loadingMedia !== null || state.loadingPreviews.length) {
			return
		}
		doSendMessage(state)
	}
	const doSendMessage = (state: ComposerState) => {
		if (editing) {
			setState(draftStore.get(room.roomID, roomCtx.threadRoot) ?? emptyComposer)
		} else {
			setState(emptyComposer)
		}
		rawSetEditing(null)
		setAutocomplete(null)
		const mentions: Mentions = {
			user_ids: [],
			room: state.text.includes("@room") && state.mentionRoom,
		}
		let relates_to: RelatesTo | undefined = undefined
		if (roomCtx.threadRoot) {
			relates_to = {
				rel_type: "m.thread",
				event_id: roomCtx.threadRoot,
				is_falling_back: true,
				"m.in_reply_to": {
					event_id: roomCtx.lastThreadEventID ?? roomCtx.threadRoot,
				},
			}
		}
		if (editing) {
			relates_to = {
				rel_type: "m.replace",
				event_id: editing.event_id,
			}
		} else if (replyToEvt) {
			const replyToEvtRelation = getRelatesTo(replyToEvt)
			const isThread = !roomCtx.threadRoot
				&& replyToEvtRelation?.rel_type === "m.thread"
				&& isEventID(replyToEvtRelation?.event_id)
			if (!state.silentReply && (!isThread || state.explicitReplyInThread)) {
				mentions.user_ids.push(replyToEvt.sender)
			}
			if (!relates_to) {
				relates_to = {}
			}
			relates_to["m.in_reply_to"] = {
				event_id: replyToEvt.event_id,
			}
			if (roomCtx.threadRoot) {
				relates_to.is_falling_back = false
			} else if (isThread) {
				relates_to.rel_type = "m.thread"
				relates_to.event_id = replyToEvtRelation.event_id
				relates_to.is_falling_back = !state.explicitReplyInThread
			} else if (state.startNewThread) {
				relates_to.rel_type = "m.thread"
				relates_to.event_id = replyToEvt.event_id
				relates_to.is_falling_back = true
			}
		}
		let base_content: MessageEventContent | undefined
		let extra: Record<string, unknown> | undefined
		let text = state.text
		if (state.media) {
			base_content = state.media
		} else if (state.location) {
			base_content = {
				body: "Location",
				msgtype: "m.location",
				geo_uri: `geo:${state.location.lat},${state.location.long}`,
			}
			extra = {
				"org.matrix.msc3488.asset": {
					type: "m.pin",
				},
				"org.matrix.msc3488.location": {
					uri: `geo:${state.location.lat},${state.location.long}`,
					description: state.text,
				},
			}
		}
		if (state.command) {
			base_content = {
				...(base_content ?? { msgtype: "m.text" }),
				body: text,
				"org.matrix.msc4391.command": {
					command: state.command.spec.command,
					arguments: state.command.inputArgs as Record<string, BotArgumentValue>,
				},
			}
			mentions.user_ids = [state.command.spec.source]
			mentions.room = false
			text = ""
			if (interceptCommand(client, mainScreen, roomCtx, state.command.spec, state.command.inputArgs)) {
				return
			}
		}
		client.sendMessage({
			room_id: room.roomID,
			base_content,
			extra,
			text,
			relates_to,
			mentions,
			url_previews: state.previews,
		}).catch(err => window.alert("Failed to send message: " + err))
	}
	const onComposerCaretChange = (evt: CaretEvent<HTMLTextAreaElement>, newText?: string) => {
		const area = evt.currentTarget
		if (area.selectionStart <= (autocomplete?.startPos ?? 0)) {
			if (
				autocomplete
				// Don't stop autocomplete on care move for commands, except if the leading / is removed
				&& (autocomplete.type !== "command"
					|| (newText !== undefined && !startsWithSingleSlash(newText)))
			) {
				setAutocomplete(null)
			}
			return
		}
		if (autocomplete?.frozenQuery) {
			if (area.selectionEnd !== autocomplete.endPos) {
				setAutocomplete(null)
			}
		} else if (autocomplete) {
			const newEndPos = autocomplete.type === "command" ? (newText ?? state.text).length : area.selectionEnd
			const newQuery = (newText ?? state.text).slice(autocomplete.startPos, newEndPos)
			if (
				(autocomplete.type !== "command" && newQuery.includes(" "))
				|| (autocomplete.type === "command" && !startsWithSingleSlash(newQuery))
				|| (autocomplete.type === "emoji" && !emojiQueryRegex.test(newQuery))
			) {
				setAutocomplete(null)
			} else if (newQuery !== autocomplete.query) {
				setAutocomplete({ ...autocomplete, query: newQuery, endPos: newEndPos })
			}
		} else if (area.selectionStart === area.selectionEnd) {
			if (newText && !state.command && canAutocompleteCommand(newText)) {
				setAutocomplete({
					type: "command",
					query: newText,
					startPos: 0,
					endPos: area.selectionEnd,
				})
				return
			}
			const acType = charToAutocompleteType(newText?.slice(area.selectionStart - 1, area.selectionStart))
			const prevChar = newText?.[area.selectionStart - 2]
			if (
				acType && (
					area.selectionStart === 1
					|| prevChar === " "
					|| prevChar === "\n"
					|| prevChar === `"`
				)
			) {
				setAutocomplete({
					type: acType,
					query: "",
					startPos: area.selectionStart - 1,
					endPos: area.selectionEnd,
				})
			}
		}
	}
	const onComposerKeyDown = (evt: React.KeyboardEvent<HTMLTextAreaElement>) => {
		const inp = evt.currentTarget
		const fullKey = keyToString(evt)
		const sendKey = fullKey === "Enter" || fullKey === "Ctrl+Enter"
			? (room.preferences.ctrl_enter_send ? "Ctrl+Enter" : "Enter")
			: null
		if (fullKey === sendKey && (
			// If the autocomplete already has a selected item or has no results, send message even if it's open.
			// Otherwise, don't send message on enter, select the first autocomplete entry instead.
			// Also don't send for command autocompletions, as we need to open the argument input.
			!autocomplete
			|| (autocomplete.selected !== undefined && autocomplete.type !== "command")
			|| !document.getElementById("composer-autocompletions")?.classList.contains("has-items")
		)) {
			onClickSend(evt)
		} else if (autocomplete) {
			let autocompleteUpdate: Partial<AutocompleteQuery> | null | undefined
			if (fullKey === "Tab" || fullKey === "ArrowDown") {
				autocompleteUpdate = { selected: (autocomplete.selected ?? -1) + 1 }
			} else if (fullKey === "Shift+Tab" || fullKey === "ArrowUp") {
				autocompleteUpdate = { selected: (autocomplete.selected ?? 0) - 1 }
			} else if (fullKey === "Enter") {
				autocompleteUpdate = { selected: autocomplete.selected ?? 0, close: true }
			} else if (fullKey === "Escape") {
				autocompleteUpdate = null
				if (autocomplete.frozenQuery) {
					setState({
						text: state.text.slice(0, autocomplete.startPos)
							+ autocomplete.frozenQuery
							+ state.text.slice(autocomplete.endPos),
						command: null,
					})
				}
			}
			if (autocompleteUpdate !== undefined) {
				setAutocomplete(autocompleteUpdate && { ...autocomplete, ...autocompleteUpdate })
				evt.preventDefault()
			}
		} else if (fullKey === "ArrowUp" && inp.selectionStart === 0 && inp.selectionEnd === 0) {
			const currentlyEditing = editing
				? room.editTargets.indexOf(editing.rowid)
				: room.editTargets.length
			const prevEventToEditID = room.editTargets[currentlyEditing - 1]
			const prevEventToEdit = prevEventToEditID ? room.eventsByRowID.get(prevEventToEditID) : undefined
			if (prevEventToEdit) {
				roomCtx.setEditing(prevEventToEdit)
				evt.preventDefault()
			}
		} else if (editing && fullKey === "ArrowDown" && inp.selectionStart === state.text.length) {
			const currentlyEditingIdx = room.editTargets.indexOf(editing.rowid)
			const nextEventToEdit = currentlyEditingIdx
				? room.eventsByRowID.get(room.editTargets[currentlyEditingIdx + 1]) : undefined
			roomCtx.setEditing(nextEventToEdit ?? null)
			// This timeout is very hacky and probably doesn't work in every case
			setTimeout(() => inp.setSelectionRange(0, 0), 0)
			evt.preventDefault()
		} else if (editing && fullKey === "Escape") {
			evt.stopPropagation()
			roomCtx.setEditing(null)
		} else if (!editing && fullKey === "Ctrl+ArrowUp" && room.preferences.ctrl_arrow_reply) {
			let replyToIdx = replyToEvt ? room.timeline.findIndex(item => item.event_rowid === replyToEvt.rowid) : -1
			if (replyToIdx === -1) {
				replyToIdx = room.timeline.length - 1
			} else if (replyToIdx > 0) {
				replyToIdx -= 1
			} else {
				return
			}
			const newReplyEvt = room.eventsByRowID.get(room.timeline[replyToIdx].event_rowid)
			if (newReplyEvt) {
				roomCtx.setReplyTo(newReplyEvt.event_id)
				evt.preventDefault()
			}
		} else if (
			!editing && fullKey === "Ctrl+ArrowDown" && replyToEvt !== null && room.preferences.ctrl_arrow_reply
		) {
			const replyToIdx = room.timeline.findIndex(item => item.event_rowid === replyToEvt.rowid)
			if (replyToIdx >= room.timeline.length - 1) {
				roomCtx.setReplyTo(null)
				evt.preventDefault()
			} else if (replyToIdx >= 0) {
				const newReplyEvt = room.eventsByRowID.get(room.timeline[replyToIdx + 1].event_rowid)
				if (newReplyEvt) {
					roomCtx.setReplyTo(newReplyEvt.event_id)
					evt.preventDefault()
				}
			}
		}
	}
	const onChange = (evt: React.ChangeEvent<HTMLTextAreaElement>) => {
		const newText = evt.target.value
		const newState: Partial<ComposerState> = { text: newText }
		if (state.command) {
			const inputArgs = stringToCommandArgs(state.command.spec, newText)
			if (inputArgs === null) {
				if (canAutocompleteCommand(newText)) {
					setAutocomplete({
						type: "command",
						query: newText,
						startPos: 0,
						endPos: evt.currentTarget.selectionEnd,
					})
				}
				newState.command = null
			} else {
				newState.command = { ...state.command, inputArgs }
			}
		}
		setState(newState)
		const now = Date.now()
		if (evt.target.value !== "" && typingSentAt.current + 5_000 < now) {
			typingSentAt.current = now
			if (!room.groupSessionAutoShared && isEncrypted) {
				client.rpc.ensureGroupSessionShared(room.roomID).then(
					() => room.groupSessionAutoShared = true,
					err => console.error("Failed to share group session:", err),
				)
			}
			if (room.preferences.send_typing_notifications) {
				client.rpc.setTyping(room.roomID, 10_000)
					.catch(err => console.error("Failed to send typing notification:", err))
			}
		} else if (evt.target.value === "" && typingSentAt.current > 0) {
			typingSentAt.current = 0
			if (room.preferences.send_typing_notifications) {
				client.rpc.setTyping(room.roomID, 0)
					.catch(err => console.error("Failed to send stop typing notification:", err))
			}
		}
		onComposerCaretChange(evt, evt.target.value)
	}
	const doUploadFile = useCallback((
		file: Blob,
		filename: string,
		encodingOpts?: MediaEncodingOptions,
	) => {
		const encryptUpload = Boolean(isEncrypted && !encodingOpts?._no_encrypt)
		if (client.rpc.rpcMediaUpload) {
			setLoadingMedia(0)
			client.rpc.uploadMedia(file, filename, encryptUpload).then(
				media => setState({ media, location: null }),
				err => window.alert(`Failed to upload file: ${err.message}`),
			).finally(() => setLoadingMedia(null))
			return
		}
		const params = new URLSearchParams([
			["encrypt", encryptUpload.toString()],
			["progress", "true"],
			["filename", filename],
			...Object.entries(encodingOpts ?? {})
				.filter(([key, value]) => !key.startsWith("_") && !!value)
				.map(([key, value]) => [key, value.toString()]),
		])
		const xhr = new XMLHttpRequest()
		xhr.upload.addEventListener("progress", evt => {
			setLoadingMedia(evt.lengthComputable ? evt.loaded / evt.total : 0)
		})
		let readUpTo = 0
		xhr.addEventListener("progress", () => {
			let newText = xhr.responseText.slice(readUpTo).trimEnd()
			readUpTo = xhr.responseText.length
			if (newText.includes("\n")) {
				newText = newText.slice(newText.lastIndexOf("\n")+1)
			}
			if (newText.startsWith("0.") || newText === "1") {
				setLoadingMedia(1+parseFloat(newText))
			}
		})
		xhr.addEventListener("load", () => {
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			let media: any = null
			try {
				media = JSON.parse(xhr.responseText.slice(xhr.responseText.indexOf("{")))
			} catch {}
			if (xhr.status >= 200 && xhr.status < 300 && !media?.error) {
				setState({ media, location: null })
			} else {
				window.alert(`Failed to upload file: ${media?.error || xhr.statusText}`)
			}
		})
		xhr.addEventListener("error", () => {
			window.alert(`Failed to upload file: request failed`)
		})
		xhr.addEventListener("abort", () => {
			window.alert(`Failed to upload file: request aborted`)
		})
		xhr.addEventListener("loadend", () => {
			cancelMediaUpload.current = () => {}
			setLoadingMedia(null)
		})

		cancelMediaUpload.current = () => xhr.abort()
		setLoadingMedia(0)
		xhr.open("POST", `_gomuks/upload?${params.toString()}`)
		xhr.setRequestHeader("Content-Type", file.type)
		xhr.send(file)
	}, [client.rpc, isEncrypted])
	const openFileUploadModal = (file: File | null | undefined, isVoice?: true) => {
		if (!file) {
			return
		}
		if (room.preferences.upload_dialog || (state.text.startsWith("/") && !isFakeCommand(state.text))) {
			openModal(modals.mediaUpload(file, doUploadFile, isEncrypted, isVoice))
		} else {
			window.closeModal()
			const encTo = isVoice && file.type !== "audio/ogg; codecs=opus" ? "audio/ogg; codecs=opus" : undefined
			doUploadFile(file, file.name, { voice_message: isVoice, encode_to: encTo })
		}
	}
	const onPaste = (evt: React.ClipboardEvent<HTMLTextAreaElement>) => {
		const file = evt.clipboardData?.files?.[0]
		const text = evt.clipboardData.getData("text/plain")
		const input = evt.currentTarget
		if (file) {
			openFileUploadModal(file)
		} else if (
			input.selectionStart !== input.selectionEnd
			&& (text.startsWith("http://") || text.startsWith("https://") || text.startsWith("matrix:"))
			&& state.text.slice(input.selectionStart, input.selectionStart + 8) !== text.slice(0, 8)
		) {
			document.execCommand("insertText", false, `[${
				state.text.slice(input.selectionStart, input.selectionEnd)
			}](${escapeMarkdown(text)})`)
		} else if (
			input.selectionStart === 0 && input.selectionEnd === state.text.length && canAutocompleteCommand(text)
		) {
			let matches: CommandState[] = []
			let longestMatch = 0
			for (const spec of room.getAllBotCommands()) {
				const inputArgs = stringToCommandArgs(spec, text)
				if (inputArgs !== null) {
					if (spec.command.length > longestMatch) {
						longestMatch = spec.command.length
						matches = [{ spec, inputArgs }]
					} else {
						matches.push({ spec, inputArgs })
					}
				}
			}
			if (matches.length) {
				setState({
					text,
					command: matches[0],
				})
				evt.preventDefault()
			}
			return
		} else {
			return
		}
		evt.preventDefault()
	}
	const resolvePreview = useCallback((url: string) => {
		setState(s => ({ loadingPreviews: [...s.loadingPreviews, url]}))
		fetch(`_gomuks/url_preview?encrypt=${isEncrypted}&url=${encodeURIComponent(url)}`, {
			method: "GET",
		})
			.then(async res => {
				const json = await res.json()
				if (!res.ok) {
					throw new Error(json.error)
				} else {
					setState(s => ({
						previews: [...s.previews, json],
						loadingPreviews: s.loadingPreviews.filter(u => u !== url),
					}))
				}
			})
			.catch(err => {
				console.error("Error fetching preview for URL", url, err)
				setState(s => ({
					loadingPreviews: s.loadingPreviews.filter(u => u !== url),
				}))
			})
	}, [isEncrypted])
	// To ensure the cursor jumps to the end, do this in an effect rather than as the initial value of useState
	// To try to avoid the input bar flashing, use useLayoutEffect instead of useEffect
	useLayoutEffect(() => {
		const draft = draftStore.get(room.roomID, roomCtx.threadRoot)
		setState(draft ?? emptyComposer)
		setAutocomplete(null)
		return () => {
			if (typingSentAt.current > 0) {
				typingSentAt.current = 0
				if (room.preferences.send_typing_notifications) {
					client.rpc.setTyping(room.roomID, 0)
						.catch(err => console.error("Failed to send stop typing notification due to room switch:", err))
				}
			}
		}
	}, [client, room, roomCtx])
	useEffect(() => {
		if (mainScreen.pendingShare) {
			console.info("Processing pending share")
			openModal(modals.mediaUpload(mainScreen.pendingShare, doUploadFile, isEncrypted))
			mainScreen.setPendingShare(null)
		}
	}, [mainScreen, roomCtx, doUploadFile, isEncrypted, openModal])
	useLayoutEffect(() => {
		if (!textInput.current) {
			return
		}
		// This is a hacky way to auto-resize the text area. Setting the rows to 1 and then
		// checking scrollHeight seems to be the only reliable way to get the size of the text.
		textInput.current.rows = 1
		const newTextRows = Math.min((textInput.current.scrollHeight - 16) / 20, MAX_TEXTAREA_ROWS)
		if (newTextRows === MAX_TEXTAREA_ROWS) {
			textInput.current.style.overflowY = "auto"
		} else {
			// There's a weird 1px scroll when using line-height, so set overflow to hidden when it's not needed
			textInput.current.style.overflowY = "hidden"
		}
		textInput.current.rows = newTextRows
		textRows.current = newTextRows
		// This has to be called unconditionally, because setting rows = 1 messes up the scroll state otherwise
		roomCtx.scrollToBottom()
		// scrollToBottom needs to be called when replies/attachments/etc change,
		// so listen to state instead of only state.text
	}, [state, roomCtx])
	// Saving to localStorage could be done in the reducer, but that's not very proper, so do it in an effect.
	useEffect(() => {
		roomCtx.isEditing.emit(editing !== null)
		if (state.uninited || editing) {
			return
		}
		if (!state.text && !state.media && !state.replyTo && !state.location) {
			draftStore.clear(room.roomID, roomCtx.threadRoot)
		} else {
			draftStore.set(room.roomID, state, roomCtx.threadRoot)
		}
	}, [roomCtx, room, state, editing])
	useEffect(() => {
		if (state.uninited) {
			return
		}
		if (!room.preferences.send_bundled_url_previews) {
			setState({ previews: [], loadingPreviews: [], possiblePreviews: []})
			return
		}
		const urls = state.text.matchAll(/\bhttps?:\/\/[^\s/_*]+(?:\/\S*)?\b/gi)
			.map(m => m[0])
			.filter(u => !u.startsWith("https://matrix.to"))
			.toArray()
		setState(s => ({
			previews: s.previews.filter(p => urls.includes(p.matched_url)),
			loadingPreviews: s.loadingPreviews.filter(u => urls.includes(u)),
			possiblePreviews: urls,
		}))
	}, [room.preferences, state.uninited, state.text])
	const clearMedia = useCallback(() => setState({ media: null, location: null }), [])
	const onChangeLocation = useCallback((location: ComposerLocationValue) => setState({ location }), [])
	const closeReply = useCallback((evt: React.MouseEvent) => {
		evt.stopPropagation()
		setState({ replyTo: null })
	}, [])
	const stopEditing = useCallback((evt: React.MouseEvent) => {
		evt.stopPropagation()
		roomCtx.setEditing(null)
	}, [roomCtx])
	const Autocompleter = getAutocompleter(autocomplete, client, room)
	let mediaDisabledTitle: string | undefined
	let stickerDisabledTitle: string | undefined
	let locationDisabledTitle: string | undefined
	if (state.media) {
		mediaDisabledTitle = "You can only attach one file at a time"
		locationDisabledTitle = "You can't attach a location to a message with a file"
	} else if (state.location) {
		mediaDisabledTitle = "You can't attach a file to a message with a location"
		locationDisabledTitle = "You can only attach one location at a time"
	} else if (loadingMedia !== null) {
		mediaDisabledTitle = "Uploading file..."
		locationDisabledTitle = "You can't attach a location to a message with a file"
	}
	if (state.media?.msgtype !== "m.sticker") {
		stickerDisabledTitle = mediaDisabledTitle
		if (!stickerDisabledTitle && editing) {
			stickerDisabledTitle = "You can't edit a message into a sticker"
		}
	} else if (state.text && !editing) {
		stickerDisabledTitle = "You can't attach a sticker to a message with text"
	}
	const getEmojiPickerStyle = () => ({
		bottom: (composerRef.current?.clientHeight ?? 32) + 4 + 24,
		right: "var(--timeline-horizontal-padding)",
	})
	const makeAttachmentButtons = (includeText = false) => {
		const openEmojiPicker = () => {
			openModal({
				content: <EmojiPicker
					style={getEmojiPickerStyle()}
					room={roomCtx.store}
					onSelect={(emoji: PartialEmoji) => {
						const mdEmoji = emojiToMarkdown(emoji)
						setState({
							text: state.text.slice(0, textInput.current?.selectionStart ?? 0)
								+ mdEmoji
								+ state.text.slice(textInput.current?.selectionEnd ?? 0),
						})
						if (textInput.current) {
							textInput.current.setSelectionRange(textInput.current.selectionStart + mdEmoji.length, 0)
						}
					}}
					// TODO allow keeping open on select on non-mobile devices
					//      (requires onSelect to be able to keep track of the state after updating it)
					closeOnSelect={true}
				/>,
				onClose: () => !isMobileDevice && textInput.current?.focus(),
			})
		}
		const openGIFPicker = () => {
			openModal({
				content: <GIFPicker
					style={getEmojiPickerStyle()}
					room={roomCtx.store}
					onSelect={media => setState({ media })}
				/>,
				onClose: () => !isMobileDevice && textInput.current?.focus(),
			})
		}
		const openStickerPicker = () => {
			openModal({
				content: <StickerPicker
					style={getEmojiPickerStyle()}
					room={roomCtx.store}
					onSelect={media => doSendMessage({ ...state, media, text: "" })}
				/>,
				onClose: () => !isMobileDevice && textInput.current?.focus(),
			})
		}
		const openVoiceRecorder = () => {
			openModal(modals.voiceRecorder(openFileUploadModal, textInput))
		}
		const openLocationPicker = () => {
			setState({ location: { lat: 0, long: 0, prec: 1 }, media: null })
		}
		return <>
			<button onClick={openEmojiPicker} title="Add emoji"><EmojiIcon/>{includeText && "Emoji"}</button>
			<button
				onClick={openStickerPicker}
				disabled={!!stickerDisabledTitle}
				title={stickerDisabledTitle ?? "Add sticker attachment"}
			>
				<StickerIcon/>{includeText && "Sticker"}
			</button>
			<button
				onClick={openGIFPicker}
				disabled={!!mediaDisabledTitle}
				title={mediaDisabledTitle ?? "Add gif attachment"}
			>
				<GIFIcon/>{includeText && "GIF"}
			</button>
			<button
				onClick={openLocationPicker}
				disabled={!!locationDisabledTitle}
				title={locationDisabledTitle ?? "Add location"}
			><LocationIcon/>{includeText && "Location"}</button>
			<button
				onClick={openVoiceRecorder}
				disabled={!!mediaDisabledTitle}
				title={mediaDisabledTitle ?? "Record voice message"}
			><MicIcon/>{includeText && "Voice"}</button>
			<button
				onClick={() => fileInput.current!.click()}
				disabled={!!mediaDisabledTitle}
				title={mediaDisabledTitle ?? "Add file attachment"}
			><AttachIcon/>{includeText && "File"}</button>
		</>
	}
	const openButtonsModal = (evt: React.MouseEvent<HTMLButtonElement>) => {
		const style: CSSProperties = {
			bottom: (composerRef.current?.clientHeight ?? 32) + 4 + 24,
			left: evt.currentTarget.getBoundingClientRect().left - 1,
		}
		openModal({
			content: <div className="context-menu event-context-menu" style={style}>
				{makeAttachmentButtons(true)}
			</div>,
		})
	}
	const collapseButtons = (composerRef.current ? composerRef.current.clientWidth : window.innerWidth - 16) < 600
	const inlineButtons = state.text === "" || !collapseButtons
	const showSendButton = canSend || !collapseButtons
	const disableClearMedia = editing && state.media?.msgtype === "m.sticker"
	if (tombstoneEvent !== null) {
		const content = tombstoneEvent.content
		const hasReplacement = content.replacement_room?.startsWith("!")
		let link: JSX.Element | null = null
		if (hasReplacement) {
			const via = getServerName(tombstoneEvent.sender)
			const handleNavigate = (e: React.MouseEvent<HTMLAnchorElement, MouseEvent>) => {
				e.preventDefault()
				mainScreen.setActiveRoom(content.replacement_room, {
					previewMeta: {
						via: [via],
					},
				})
			}
			const url = `matrix:roomid/${content.replacement_room.slice(1)}?via=${via}`
			link = <a href={url} onClick={handleNavigate}>
				Join the new one here
			</a>
		}
		let body = content.body
		if (!body) {
			body = hasReplacement ? "This room has been replaced." : "This room has been shut down."
		}
		if (!body.endsWith(".")) {
			body += "."
		}
		return <div className="message-composer tombstoned" ref={composerRef}>
			{body} {link}
		</div>
	} else if (
		!ignorePermissions
		&& getUserLevel(pls, createEvent, client.userID)
			< getEventLevel(pls, isEncrypted ? "m.room.encrypted" : "m.room.message")
	) {
		return <div className="message-composer no-permission" ref={composerRef}>
			You don't have permission to send messages in this room. <a
				href="javascript:"
				onClick={() => setIgnorePermissions(true)}
			>Click to show composer anyway</a>
		</div>
	}
	const possiblePreviewsNotLoadingOrPreviewed = state.possiblePreviews.filter(
		url => !state.loadingPreviews.includes(url) && !state.previews.some(p => p.matched_url === url))
	return <>
		{Autocompleter && autocomplete ? <div className="autocompletions-wrapper">
			<ErrorBoundary thing="autocompleter" wrapperClassName="autocompletions">
				<Autocompleter
					params={autocomplete}
					room={room}
					state={state}
					setState={setState}
					setAutocomplete={setAutocomplete}
					textInput={textInput}
				/>
			</ErrorBoundary>
		</div> : state.command ? <div className="command-argument-wrapper">
			<ErrorBoundary thing="command input" wrapperClassName="command-arguments">
				<CommandInput
					room={room}
					state={state}
					setState={setState}
				/>
			</ErrorBoundary>
		</div> : null}
		<div className="message-composer" ref={composerRef}>
			{replyToEvt && <ReplyBody
				roomCtx={roomCtx}
				event={replyToEvt}
				onClose={closeReply}
				isThread={!roomCtx.threadRoot && getRelatesTo(replyToEvt)?.rel_type === "m.thread"}
				isSilent={state.silentReply}
				onSetSilent={setSilentReply}
				isExplicitInThread={state.explicitReplyInThread}
				onSetExplicitInThread={!roomCtx.threadRoot ? setExplicitReplyInThread : undefined}
				startNewThread={state.startNewThread}
				onSetStartNewThread={!roomCtx.threadRoot ? setStartNewThread : undefined}
			/>}
			{editing && <ReplyBody
				roomCtx={roomCtx}
				event={editing}
				isEditing={true}
				isThread={false}
				onClose={stopEditing}
			/>}
			{loadingMedia !== null && <div className="composer-media">
				<label>
					<div>Uploading media...</div>
					<progress max={2} value={loadingMedia === 0 ? undefined : loadingMedia} />
				</label>
				{<button onClick={cancelMediaUpload.current}><CloseIcon/></button>}
			</div>}
			{state.media && <ComposerMedia content={state.media} clearMedia={!disableClearMedia && clearMedia}/>}
			{state.location && <ComposerLocation
				room={room} client={client}
				location={state.location} onChange={onChangeLocation} clearLocation={clearMedia}
			/>}
			{state.text.includes("@room") && <label className="mention-confirmations">
				<input
					type="checkbox"
					checked={state.mentionRoom}
					onChange={evt => setState({ mentionRoom: evt.currentTarget.checked })}
				/>
				Mention @room
			</label>}
			{state.previews.length || state.loadingPreviews.length || possiblePreviewsNotLoadingOrPreviewed
				? <div className="url-previews">
					{state.previews.map((preview, i) => <URLPreview
						key={i}
						url={preview.matched_url}
						preview={preview}
						clearPreview={() => setState(s => ({ previews: s.previews.filter((_, j) => j !== i) }))}
					/>)}
					{state.loadingPreviews.map((previewURL, i) =>
						<URLPreview	key={i} url={previewURL} preview="loading"/>)}
					{possiblePreviewsNotLoadingOrPreviewed.map((url, i) =>
						<URLPreview
							key={i}
							url={url}
							preview="awaiting_user"
							startLoadingPreview={() => resolvePreview(url)}
						/>)}
				</div>
				: null}
			<div className="input-area">
				{!inlineButtons && <button className="show-more" onClick={openButtonsModal}><MoreIcon/></button>}
				<textarea
					autoFocus={!isMobileDevice}
					ref={textInput}
					rows={textRows.current}
					value={state.text}
					onKeyDown={onComposerKeyDown}
					onKeyUp={onComposerCaretChange}
					onClick={onComposerCaretChange}
					onPaste={onPaste}
					onChange={onChange}
					placeholder={isEncrypted ? "Send a message (encrypted)" : "Send a message"}
					id="message-composer"
				/>
				{inlineButtons && makeAttachmentButtons()}
				{showSendButton && <button
					onClick={onClickSend}
					disabled={!canSend || loadingMedia !== null || !!state.loadingPreviews.length}
					title="Send message"
				><SendIcon/></button>}
				<input
					ref={fileInput}
					onChange={evt => openFileUploadModal(evt.target.files?.[0])}
					type="file"
					value=""
				/>
			</div>
		</div>
	</>
}

export default MessageComposer
