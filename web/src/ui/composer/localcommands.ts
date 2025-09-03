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
import { BotArgumentValue, RawDBEvent, RoomID, WrappedBotCommand } from "@/api/types"
import type { CommandName } from "@/api/types/stdcommands.d.ts"
import { escapeHTML } from "@/util/markdown.ts"
import { matrixToToMatrixURI, parseMatrixURI } from "@/util/validation.ts"
import { MainScreenContextFields } from "../MainScreenContext.ts"
import { RoomContextData } from "../roomview/roomcontext.ts"

const commandHandlers: { [K in CommandName]?: CommandCallback } = {
	"join {room_reference} {reason}": ({ client, mainScreen, reply }, { room_reference }) => {
		if (typeof room_reference !== "string") {
			return
		}
		room_reference = matrixToToMatrixURI(room_reference) ?? room_reference
		let via: string[] = []
		if (room_reference.startsWith("matrix:")) {
			const parsed = parseMatrixURI(room_reference)
			if (parsed) {
				room_reference = parsed.identifier
				via = parsed.params.getAll("via")
			}
		}
		if (room_reference.startsWith("#")) {
			client.rpc.resolveAlias(room_reference).then(
				res => {
					mainScreen.setActiveRoom(res.room_id, {
						previewMeta: {
							alias: room_reference,
							via: res.servers.slice(0, 3),
						},
					})
				},
				err => reply(escapedHTML`Failed to resolve alias <code>${room_reference}</code>: ${err.message}`),
			)
		} else if (room_reference.startsWith("!")) {
			mainScreen.setActiveRoom(room_reference, {
				previewMeta: { via },
			})
		} else {
			reply(escapedHTML`Invalid room reference <code>${room_reference}</code>`)
		}
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
	if (spec.source !== fakeGomuksSender) {
		return false
	}
	const handler = commandHandlers[spec.syntax as CommandName]
	if (!handler) {
		return false
	}
	const reply = (html: string) => {
		client.handleOutgoingEvent(makeFakeEvent(roomCtx.store.roomID, html), roomCtx.store)
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
