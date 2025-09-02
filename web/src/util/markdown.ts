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
import { UserID } from "@/api/types"

export const escapeHTML = (input: string) => input
	.replaceAll("&", "&amp;")
	.replaceAll("<", "&lt;")
	.replaceAll(">", "&gt;")
	.replaceAll(`"`, "&quot;")
	.replaceAll("'", "&#039;")

export const escapeMarkdown = (input: string) => input
	.replace(/([\\`*_[\]()])/g, "\\$1")
	.replaceAll("<", "&lt;")
	.replaceAll(">", "&gt;")

export const escapeMarkdownAndURI = (input: string) => {
	return escapeMarkdown(encodeURIComponent(input))
}

export const makeMentionMarkdown = (displayname: string, userID: UserID) =>
	`[${escapeMarkdown(displayname).replace("\n", " ")}](https://matrix.to/#/${escapeMarkdownAndURI(userID)}) `
