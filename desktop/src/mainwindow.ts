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
import { BaseWindow, ipcMain } from "electron"
import { GomuksView } from "./webview.ts"
import { GomuksConfig } from "./config.ts"
import { TabInfo } from "./tabinfo.ts"

export class GomuksWindow {
	private window: BaseWindow | null = null
	private views: Map<string, GomuksView> = new Map()
	private activeView: GomuksView | null = null
	public config: GomuksConfig | null = null
	public quitting = false

	constructor() {
		ipcMain.on("switch-tab", (_evt, tab) => {
			const view = this.views.get(tab)
			if (!view) {
				console.log("Received switch tab request for unknown tab", tab)
			} else {
				console.log("Switching to", tab)
				view.focus()
			}
		})
	}

	public setFocused(view: GomuksView) {
		this.activeView = view
	}

	public getTabs(): TabInfo[] {
		return this.views.entries().map(([id, view]): TabInfo => ({
			id,
			displayname: view.config.displayname || id,
			icon: view.config.icon,
			unread: view.unreadCount,
			exited: view.exited,
		})).toArray()
	}

	public emitTabs() {
		const tabs = this.getTabs()
		for (const view of this.views.values()) {
			view.emitTabs(tabs)
		}
		console.debug("Sent tabs", tabs)
	}

	public initialize() {
		if (!this.config) {
			throw new Error("Config not loaded")
		}
		for (const backend of this.config.backends) {
			if (this.views.has(backend.name)) {
				throw new Error(`Duplicate backend name: ${backend.name}`)
			}
			const view = new GomuksView(backend, this)
			this.views.set(backend.name, view)
		}
	}

	public open = () => {
		if (this.window && BaseWindow.getAllWindows().length > 0) {
			if (this.window.isMinimized()) {
				this.window.restore()
			}
			this.window.focus()
			return this.window
		}
		const newWindow = new BaseWindow({
			width: 1280,
			height: 720,
			autoHideMenuBar: true,
		})
		newWindow.on("close", () => {
			if (this.window === newWindow) {
				this.window = null
			}
		})
		this.window = newWindow
		for (const view of this.views.values()) {
			view.onWindowCreated(newWindow)
		}
		this.emitTabs()
		return newWindow
	}

	public handleMatrixURI(uri: string) {
		this.activeView?.handleMatrixURI(uri)
	}

	toggleDevTools = () => this.activeView?.toggleDevTools()
}
