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

import { WrappedBotCommand } from "@/api/types/commands.ts"
import { BotParameter } from "@/api/types/mxtypes.ts"

function makeFakeCommand(name: string, description: string, ...extraParams: BotParameter[]): WrappedBotCommand {
	return {
		source: "@gomuks",
		fake: true,
		command: name,
		parameters: [...extraParams, {
			key: "text",
			schema: {
				schema_type: "primitive",
				type: "string",
			},
		}],
		description: {
			"m.text": [{ "body": description }],
		},
	}
}

const FakeCommands: WrappedBotCommand[] = [
	makeFakeCommand("plain", "Send a plain text message without any formatting"),
	makeFakeCommand("html", "Send a formatted message with only HTML (no markdown)"),
	makeFakeCommand("rainbow", "Send a message with rainbow colors (markdown allowed)"),
	makeFakeCommand("me", "Send an m.emote message"),
	makeFakeCommand("notice", "Send an m.notice message"),
	makeFakeCommand("unencrypted", "Send an unencrypted message even if the room is encrypted"),
	makeFakeCommand(
		"rawinputbody", "Use the input text as the body field as-is, rather than re-parsing generated HTML",
	),
	makeFakeCommand("timestamp", "Send a message with a custom timestamp", {
		key: "timestamp",
		schema: {
			schema_type: "primitive",
			type: "integer",
		},
	}),
]

export function isFakeCommand(text: string): boolean {
	return text.startsWith("/plain ")
		|| text.startsWith("/me ")
		|| text.startsWith("/notice ")
		|| text.startsWith("/rainbow ")
		|| text.startsWith("/html ")
		|| text.startsWith("/timestamp ")
		|| text.startsWith("/unencrypted ")
		|| text.startsWith("/rawinputbody ")
}

export default FakeCommands
