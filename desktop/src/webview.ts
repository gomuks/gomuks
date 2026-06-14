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
import { app, BaseWindow, shell, WebContentsView, desktopCapturer } from "electron"
import path from "node:path"
import contextMenu from "electron-context-menu"
import { EmbeddedBackend, GomuksBackend, RemoteBackend } from "./backend.ts"
import { type GomuksWindow } from "./mainwindow.ts"
import { loadPage } from "./html.ts"
import { TabInfo } from "./tabinfo.ts"

interface BaseBackendConfig {
	type: "embedded" | "remote"
	name: string
	displayname?: string
	icon?: string
}

export type BackendConfig = BaseBackendConfig & ({
	type: "embedded"
	env?: Record<string, string>
} | {
	type: "remote"
	address: string
	username: string
	password: string
})

export class GomuksView {
	public unreadCount: number = 0
	public exited = false
	private webContentsView: WebContentsView | null = null
	private readonly backend: GomuksBackend
	private readonly partition: string

	constructor(public config: BackendConfig, private parent: GomuksWindow) {
		this.partition = `persist:${config.name}`
		if (config.type === "embedded") {
			this.backend = new EmbeddedBackend(config.name, config.env, this.onBackendQuit, this.handleMatrixURI)
		} else {
			this.backend = new RemoteBackend(config.address, config.username, config.password)
		}
	}

	private onBackendQuit = () => {
		if (this.webContentsView && !this.parent.quitting) {
			this.exited = true
			loadPage(this.webContentsView.webContents, "exited.html")
		}
	}

	public emitTabs(tabs: TabInfo[] | null = null) {
		if (this.webContentsView) {
			this.webContentsView.webContents.send("update-tabs", tabs ?? this.parent.getTabs())
		}
	}

	public handleMatrixURI = (uri: string) => {
		this.focus()
		if (this.webContentsView) {
			this.webContentsView.webContents.send("open-matrix-uri", uri)
		} else {
			console.error("No web contents available to handle matrix URI:", uri)
		}
	}

	public toggleDevTools() {
		if (this.webContentsView) {
			this.webContentsView.webContents.toggleDevTools()
		}
	}

	public onWindowCreated(window: BaseWindow) {
		this.makeWebContentsView(window)
	}

	public focus = () => {
		const parentView = this.parent.open()
		if (!this.webContentsView || this.exited) {
			this.makeWebContentsView(parentView)
		} else {
			parentView.contentView.addChildView(this.webContentsView)
			this.parent.setFocused(this)
		}
	}

	private makeWebContentsView(parent: BaseWindow) {
		if (this.parent.quitting) {
			throw new Error("Can't create view when app is quitting")
		}
		this.webContentsView?.webContents.close({ waitForBeforeUnload: false })
		this.exited = false
		this.backend.start()
		const view = new WebContentsView({
			webPreferences: {
				preload: path.join(__dirname, "preload.js"),
				partition: this.partition,
			},
		})
		const onResize = () => {
			const bounds = parent.contentView.getBounds()
			view.setBounds({ x: 0, y: 0, width: bounds.width, height: bounds.height })
		}
		parent.contentView.on("bounds-changed", onResize)
		onResize()
		parent.contentView.addChildView(view)
		this.parent.setFocused(this)

		let serverURL: string | null = null
		view.webContents.ipc.on("set-notification-count", (_evt, count) => {
			this.unreadCount = count
			this.parent.emitTabs()
		})
		view.webContents.ipc.on("restart-backend", () => {
			if (this.exited) {
				this.focus()
			}
		})
		view.webContents.ipc.on("quit-app", () => {
			app.quit()
		})
		view.webContents.on("destroyed", () => {
			if (this.webContentsView === view) {
				this.webContentsView = null
			}
		})
		contextMenu({
			window: view,
			showInspectElement: true,
			showSaveImageAs: true,
			showSaveVideoAs: true,
		})
		view.webContents.setWindowOpenHandler(details => {
			if (!serverURL) {
				console.warn("Window open request before server URL is known:", details.url)
				return { action: "deny" }
			}
			if (details.url.startsWith(`${serverURL}/_gomuks/media/`)) {
				console.log("Downloading", details.url)
				view.webContents.downloadURL(details.url)
			} else {
				const parsedURL = new URL(details.url)
				switch (parsedURL.protocol) {
				case "http:":
				case "https:":
				case "mailto:":
				case "magnet:":
				case "ftp:":
					console.log("Opening", parsedURL.toString(), "externally")
					shell.openExternal(parsedURL.toString())
					break
				default:
					console.warn("Not opening unexpected protocol in URL", details.url)
				}
			}
			return { action: "deny" }
		})
		view.webContents.on("login", (event, authenticationResponseDetails, _authInfo, callback) => {
			event.preventDefault()
			if (serverURL && authenticationResponseDetails.url.startsWith(`${serverURL}/_gomuks/auth`)) {
				callback(this.backend.username, this.backend.password)
			} else {
				console.warn("Unexpected auth request from", authenticationResponseDetails.url)
				callback()
			}
		})
		view.webContents.session.setDisplayMediaRequestHandler((_, callback) => {
			if (process.env.XDG_SESSION_TYPE === "wayland") {
				// Wayland forces using the system picker and will always only return one source.
				desktopCapturer.getSources({ types: ["screen", "window"] }).then(
					sources => callback({ video: sources[0] }),
					err => {
						console.error("Wayland: failed to get user-selected source:", err)
						callback({ video: { id: "", name: "" } })
					},
				)
			} else {
				// TODO add support for windows
				console.error("Screen capture requested on non-Wayland session, which is not supported")
				callback({ video: { id: "", name: "" } })
			}
		}, { useSystemPicker: true })
		this.backend.address.then(addr => {
			serverURL = addr
			return view.webContents.loadURL(addr)
		}, err => {
			console.error("Failed to get backend address:", err)
		})
		if (this.backend instanceof EmbeddedBackend || process.env.GOMUKS_DESKTOP_DISABLE_NOTIFICATIONS === "true") {
			view.webContents.send("disable-notifications")
		}
		view.webContents.send("tab-id", {
			name: this.config.name,
			embedded: this.backend instanceof EmbeddedBackend,
		})

		if (process.env.NODE_ENV === "development") {
			view.webContents.openDevTools()
		}

		this.webContentsView = view
	}
}
