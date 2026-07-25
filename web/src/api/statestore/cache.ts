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
import {
	DBAccountData,
	DBInvitedRoom,
	DBRoomAccountData,
	DBSpaceEdge,
	RoomID,
	SyncCompleteData,
	SyncRoom,
} from "@/api/types"

const CACHE_DB = "gomuks-cache"

const KEY_TOP_LEVEL_SPACES = "top_level_spaces"
const KEY_SERVER_TIMESTAMP = "server_timestamp"

const KV_STORE = "kv_store"
const INVITED_ROOM_STORE = "invited_room"
const ROOM_STORE = "room"
const SPACE_EDGE_STORE = "space_edges"
const ACCOUNT_DATA_STORE = "account_data"
const ROOM_ACCOUNT_DATA_STORE = "room_account_data"

const allStores = [
	INVITED_ROOM_STORE, ROOM_STORE, SPACE_EDGE_STORE, ROOM_ACCOUNT_DATA_STORE, ACCOUNT_DATA_STORE, KV_STORE,
]


type indexedDBRoom = Required<Pick<SyncRoom, "meta" | "events" | "state">>

interface indexedDBSpaceEdges {
	room_id: RoomID
	edges: DBSpaceEdge[]
}

type Update = (txn: IDBTransaction) => void

export default class StateCache {
	private db?: IDBDatabase
	private queue: Map<string, Update> = new Map()
	private flushing = false
	private flushInterval?: ReturnType<typeof setInterval>

	async load() {
		const initStart = performance.now()
		const db = await this.initDB()
		console.info("Opened cache db in", performance.now() - initStart, "ms")
		const loadStart = performance.now()
		const data = await this.loadData(db)
		console.info("Loaded cache data in", performance.now() - loadStart, "ms")
		this.db = db
		this.flushInterval = setInterval(this.tryFlush, 60_000)
		db.onversionchange = () => this.close()
		return data
	}

	close() {
		if (this.flushInterval) {
			clearInterval(this.flushInterval)
			this.flushInterval = undefined
		}
		if (this.db) {
			this.db.close()
			this.db = undefined
		}
		console.log("Closed cache db")
	}

	private initDB = () => new Promise<IDBDatabase>((resolve, reject) => {
		const req = window.indexedDB.open(CACHE_DB, 2)
		req.onsuccess = () => resolve(req.result)
		req.onerror = () => reject(req.error ?? new Error("Open failed"))
		req.onblocked = () => console.warn("Cache db open blocked")
		req.onupgradeneeded = evt => {
			console.info("Upgrading cache db from", evt.oldVersion)
			if (evt.oldVersion === 1) {
				console.log("Deleting old stores")
				for (const store of [INVITED_ROOM_STORE, ROOM_STORE, SPACE_EDGE_STORE, ACCOUNT_DATA_STORE, KV_STORE]) {
					req.result.deleteObjectStore(store)
				}
			}
			req.result.createObjectStore(INVITED_ROOM_STORE, { keyPath: "room_id" })
			req.result.createObjectStore(ROOM_STORE)
			req.result.createObjectStore(SPACE_EDGE_STORE, { keyPath: "room_id" })
			req.result.createObjectStore(ACCOUNT_DATA_STORE, { keyPath: "type" })
			req.result.createObjectStore(KV_STORE, { keyPath: "key" })
			req.result.createObjectStore(ROOM_ACCOUNT_DATA_STORE, {
				keyPath: ["room_id", "type"],
			})
		}
	})

	static delete() {
		return new Promise<void>((resolve, reject) => {
			const res = window.indexedDB.deleteDatabase(CACHE_DB)
			res.onsuccess = () => resolve()
			res.onerror = () => reject(res.error ?? new Error("Delete failed"))
			res.onblocked = () => console.warn("Cache db delete blocked")
		})
	}

