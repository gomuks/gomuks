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
import { app, Notification } from "electron"
import path from "node:path"
import { ChildProcess, spawn } from "node:child_process"
import { randomBytes } from "node:crypto"

const binaryName = "gomuks" + (process.platform === "win32" ? ".exe" : "")
const backendBinaryPath = process.env.GOMUKS_DESKTOP_BINARY_PATH ||
	(app.isPackaged ? path.join(process.resourcesPath, binaryName) : binaryName)

export interface GomuksBackend {
	address: Promise<string>
	username: string
	password: string

	start(): void
	stop(): void
}

export class RemoteBackend implements GomuksBackend {
	address: Promise<string>

	constructor(address: string, public username: string, public password: string) {
		this.address = Promise.resolve(address)
	}

	start() {}
	stop() {}
}

export class EmbeddedBackend implements GomuksBackend {
	private static instances: Map<string, EmbeddedBackend> | null = new Map<string, EmbeddedBackend>()
	private static stopAllPromise: Promise<void[]> | null = null
	private addressPromise: Promise<string> | null = null
	private readonly openNotifications = new Map<string, Map<number, Notification>>()
	private readonly desktopKey = randomBytes(32).toString("hex")
	private process: ChildProcess | null = null
	private waitForExit: Promise<void> = Promise.resolve()

	constructor(
		private profileName = "backend",
		private env: Record<string, string> = {},
		private onQuit: () => void,
		private handleMatrixURI: (uri: string) => void,
	) {}

	get username() {
		return "desktop-key"
	}

	get password() {
		return this.desktopKey
	}

	get address() {
		if (!this.addressPromise) {
			return Promise.reject(new Error("Backend not started"))
		}
		return this.addressPromise
	}

	start() {
		if (this.process) {
			return
		} else if (EmbeddedBackend.instances === null) {
			throw new Error("App is stopping")
		} else if (EmbeddedBackend.instances.has(this.profileName) && EmbeddedBackend.instances.get(this.profileName) !== this) {
			throw new Error("Duplicate backend profile name: " + this.profileName)
		}
		EmbeddedBackend.instances.set(this.profileName, this)
		console.log("Spawning", backendBinaryPath, "--desktop")
		const proc = spawn(backendBinaryPath, ["--desktop"], {
			stdio: ["ignore", "pipe", "inherit"],
			windowsHide: true,
			env: {
				...process.env,
				GOMUKS_LOGS_HOME: path.join(app.getPath("logs"), this.profileName),
				GOMUKS_CACHE_HOME: path.join(app.getPath("sessionData"), "gomuks-cache"),
				GOMUKS_ROOT: path.join(app.getPath("sessionData"), this.profileName),
				...this.env,
				GOMUKS_DESKTOP_KEY: this.desktopKey,
			},
		})
		let resolveServerAddress: (addr: string) => void
		let rejectServerAddress: ((err: Error) => void) | null
		this.addressPromise = new Promise((resolve, reject) => {
			resolveServerAddress = resolve
			rejectServerAddress = reject
		})
		this.waitForExit = new Promise(resolve => proc.once("exit", () => resolve()))
		this.process = proc
		proc.on("exit", code => {
			rejectServerAddress?.(new Error(`Backend exited with status ${code}`))
			this.process = null
			this.addressPromise = null
			EmbeddedBackend.instances?.delete(this.profileName)
			if (code !== 0) {
				console.error(`Backend exited with code ${code}`)
			} else {
				console.log("Backend exited normally")
			}
			this.onQuit()
		})
		proc.stdout.on("data", (output: string) => {
			try {
				const data = JSON.parse(output)
				if (data.started === true && data.address) {
					console.info("Got status from backend:", data)
					resolveServerAddress(`http://${data.address}`)
					rejectServerAddress = null
				} else if (data.desktop_notification) {
					this.onPushNotification(data.desktop_notification)
				} else if (data.dismiss_notification) {
					this.onDismissNotification(data.dismiss_notification)
				} else {
					console.warn("Unexpected backend output:", data)
				}
			} catch (err) {
				console.error("Failed to parse backend output:", output.toString())
			}
		})
	}

	public static async stopAll() {
		const instances = this.instances
		if (instances === null) {
			if (this.stopAllPromise) {
				await this.stopAllPromise
			}
			return
		}
		this.instances = null
		this.stopAllPromise = Promise.all(instances.values().map(instance => instance.stop()))
		await this.stopAllPromise
	}

	public static get runningInstances(): number {
		return this.instances?.size ?? 0
	}

	async stop() {
		if (!this.process) {
			return
		}
		this.process.kill("SIGTERM")
		const timeout = setTimeout(() => {
			console.warn("Backend did not exit after SIGTERM, sending SIGKILL")
			this.process?.kill("SIGKILL")
		}, 3000)
		await this.waitForExit
		clearTimeout(timeout)
	}

	private getNotifMap(roomID: string) {
		let map = this.openNotifications.get(roomID)
		if (!map) {
			map = new Map()
			this.openNotifications.set(roomID, map)
		}
		return map
	}

	private onPushNotification(data: PushNewMessage) {
		if (process.env.GOMUKS_DESKTOP_DISABLE_NOTIFICATIONS === "true") {
			return
		}
		const notif = new Notification({
			body: data.text,
			title: data.sender.name === data.room_name ? data.sender.name : `${data.sender.name} (${data.room_name})`,
			silent: !data.sound,
			// TODO this doesn't support webp
			// icon: data.sender.avatar,
			groupId: data.room_id,
			groupTitle: data.room_name,
		})
		this.getNotifMap(data.room_id).set(data.event_rowid, notif)
		notif.on("close", () => this.getNotifMap(data.room_id).delete(data.event_rowid))
		notif.on("click", () => {
			const targetURI = `matrix:roomid/${encodeURIComponent(data.room_id.slice(1))}/e/${encodeURIComponent(data.event_id.slice(1))}`
			console.log("Opening", targetURI, "after notification click")
			this.handleMatrixURI(targetURI)
		})
		console.log("Displaying notification for", data.event_id, "in", data.room_id)
		notif.show()
	}

	private onDismissNotification(roomID: string) {
		const map = this.openNotifications.get(roomID)
		if (map?.size) {
			console.log("Clearing active notifications in", roomID)
			for (const notif of map.values()) {
				notif.close()
			}
			map.clear()
		}
	}
}

interface NotificationUser {
	id: string
	name: string
	avatar?: string
}

interface PushNewMessage {
	desktop_notification: true
	timestamp: number
	event_id: string
	event_rowid: number

	room_id: string
	room_name: string
	room_avatar?: string
	sender: NotificationUser
	self: NotificationUser

	text: string
	image?: string
	mention?: true
	reply?: true
	sound?: true
}
