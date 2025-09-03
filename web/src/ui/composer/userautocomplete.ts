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
import { useRef } from "react"
import { AutocompleteMemberEntry, RoomStateStore, useBotCommands, useRoomMembers } from "@/api/statestore"
import { WrappedBotCommand } from "@/api/types"
import toSearchableString from "@/util/searchablestring.ts"

export function filterAndSort<T>(items: T[], key: keyof T, query: string): T[] {
	return items
		.map(user => ({ user, matchIndex: (user[key] as string).indexOf(query) }))
		.filter(({ matchIndex }) => matchIndex !== -1)
		.sort((e1, e2) => e1.matchIndex - e2.matchIndex)
		.map(({ user }) => user)
}

export function filter<T>(items: T[], key: keyof T, query: string): T[] {
	return items.filter(user => (user[key] as string).includes(query))
}

interface filterCache<T> {
	query: string
	result: T[]
	slicedResult?: T[]
}

export function useFiltered<T>(
	allItems: T[], key: keyof T, query: string, sort = true, slice = true,
): T[] {
	const prev = useRef<filterCache<T>>({ query: "", result: allItems })
	if (!query) {
		prev.current.query = ""
		prev.current.result = allItems
		prev.current.slicedResult = slice && allItems.length > 100 ? allItems.slice(0, 100) : undefined
	} else if (prev.current.query !== query) {
		prev.current.result = (sort ? filterAndSort : filter)(
			query.startsWith(prev.current.query) ? prev.current.result : allItems,
			key,
			query,
		)
		prev.current.slicedResult = prev.current.result.length > 100 && slice
			? prev.current.result.slice(0, 100)
			: undefined
		prev.current.query = query
	}
	return prev.current.slicedResult ?? prev.current.result
}

export function useFilteredMembers(
	room: RoomStateStore | undefined, query: string, sort = true, slice = true,
): AutocompleteMemberEntry[] {
	return useFiltered(useRoomMembers(room), "searchString", toSearchableString(query), sort, slice)
}

export function useFilteredCommands(
	room: RoomStateStore, query: string, sort = true, slice = true,
): WrappedBotCommand[] {
	return useFiltered(useBotCommands(room), "syntax", query, sort, slice)
}