	private loadData = (db: IDBDatabase) => new Promise<SyncCompleteData | null>((resolve, reject) => {
		const txn = db.transaction(allStores, "readonly")
		const kv = txn.objectStore(KV_STORE)
		const serverTimestamp = kv.get(KEY_SERVER_TIMESTAMP)
		const topLevelSpaces = kv.get(KEY_TOP_LEVEL_SPACES)
		const invitedRooms = txn.objectStore(INVITED_ROOM_STORE).getAll()
		const roomsQuery = txn.objectStore(ROOM_STORE).getAll()
		const spaceEdges = txn.objectStore(SPACE_EDGE_STORE).getAll()
		const accountData = txn.objectStore(ACCOUNT_DATA_STORE).getAll()
		const roomAccountData = txn.objectStore(ROOM_ACCOUNT_DATA_STORE).getAll()
		txn.oncomplete = () => {
			if (!serverTimestamp.result || !topLevelSpaces.result
				|| !roomsQuery.result.length && !accountData.result.length) {
				resolve(null)
				return
			}
			const rooms = Object.fromEntries((roomsQuery.result as indexedDBRoom[])
				.map(r => [r.meta.room_id, r as SyncRoom]))
			for (const evt of roomAccountData.result as DBRoomAccountData[]) {
				if (rooms[evt.room_id]) {
					const ad = rooms[evt.room_id].account_data ?? {}
					ad[evt.type] = evt
					rooms[evt.room_id].account_data = ad
				}
			}
			resolve({
				server_timestamp: serverTimestamp.result.value as number,
				top_level_spaces: topLevelSpaces.result.value as RoomID[],
				invited_rooms: invitedRooms.result as DBInvitedRoom[],
				account_data: Object.fromEntries(accountData.result.map(e => [e.type, e])),
				rooms,
				space_edges: Object.fromEntries((spaceEdges.result as indexedDBSpaceEdges[])
					.map(e => [e.room_id, e.edges])),
				clear_state: true,
			})
		}
		txn.onerror = () => reject(txn.error)
		txn.onabort = () => reject(txn.error)
	})

	clear = () => new Promise<void>((resolve, reject) => {
		if (!this.db) {
			resolve()
			return
		}
		const txn = this.db.transaction(allStores, "readwrite")
		txn.oncomplete = () => resolve()
		txn.onerror = () => reject(txn.error)
		txn.onabort = () => reject(txn.error)
		for (const store of allStores) {
			txn.objectStore(store).clear()
		}
		this.db = undefined
	})

	tryFlush = async () => {
		try {
			const flushResult = await this.flush()
			console.log("Flushed cache to indexeddb:", flushResult)
		} catch (err) {
			console.error("Failed to flush cache:", err)
		}
	}

	private flush = () => new Promise<string>((resolve, reject) => {
		if (!this.db) {
			reject(new Error("No database"))
			return
		} else if (this.flushing) {
			reject(new Error("Already flushing"))
			return
		} else if (!this.queue.size) {
			resolve("Nothing to flush")
			return
		}
		this.flushing = true
		const flushing = this.queue
		this.queue = new Map()
		const start = performance.now()
		const itemCount = flushing.size
		const txn = this.db.transaction(allStores, "readwrite")
		txn.oncomplete = () => {
			this.flushing = false
			resolve(`${itemCount} items in ${performance.now() - start}ms`)
		}
		txn.onerror = () => {
			for (const [key, value] of this.queue.entries()) {
				flushing.set(key, value)
			}
			this.queue = flushing
			reject(txn.error)
		}
		txn.onabort = txn.onerror
		for (const value of flushing.values()) {
			value(txn)
		}
		txn.commit()
	})

	private addToQueue(key: string, fn: Update) {
		if (!this.db) {
			return
		}
		this.queue.set(key, fn)
	}

	private setKV(key: string, value: unknown) {
		this.addToQueue(key, txn => txn.objectStore(KV_STORE).put({ key, value }))
	}

	setServerTimestamp(timestamp: number) {
		this.setKV(KEY_SERVER_TIMESTAMP, timestamp)
	}

	setTopLevelSpaces(topLevelSpaces: RoomID[]) {
		this.setKV(KEY_TOP_LEVEL_SPACES, topLevelSpaces)
	}

	setSpaceEdges(room_id: RoomID, edges: DBSpaceEdge[]) {
		this.addToQueue(`space:${room_id}`, txn => txn.objectStore(SPACE_EDGE_STORE).put({ room_id, edges }))
	}

	setAccountData(evt: DBAccountData) {
		this.addToQueue(`ad:${evt.type}`, txn => txn.objectStore(ACCOUNT_DATA_STORE).put(evt))
	}

	setRoomAccountData(evt: DBRoomAccountData) {
		this.addToQueue(`ad:${evt.type}:${evt.room_id}`, txn => txn.objectStore(ROOM_ACCOUNT_DATA_STORE).put(evt))
	}

	setInvitedRoom(evt: DBInvitedRoom) {
		this.addToQueue(`invite:${evt.room_id}`, txn => txn.objectStore(INVITED_ROOM_STORE).put(evt))
	}

	deleteInvitedRoom(roomID: RoomID) {
		this.addToQueue(`invite:${roomID}`, txn => txn.objectStore(INVITED_ROOM_STORE).delete(roomID))
	}

	setRoom(evt: indexedDBRoom) {
		this.addToQueue(`room:${evt.meta.room_id}`, txn => txn.objectStore(ROOM_STORE).put(evt, evt.meta.room_id))
	}

	deleteRoom(roomID: RoomID) {
		this.addToQueue(`room:${roomID}`, txn => txn.objectStore(ROOM_STORE).delete(roomID))
	}
}
