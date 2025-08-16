// gomuks - A Matrix client written in Go.
// Copyright (C) 2025 Tulir Asokan
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
import "./go_wasm_exec.js"
import initGomuksWasm from "./gomuks.wasm?init"
import initSqlite from "./sqlite_bridge.ts"

(async () => {
	const go = new Go()
	await initSqlite()
	const instance = await initGomuksWasm(go.importObject)
	await go.run(instance)
	self.postMessage({
		command: "wasm-connection",
		data: {
			connected: false,
			reconnecting: false,
			error: `Go process exited`,
		},
	})
})().catch(err => {
	console.error("Fatal error in wasm worker:", err)
	self.postMessage({
		command: "wasm-connection",
		data: {
			connected: false,
			reconnecting: false,
			error: `${err}`,
		},
	})
})
