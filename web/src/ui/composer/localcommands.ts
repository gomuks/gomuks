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
import Client from "@/api/client.ts"
import { fakeGomuksSender } from "@/api/statestore"
import { BotArgumentValue, EventID, RawDBEvent, RoomID, WrappedBotCommand } from "@/api/types"
import type { CommandName } from "@/api/types/stdcommands.d.ts"
import { escapeHTML } from "@/util/markdown.ts"
import { ensureString, ensureStringArray, matrixToToMatrixURI, parseMatrixURI } from "@/util/validation.ts"
import { MainScreenContextFields } from "../MainScreenContext.ts"
import { modals } from "../modal"
import { RoomContextData } from "../roomview/roomcontext.ts"
import { jumpToEvent } from "../util/jumpToEvent.tsx"

const commandHandlers: { [K in CommandName]?: CommandCallback } = {
	join: ({ client, mainScreen, reply }, { room_reference, reason, via }) => {
		if (typeof room_reference !== "string") {
			return
		}
		room_reference = matrixToToMatrixURI(room_reference) ?? room_reference
		let parsedVia = ensureStringArray(via)
		let openEventID: EventID | undefined
		if (room_reference.startsWith("matrix:")) {
			const parsed = parseMatrixURI(room_reference)
			if (parsed) {
				room_reference = parsed.identifier
				parsedVia = parsed.params.getAll("via")
				openEventID = parsed.eventID
			}
		}
		if (room_reference.startsWith("#")) {
			client.rpc.resolveAlias(room_reference).then(
				res => {
					mainScreen.setActiveRoom(res.room_id, {
						previewMeta: {
							alias: room_reference,
							via: res.servers.slice(0, 3),
							joinReason: ensureString(reason) || undefined,
						},
					})
				},
				err => reply(escapedHTML`Failed to resolve alias <code>${room_reference}</code>: ${err.message}`),
			)
		} else if (room_reference.startsWith("!")) {
			mainScreen.setActiveRoom(room_reference, {
				previewMeta: {
					via: parsedVia,
					joinReason: ensureString(reason) || undefined,
				},
				openEventID,
			})
		} else if (room_reference.startsWith("@")) {
			mainScreen.setRightPanel({
				type: "user",
				userID: room_reference,
			})
		} else if (room_reference.startsWith("$") && window.activeRoomContext) {
			jumpToEvent(window.activeRoomContext, room_reference)
		} else {
			reply(escapedHTML`Invalid room reference <code>${room_reference}</code>`)
		}
	},
	devtools: ({ roomCtx }) => {
		window.openModal(modals.roomStateExplorer(roomCtx.store))
	},
}

type BotArgMap = Record<string, BotArgumentValue>

interface CommandCallbackContext {
	client: Client
	mainScreen: MainScreenContextFields
	roomCtx: RoomContextData
	reply: (html: string) => void
}

type CommandCallback = (
	ctx: CommandCallbackContext,
	args: BotArgMap,
) => void

export function interceptCommand(
	client: Client,
	mainScreen: MainScreenContextFields,
	roomCtx: RoomContextData,
	spec: WrappedBotCommand,
	inputArgs: BotArgMap,
): boolean {
	const reply = (html: string) => {
		client.handleOutgoingEvent(makeFakeEvent(roomCtx.store.roomID, html), roomCtx.store)
	}
	if (spec.source !== fakeGomuksSender) {
		if (roomCtx.store.preferences.hide_fingerprint) {
			reply("External bot commands are disabled when the hide fingerprint option is enabled")
			return true
		}
		return false
	}
	const handler = commandHandlers[spec.command as CommandName]
	if (!handler) {
		return false
	}
	handler({ client, mainScreen, roomCtx, reply }, inputArgs)
	return true
}

const escapedHTML = (strings: TemplateStringsArray, ...values: string[]) => {
	for (let i = 0; i < values.length; i++) {
		values[i] = escapeHTML(values[i])
	}
	return String.raw({ raw: strings }, ...values)
}

function makeFakeEvent(roomID: RoomID, sanitized_html: string): RawDBEvent {
	const ts = Date.now()
	return {
		rowid: -ts,
		timeline_rowid: 0,
		room_id: roomID,
		event_id: `$gomuks-internal-fe-${ts}`,
		sender: fakeGomuksSender,
		type: "m.room.message",
		timestamp: ts,
		content: { msgtype: "m.text" },
		unsigned: {},
		local_content: { sanitized_html },
		unread_type: 0,
	}
}
