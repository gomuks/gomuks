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
import { JSX, RefObject, use, useEffect, useLayoutEffect, useRef } from "react"
import { getAvatarThumbnailURL, getMediaURL, getRoomAvatarThumbnailURL } from "@/api/media.ts"
import {
	AutocompleteMemberEntry,
	RoomStateStore,
	maybeRedactMemberEvent,
	useCustomEmojis,
	useRoomMember,
} from "@/api/statestore"
import {
	UserID,
	WrappedBotCommand,
	commandArgsToString,
	getDefaultArguments,
	stringToCommandArgs,
	unpackExtensibleText,
} from "@/api/types"
import { isFakeCommand } from "@/api/types/fakecommands.ts"
import { Emoji, emojiToMarkdown, useSortedAndFilteredEmojis } from "@/util/emoji"
import { makeMentionMarkdown, makeRoomMentionMarkdown } from "@/util/markdown.ts"
import useEvent from "@/util/useEvent.ts"
import ClientContext from "../ClientContext.ts"
import { RoomContext } from "../roomview/roomcontext.ts"
import type { ComposerState } from "./MessageComposer.tsx"
import { charToAutocompleteType } from "./getAutocompleter.ts"
import { useFilteredCommands, useFilteredMembers, useFilteredRooms } from "./userautocomplete.ts"
import "./Autocompleter.css"

export interface AutocompleteQuery {
	type: "user" | "room" | "emoji" | "command"
	query: string
	startPos: number
	endPos: number
	frozenQuery?: string
	selected?: number
	close?: boolean
}

export interface AutocompleterProps {
	setState: (state: Partial<ComposerState>) => void
	setAutocomplete: (params: AutocompleteQuery | null) => void
	textInput: RefObject<HTMLTextAreaElement | null>
	state: ComposerState
	params: AutocompleteQuery
	room: RoomStateStore
}

const positiveMod = (val: number, div: number) => (val % div + div) % div

interface InnerAutocompleterProps<T> extends AutocompleterProps {
	items: T[]
	getText: (item: T, state: ComposerState) => string
	getKey: (item: T) => string
	getNewState?: (item: T, params: AutocompleteQuery) => readonly [Partial<ComposerState>, number]
	render: (item: T) => JSX.Element
}

function useAutocompleter<T>({
	params, state, setState, setAutocomplete, textInput,
	items, getText, getKey, getNewState, render,
}: InnerAutocompleterProps<T>) {
	const prevItems = useRef<T[]>(null)
	const onSelect = useEvent((index: number, clearAutocomplete = false) => {
		if (items.length === 0) {
			return
		}
		index = positiveMod(index, items.length)
		const item = items[index]
		let newState: Partial<ComposerState>
		let endPos: number
		if (getNewState) {
			[newState, endPos] = getNewState(item, params)
		} else {
			const replacementText = getText(item, state)
			const newText = state.text.slice(0, params.startPos) + replacementText + state.text.slice(params.endPos)
			endPos = params.startPos + replacementText.length
			newState = {
				text: newText,
			}
		}
		if (textInput.current && newState.text) {
			// React messes up the selection when changing the value for some reason,
			// so bypass react here to avoid the caret jumping to the end and closing the autocompleter
			textInput.current.value = newState.text
			textInput.current.setSelectionRange(endPos, endPos)
		}
		setState(newState)
		setAutocomplete(clearAutocomplete ? null : {
			...params,
			endPos,
			frozenQuery: params.frozenQuery ?? params.query,
		})
		document.querySelector(`div.autocompletion-item[data-index='${index}']`)?.scrollIntoView({ block: "nearest" })
	})
	const onClick = (evt: React.MouseEvent<HTMLDivElement>) => {
		const idx = evt.currentTarget.getAttribute("data-index")
		if (idx) {
			onSelect(+idx, true)
		}
	}
	useEffect(() => {
		if (params.selected !== undefined) {
			onSelect(params.selected, params.close)
		}
	}, [onSelect, params.selected, params.close])
	useLayoutEffect(() => {
		if (params.type !== "command" || !state.text) {
			return
		}
		if (isFakeCommand(state.text)) {
			// Special case commands that don't use MSC4332
			setAutocomplete(null)
			return
		} else if (items.length === 0 && prevItems.current?.length) {
			for (const item of prevItems.current as WrappedBotCommand[]) {
				const argVals = stringToCommandArgs(item, state.text)
				if (argVals !== null) {
					setState({
						command: {
							spec: item,
							inputArgs: argVals,
						},
					})
					// This is an evil hack to make non-command autocompletion immediately start after
					// command autocompletion ends (if applicable) because onComposerCaretChange isn't fired.
					const acType = charToAutocompleteType(state.text.slice(-1))
					const secondToLastChar = state.text[state.text.length - 2]
					if (acType && (secondToLastChar === " " || secondToLastChar === "\n")) {
						setAutocomplete({
							type: acType,
							query: "",
							startPos: state.text.length - 1,
							endPos: state.text.length,
						})
					} else {
						setAutocomplete(null)
					}
					return
				}
			}
		}
		prevItems.current = items
	}, [params.type, items, state.text, setAutocomplete, setState])
	const selected = params.selected !== undefined ? positiveMod(params.selected, items.length) : -1
	return <div
		className={`autocompletions ac-${params.type} ${items.length === 0 ? "empty" : "has-items"}`}
		id="composer-autocompletions"
	>
		{items.map((item, i) => <div
			onClick={onClick}
			data-index={i}
			className={`autocompletion-item ac-${params.type} ${selected === i ? "selected" : ""}`}
			key={getKey(item)}
		>{render(item)}</div>)}
		{!items.length ? `No ${params.type}s matching ${params.query} found` : null}
	</div>
}

