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
import { RoomListEntry, StateStore } from "@/api/statestore/main.ts"
import { DBSpaceEdge, RoomID } from "@/api/types"

export interface RoomListFilter {
	id: string
	include(room: RoomListEntry): boolean
}

export const DirectChatSpace: RoomListFilter = {
	id: "fi.mau.gomuks.direct_chats",
	include: room => !!room.dm_user_id,
}

export const UnreadsSpace: RoomListFilter = {
	id: "fi.mau.gomuks.unreads",
	include: room => Boolean(room.unread_messages
		|| room.unread_notifications
		|| room.unread_highlights
		|| room.marked_unread),
}

export class SpaceEdgeStore implements RoomListFilter {
	#children: DBSpaceEdge[] = []
	#childRooms: Set<RoomID> = new Set()
	#flattenedRooms: Set<RoomID> = new Set()
	#childSpaces: Set<SpaceEdgeStore> = new Set()
	readonly #parentSpaces: Set<SpaceEdgeStore> = new Set()

	constructor(public id: RoomID, private parent: StateStore) {
	}

	#addParent(parent: SpaceEdgeStore) {
		this.#parentSpaces.add(parent)
	}

	#removeParent(parent: SpaceEdgeStore) {
		this.#parentSpaces.delete(parent)
	}

	include(room: RoomListEntry) {
		return this.#flattenedRooms.has(room.room_id)
	}

	get children() {
		return this.#children
	}

	#updateFlattened(recalculate: boolean, added: Set<RoomID>) {
		if (recalculate) {
			let flattened = new Set(this.#childRooms)
			for (const space of this.#childSpaces) {
				flattened = flattened.union(space.#flattenedRooms)
			}
			this.#flattenedRooms = flattened
		} else if (added.size > 50) {
			this.#flattenedRooms = this.#flattenedRooms.union(added)
		} else if (added.size > 0) {
			for (const room of added) {
				this.#flattenedRooms.add(room)
			}
		}
	}

	#notifyParentsOfChange(recalculate: boolean, added: Set<RoomID>, stack: WeakSet<SpaceEdgeStore>) {
		if (stack.has(this)) {
			return
		}
		stack.add(this)
		for (const parent of this.#parentSpaces) {
			parent.#updateFlattened(recalculate, added)
			parent.#notifyParentsOfChange(recalculate, added, stack)
		}
		stack.delete(this)
	}

	set children(newChildren: DBSpaceEdge[]) {
		const newChildRooms = new Set<RoomID>()
		const newChildSpaces = new Set<SpaceEdgeStore>()
		for (const child of newChildren) {
			const spaceStore = this.parent.getSpaceStore(child.child_id)
			if (spaceStore) {
				newChildSpaces.add(spaceStore)
				spaceStore.#addParent(this)
			} else {
				newChildRooms.add(child.child_id)
			}
		}
		for (const space of this.#childSpaces) {
			if (!newChildSpaces.has(space)) {
				space.#removeParent(this)
			}
		}
		const addedRooms = newChildRooms.difference(this.#childRooms)
		const removedRooms = this.#childRooms.difference(newChildRooms)
		const didAddChildren = newChildSpaces.difference(this.#childSpaces).size > 0
		const recalculateFlattened = removedRooms.size > 0 || didAddChildren
		this.#children = newChildren
		this.#childRooms = newChildRooms
		this.#childSpaces = newChildSpaces
		if (this.#childSpaces.size > 0) {
			this.#updateFlattened(recalculateFlattened, addedRooms)
		} else {
			this.#flattenedRooms = newChildRooms
		}
		if (this.#parentSpaces.size > 0) {
			this.#notifyParentsOfChange(recalculateFlattened, addedRooms, new WeakSet())
		}
	}
}

export class SpaceOrphansSpace extends SpaceEdgeStore {
	static id = "fi.mau.gomuks.space_orphans"

	constructor(parent: StateStore) {
		super(SpaceOrphansSpace.id, parent)
	}

	include(room: RoomListEntry): boolean {
		return !super.include(room) && !room.dm_user_id
	}
}
