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
import initGomuksWasm from "./_gomuks.wasm?init"
import "./go_wasm_exec.js"
import initSqlite from "./sqlite_bridge.ts"

interface MediaResponse { buffer: Uint8Array, contentType: string, contentDisposition: string }

declare global {
	interface Window {
		meowDownloadMedia: (
			path: string,
			query: string,
			callbacks: {
				resolve: (data: MediaResponse) => void,
				reject: () => void
			},
		) => void
	}
}

async function setupMediaChannel() {
	const bc = new BroadcastChannel("wasmuks-media-download")
	const cache = await caches.open("wasmuks-media-v1")
	bc.addEventListener("message", async evt => {
		if (evt.data.type !== "request") {
			return
		}
		const parsedURL = new URL(evt.data.url)
		try {
			const result = await new Promise<MediaResponse>((resolve, reject) => {
				self.meowDownloadMedia(parsedURL.pathname, parsedURL.search, { resolve, reject })
			})
			const headers: Record<string, string> = {
				"Content-Type": result.contentType,
			}
			if (result.contentDisposition) {
				headers["Content-Disposition"] = result.contentDisposition
			}
			await cache.put(parsedURL, new Response(result.buffer, { status: 200, headers }))
			bc.postMessage({ type: "response", url: evt.data.url })
		} catch (err) {
			console.error("Error handling media download request:", err)
			await cache.put(parsedURL, new Response("Failed to download", { status: 500 }))
			bc.postMessage({ type: "response", url: evt.data.url, failed: true })
		}
	})
}

;(async () => {
	const go = new Go()
	await initSqlite()
	const instance = await initGomuksWasm(go.importObject)
	await setupMediaChannel()
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
