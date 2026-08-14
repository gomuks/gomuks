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
import { CancellablePromise } from "@/util/promise.ts"
import RPCClient from "./rpc.ts"
import type { RPCEvent } from "./types"
import { checkUpdate } from "./wsclient.ts"

const PING_INTERVAL = 60_000

export default class SSEClient extends RPCClient {
	#conn: EventSource | null = null
	#pingInterval: ReturnType<typeof setInterval> | null = null
	#reconnectTimeout: ReturnType<typeof setTimeout> | null = null
	#lastReceivedEvt?: number
	#lastAckedEvt?: number
	#listenerID?: number
	#resumeRunID?: string
	#stopped = true
	#connectFailures = 0

	constructor() {
		super()
		window.addEventListener("focus", this.#onFocus)
	}

	#onFocus = () => {
		if (this.#reconnectTimeout !== null) {
			console.log("Window focused, reconnecting immediately")
			clearTimeout(this.#reconnectTimeout)
			this.#reconnectTimeout = null
			this.#restart()
		}
	}

	start() {
		this.#stopped = false
		this.#restart()
	}

	#restart() {
		try {
			const params = new URLSearchParams()
			if (this.#lastReceivedEvt && this.#resumeRunID) {
				params.set("run_id", this.#resumeRunID)
				params.set("last_received_event", this.#lastReceivedEvt.toString())
				if (this.#listenerID) {
					params.set("prev_listener_id", this.#listenerID.toString())
				}
			}
			const serverTS = this.getCachedServerTimestamp?.()
			if (serverTS) {
				params.set("last_server_ts", serverTS.toString())
			}
			const addr = `_gomuks/sse?${params}`
			console.info("Connecting to SSE", addr)
			this.#dispatchConnectionStatus(false, true, this.connect.current?.error ?? null, -1)
			this.#conn = new EventSource(addr)
			this.#conn.onmessage = this.#onMessage
			this.#conn.onopen = this.#onOpen
			this.#conn.onerror = this.#onError
		} catch (err) {
			this.#dispatchConnectionStatus(false, false, `Failed to create event source: ${err}`)
		}
	}

	stop() {
		if (this.#pingInterval !== null) {
			clearInterval(this.#pingInterval)
			this.#pingInterval = null
		}
		if (this.#reconnectTimeout !== null) {
			clearTimeout(this.#reconnectTimeout)
			this.#reconnectTimeout = null
		}
		this.#stopped = true
		this.#conn?.close()
	}

	get isConnected() {
		return this.#conn?.readyState === EventSource.OPEN
	}

	#onMessage = (ev: MessageEvent) => {
		const evt: RPCEvent = JSON.parse(ev.data)
		if (evt.request_id < 0) {
			this.#lastReceivedEvt = evt.request_id
		}
		if (evt.command === "run_id") {
			console.log("Received run ID", evt.data)
			this.#listenerID = evt.data.listener_id
			this.#resumeRunID = evt.data.run_id
			window.vapidPublicKey = evt.data.vapid_key
			checkUpdate(evt.data.etag)
		}
		this.event.emit(evt)
	}

	#onOpen = () => {
		console.info("SSE opened")
		this.#dispatchConnectionStatus(true, false, null)
		if (this.#pingInterval !== null) {
			clearInterval(this.#pingInterval)
		}
		this.#pingInterval = setInterval(this.#pingLoop, PING_INTERVAL)
		this.#connectFailures = 0
	}

	#onError = (ev: Event) => {
		console.error("SSE error:", ev)
		this.#conn?.close()
		if (this.#pingInterval !== null) {
			clearInterval(this.#pingInterval)
			this.#pingInterval = null
		}

		this.#connectFailures++
		const willReconnect = !this.#stopped && !this.#reconnectTimeout
		const backoff = Math.min(2 ** (this.#connectFailures - 4), 10) * 1000
		this.#dispatchConnectionStatus(
			false,
			willReconnect,
			`SSE disconnected`,
			Date.now() + backoff,
		)
		if (willReconnect) {
			console.log("Attempting to reconnect in", backoff, "ms")
			this.#reconnectTimeout = setTimeout(() => {
				console.log("Reconnecting now")
				this.#reconnectTimeout = null
				this.#restart()
			}, backoff)
		} else {
			console.log(`Not reconnecting (stopped=${this.#stopped}, reconnectTimeout=${this.#reconnectTimeout})`)
		}
	}

	#dispatchConnectionStatus(connected: boolean, reconnecting: boolean, error: string | null, nextAttempt?: number) {
		this.connect.emit({
			connected,
			reconnecting,
			error,
			nextAttempt: nextAttempt ?
				nextAttempt === -1
					? "currently trying to connect"
					: `next attempt at ${new Date(nextAttempt).toLocaleTimeString()}`
				: undefined,
		})
	}

	#pingLoop = () => {
		if (
			!this.#resumeRunID || !this.#listenerID || !this.#lastReceivedEvt
			|| this.#lastReceivedEvt === this.#lastAckedEvt
		) {
			return
		}
		const evtID = this.#lastReceivedEvt
		const params = new URLSearchParams({
			run_id: this.#resumeRunID,
			listener_id: this.#listenerID.toString(),
			last_received_event: evtID.toString(),
		})
		fetch(`_gomuks/sse/ping?${params}`, {
			method: "POST",
			signal: AbortSignal.timeout(PING_INTERVAL/2),
		}).then(() => {
			console.log("Successfully sent ping for", evtID)
			this.#lastAckedEvt = evtID
		}, err => {
			console.error("Failed to send ping:", err)
		})
	}

	send() {
		throw new Error("Raw sends aren't supported with SSE")
	}

	request<Req, Resp>(command: string, data: Req): CancellablePromise<Resp> {
		const ac = new AbortController()
		return new CancellablePromise((resolve, reject) => {
			fetch(`_gomuks/exec/${command}`, {
				method: "POST",
				body: JSON.stringify(data),
				headers: {
					"Content-Type": "application/json",
				},
			}).then(async res => {
				try {
					const payload = await res.json()
					if (res.ok) {
						resolve(payload)
					} else {
						reject(typeof payload.error === "string"
							? new Error(payload.error)
							: new Error(`Unexpected JSON response with status ${res.status}: ${payload}`))
					}
				} catch (err) {
					reject(new Error(`Non-JSON response with status ${res.status}: ${err}`))
				}
			}).catch(reject)
		}, ac.abort)
	}
}