const emojiFuncs = {
	getText: (emoji: Emoji) => emojiToMarkdown(emoji),
	getKey: (emoji: Emoji) => `${emoji.c}-${emoji.u}`,
	render: (emoji: Emoji) => <>{emoji.u.startsWith("mxc://")
		? <img loading="lazy" src={getMediaURL(emoji.u)} alt={`:${emoji.n}:`}/>
		: emoji.u
	} :{emoji.n}:</>,
}

export const EmojiAutocompleter = ({ params, room, ...rest }: AutocompleterProps) => {
	const client = use(ClientContext)!
	const customEmojiPacks = useCustomEmojis(client.store, room)
	const items = useSortedAndFilteredEmojis((params.frozenQuery ?? params.query).slice(1), {
		frequentlyUsed: client.store.frequentlyUsedEmoji,
		customEmojiPacks,
	})
	return useAutocompleter({ params, room, ...rest, items, ...emojiFuncs })
}

const userFuncs = {
	getText: (user: AutocompleteMemberEntry, state: ComposerState) => state.command
		? user.userID : makeMentionMarkdown(user.displayName, user.userID),
	getKey: (user: AutocompleteMemberEntry) => user.userID,
	render: (user: AutocompleteMemberEntry) => <>
		<img
			className="small avatar"
			loading="lazy"
			src={getAvatarThumbnailURL(user.userID, { displayname: user.displayName, avatar_url: user.avatarURL })}
			alt=""
		/>
		{user.event.content.membership === "invite" ? <span className="invited-indicator">(invited) </span> : null}
		{user.displayName}
	</>,
}

export const UserAutocompleter = ({ params, room, ...rest }: AutocompleterProps) => {
	const items = useFilteredMembers(room, (params.frozenQuery ?? params.query).slice(1))
	return useAutocompleter({ params, room, ...rest, items, ...userFuncs })
}

const roomFuncs = {
	getText: (room: RoomStateStore) => makeRoomMentionMarkdown(
		room.meta.current.canonical_alias || room.meta.current.name || room.roomID,
		room.meta.current.canonical_alias || room.roomID,
		room.getViaServers(),
	),
	getKey: (room: RoomStateStore) => room.roomID,
	render: (room: RoomStateStore) => <>
		<img
			className={`small avatar ${room.meta.current.creation_content?.type === "m.space" ? "space" : ""}`}
			loading="lazy"
			src={getRoomAvatarThumbnailURL(room.meta.current)}
			alt=""
		/>
		{room.meta.current.name ?? <code>room.roomID</code>}
	</>,
}

export const RoomAutocompleter = ({ params, ...rest }: AutocompleterProps) => {
	const client = use(ClientContext)!
	const items = useFilteredRooms(client.store, (params.frozenQuery ?? params.query).slice(1))
	return useAutocompleter({ params, ...rest, items, ...roomFuncs })
}

const BotSourceIcon = ({ source }: { source: UserID }) => {
	const client = use(ClientContext)
	const roomCtx = use(RoomContext)
	const memberEvt = useRoomMember(client, roomCtx?.store, source)
	const memberEvtContent = maybeRedactMemberEvent(memberEvt)
	return <img
		className="avatar"
		loading="lazy"
		src={getAvatarThumbnailURL(source, memberEvtContent)}
		alt=""
	/>
}

const commandFuncs = {
	getText: () => "",
	getKey: (cmd: WrappedBotCommand) => cmd.source + cmd.command,
	getNewState: (cmd: WrappedBotCommand) => {
		if (cmd.fake) {
			return [{ command: null, text: "/" + cmd.command + " " }, cmd.command.length + 2] as const
		}
		const state = {
			command: {
				spec: cmd,
				inputArgs: getDefaultArguments(cmd),
			},
			text: "",
		}
		state.text = commandArgsToString(cmd, state.command.inputArgs)
		// TODO adding cmd.source might be disabled, make sure to sync with commandArgsToString
		let firstArgPos = cmd.command.length + cmd.source.length + 2
		if (state.text.charAt(firstArgPos) === `"` || state.text.charAt(firstArgPos) === `<`) {
			firstArgPos++
		}
		return [state, firstArgPos || state.text.length] as const
	},
	render: (cmd: WrappedBotCommand) => <>
		<BotSourceIcon source={cmd.source} />
		<code>/{cmd.command}{cmd.parameters.map(param =>
			` {${param.key}${param.schema.schema_type === "array" ? "..." : ""}}`)}</code>
		<span> - {unpackExtensibleText(cmd.description)}</span>
	</>,
}

export const CommandAutocompleter = ({ params, room, ...rest }: AutocompleterProps) => {
	const items = useFilteredCommands(room, (params.frozenQuery ?? params.query).slice(1))
	return useAutocompleter({ params, room, ...rest, items, ...commandFuncs })
}
