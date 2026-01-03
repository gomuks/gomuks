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
import { useMemo, useRef } from "react"
import { AutocompleteMemberEntry, RoomStateStore, StateStore, useBotCommands, useRoomMembers } from "@/api/statestore"
import { WrappedBotCommand } from "@/api/types"
import toSearchableString from "@/util/searchablestring.ts"

type FilterFunc<T, K> = (items: T[], key: K, query: string) => T[]

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

const emptyArray: never[] = []

export function useFiltered<T, K extends keyof T>(
	allItems: T[], key: K, query: string, outputFilter: FilterFunc<T, K> = filterAndSort, slice = true, noEmpty = false,
): T[] {
	const prev = useRef<filterCache<T>>({ query: "", result: allItems })
	if (!query) {
		prev.current.query = ""
		prev.current.result = allItems
		prev.current.slicedResult = slice && allItems.length > 100 ? allItems.slice(0, 100) : undefined
		if (noEmpty) {
			prev.current.result = emptyArray
			if (slice) {
				prev.current.slicedResult = emptyArray
			}
		}
	} else if (prev.current.query !== query) {
		prev.current.result = outputFilter(
			(prev.current.query !== "" && query.startsWith(prev.current.query)) ? prev.current.result : allItems,
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
	return useFiltered(
		useRoomMembers(room), "searchString", toSearchableString(query), sort ? filterAndSort : filter, slice,
	)
}

export function useFilteredCommands(
	room: RoomStateStore, query: string, sort = true, slice = true,
): WrappedBotCommand[] {
	return useFiltered(useBotCommands(room), "command", query, sort ? filterAndSort : filter, slice)
}

export function filterAndSortRooms(items: RoomStateStore[], _: "searchString", query: string): RoomStateStore[] {
	return items
		.map(room => ({ room, matchIndex: room.searchString.indexOf(query) }))
		.filter(({ room, matchIndex }) => !room.tombstoned && matchIndex !== -1)
		.sort((e1, e2) =>
			(e1.matchIndex - e2.matchIndex)
			|| e2.room.meta.current.sorting_timestamp - e1.room.meta.current.sorting_timestamp,
		)
		.map(({ room }) => room)
}

export function useFilteredRooms(store: StateStore, query: string) {
	const rooms = useMemo(() => store.rooms.values().toArray(), [store])
	return useFiltered(rooms, "searchString", toSearchableString(query), filterAndSortRooms, true, true)
}
