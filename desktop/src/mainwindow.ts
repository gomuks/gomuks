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
import { BaseWindow, WebContentsView, ipcMain } from "electron"
import path from "node:path"
import { GomuksView } from "./webview.ts"
import { GomuksConfig } from "./config.ts"
import { loadPage } from "./html.ts"

export class GomuksWindow {
	private window: BaseWindow | null = null
	private views: Map<string, GomuksView> = new Map()
	private activeView: GomuksView | null = null
	public dedicated = false
	public config: GomuksConfig | null = null
	private tabBar: WebContentsView | null = null

	constructor() {}

	public setFocused(view: GomuksView) {
		this.activeView = view
		this.emitUpdateTabs()
	}

	private emitUpdateTabs() {
		if (!this.tabBar) {
			console.log("No tab bar")
			return
		}
		const tabs = Array.from(this.views.entries().map(([name, view]) => ({
			name,
			active: view === this.activeView,
		})))
		this.tabBar.webContents.send("update-tabs", tabs)
		console.log("Sent tabs", tabs)
	}

	public initialize() {
		if (!this.config) {
			throw new Error("Config not loaded")
		}
		ipcMain.on("switch-tab", (evt, tab) => {
			if (evt.sender === this.tabBar?.webContents) {
				const view = this.views.get(tab)
				if (!view) {
					console.log("Received switch tab request for unknown tab", tab)
				} else {
					console.log("Switching to", tab)
					view.focus()
				}
			} else {
				console.log("Received switch tab request from unexpected sender", evt.sender, tab)
			}
		})
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
		if (!this.dedicated) {
			this.createTabBar(newWindow)
		}
		for (const view of this.views.values()) {
			view.onWindowCreated(newWindow)
		}
		return newWindow
	}

	private createTabBar(window: BaseWindow) {
		const tabBar = new WebContentsView({
			webPreferences: {
				preload: path.join(__dirname, "tabspreload.js"),
			},
		})
		tabBar.webContents.setWindowOpenHandler(() => ({ action: "deny" }))
		tabBar.webContents.on("will-navigate", evt => evt.preventDefault())
		const onResize = () => {
			const bounds = window.getContentBounds()
			tabBar.setBounds({ x: 0, y: 0, width: bounds.width, height: 32 })
		}
		window.on("resize", onResize)
		onResize()
		loadPage(tabBar.webContents, "tabs.html").then(() => this.emitUpdateTabs())
		if (process.env.NODE_ENV === "development") {
			tabBar.webContents.openDevTools({ mode: "detach" })
		}
		window.contentView.addChildView(tabBar)
		this.tabBar = tabBar
	}

	public handleMatrixURI(uri: string) {
		this.activeView?.handleMatrixURI(uri)
	}
}
