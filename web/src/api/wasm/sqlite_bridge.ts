// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

import type { Database, SAHPoolUtil, Sqlite3Static, WasmPointer } from "@sqlite.org/sqlite-wasm"
import sqlite3InitModule from "@sqlite.org/sqlite-wasm"

interface Meowlite extends Sqlite3Static {
	PoolUtil?: SAHPoolUtil
	meow?: {
		prepare: (connPtr: Database | WasmPointer, sql: string | WasmPointer) => {
			rc?: number,
			ptr?: WasmPointer,
		}
		last_insert_rowid: (connPtr: Database | WasmPointer) => number | string
		read_int64_column: (rowPtr: Database | WasmPointer, columnIndex: number) => number | string
	}
}

declare global {
	interface Window {
		sqlite3: Meowlite
	}
}

function safeFormatBigint(value: bigint): string | number {
	if (value > Number.MAX_SAFE_INTEGER || value < Number.MIN_SAFE_INTEGER) {
		return value.toString()
	}
	return Number(value)
}

async function init() {
	const sqlite3: Meowlite = await sqlite3InitModule({
		print: console.log,
		printErr: console.error,
	})

	sqlite3.meow = {
		prepare: (connPtr, sql) => {
			const stack = sqlite3.wasm.pstack.pointer
			try {
				const ppStmt = sqlite3.wasm.pstack.allocPtr()
				const pzTail = sqlite3.wasm.pstack.allocPtr()
				const rc = sqlite3.capi.sqlite3_prepare_v2(connPtr, sql, -1, ppStmt, pzTail)
				if (rc !== sqlite3.capi.SQLITE_OK) {
					return { rc }
				}
				if (sqlite3.wasm.peekPtr(pzTail) !== 0) {
					throw new Error("sqlite3_prepare_v2 returned a non-zero tail pointer, which is unsupported")
				}
				return { ptr: sqlite3.wasm.peekPtr(ppStmt) }
			} finally {
				sqlite3.wasm.pstack.restore(stack)
			}
		},
		last_insert_rowid: (connPtr) => {
			return safeFormatBigint(sqlite3.capi.sqlite3_last_insert_rowid(connPtr))
		},
		read_int64_column(rowPtr, columnIndex: number): number | string {
			return safeFormatBigint(sqlite3.capi.sqlite3_column_int64(rowPtr, columnIndex))
		},
	}

	sqlite3.PoolUtil = await sqlite3.installOpfsSAHPoolVfs({})

	self.sqlite3 = sqlite3
}

export default init
