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
import RPCClient, { ConnectionEvent } from "./rpc.ts"
import type { BaseRPCCommand, MediaMessageEventContent, RPCCommand } from "./types"
import WasmuksWorker from "./wasm/wasmuks.ts?worker"

interface WasmConnectionCommand extends BaseRPCCommand<ConnectionEvent> {
	command: "wasm-connection"
}

interface RawJSONCommand extends BaseRPCCommand<string> {
	command: RPCCommand["command"]
}

export default class WasmClient extends RPCClient {
	public readonly rpcMediaUpload = true
	protected isConnected = true
	#worker?: Worker

	async start() {
		this.#worker = new WasmuksWorker({ name: "gomuks-wasm-worker" })
		this.#worker.addEventListener("message", this.#onMessage)
		navigator.storage.persist().then(res => console.info("Storage persistence permission:", res))
		navigator.serviceWorker.register("/wasmuks-media-sw.js", {
			scope: "/",
		}).then(reg => {
			console.info("Media service worker registered", reg)
		}).catch(err => console.error("Failed to register media service worker", err))
	}

	async doAuth(): Promise<boolean> {
		return true
	}

	async uploadMedia(file: Blob, filename: string, encrypt: boolean): Promise<MediaMessageEventContent> {
		const request_id = this.nextRequestID
		const payload = await file.bytes()
		return new Promise((resolve, reject) => {
			if (!this.#worker) {
				reject(new Error("Worker not initialized"))
				return
			}
			this.pendingRequests.set(request_id, { resolve: resolve as ((value: unknown) => void), reject })
			this.#worker.postMessage({
				command: "wasm-upload",
				request_id,
				data: "",
				encrypt,
				filename,
				payload,
			}, [payload.buffer])
		})
	}

	#onMessage = (evt: MessageEvent<RawJSONCommand | WasmConnectionCommand>) => {
		let realEvtData: RPCCommand | WasmConnectionCommand
		if (typeof evt.data.data === "string") {
			realEvtData = {
				...evt.data,
				data: JSON.parse(evt.data.data),
			}
		} else if (evt.data.command === "wasm-connection") {
			realEvtData = evt.data
		} else {
			console.error("Unexpected message data:", evt.data)
			return
		}
		// console.debug("[RPC] Go -> JS", realEvtData)
		if (realEvtData.command === "wasm-connection") {
			this.connect.emit(realEvtData.data)
		} else {
			this.onCommand(realEvtData)
		}
	}

	async stop() {
		this.#worker?.terminate()
		this.#worker = undefined
	}

	protected send(data: RPCCommand) {
		if (!this.#worker) {
			throw new Error("Worker not initialized")
		}
		const payload = {
			command: data.command ?? "",
			request_id: data.request_id ?? 0,
			data: JSON.stringify(data.data ?? {}),
		}
		// console.debug("[RPC] JS -> Go", payload)
		this.#worker.postMessage(payload)
	}
}
