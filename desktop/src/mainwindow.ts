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
import path from "node:path"
import {
	BaseWindow, Menu, MenuItemConstructorOptions, Tray, WebContentsView, app, autoUpdater, dialog, ipcMain, nativeImage,
} from "electron"
import { EmbeddedBackend } from "./backend.ts"
import { GomuksConfig, isValidTabID, saveConfig, tabInfoToConfig } from "./config.ts"
import { GomuksView, TabInfo, TabInfoUpdate } from "./webview.ts"

export class GomuksWindow {
	private window: BaseWindow | null = null
	private views: Map<string, GomuksView> = new Map()
	private activeView: GomuksView | null = null
	private tray: Tray | null = null
	public config: GomuksConfig | null = null
	public quitting = false

	constructor() {
		ipcMain.on("switch-tab", (_evt, tab: string) => {
			if (!isValidTabID(tab)) {
				console.log("Received switch tab request for invalid tab", tab)
				return
			}
			const view = this.views.get(tab)
			if (!view) {
				console.log("Received switch tab request for unknown tab", tab)
			} else {
				console.log("Switching to", tab)
				view.focus()
			}
		})
		ipcMain.handle("delete-tab", (_evt, tab: string) => this.deleteTab(tab))
		ipcMain.handle("update-tab", (_evt, tab: TabInfoUpdate) => this.updateTab(tab))
		app.on("activate", this.open)
		app.on("second-instance", (_event, commandLine) => {
			console.log("Got second instance with", commandLine)
			this.open()

			const uri = commandLine.pop()
			if (uri?.startsWith("matrix:")) {
				this.handleMatrixURI(uri)
			}
		})
		app.on("open-url", (_event, url) => {
			this.handleMatrixURI(url)
		})
	}

	private async deleteTab(tab: string) {
		if (!this.config) {
			throw new Error("Config not loaded")
		} else if (!isValidTabID(tab)) {
			throw new Error("Invalid tab ID")
		}
		const view = this.views.get(tab)
		if (view) {
			view.destroy()
			this.views.delete(tab)
		}
		this.config.backends = this.config.backends.filter(b => b.id !== tab)
		await saveConfig(this.config)
		this.emitTabs()
		await EmbeddedBackend.deleteData(tab)
	}

	private async updateTab(tab: TabInfoUpdate) {
		const config = this.config
		if (!config) {
			throw new Error("Config not loaded")
		}

		const backendIdx = config.backends.findIndex(b => b.id === tab.id)
		const backend = backendIdx !== -1 ? config.backends[backendIdx] : undefined
		const newCfg = tabInfoToConfig(tab, backend)

		const view = this.views.get(tab.id)
		if (view) {
			view.destroy()
		}
		if (backendIdx === -1) {
			config.backends.push(newCfg)
		} else {
			config.backends[backendIdx] = newCfg
		}
		const newView = new GomuksView(newCfg, this)
		this.views.set(tab.id, newView)
		if (this.window) {
			newView.onWindowCreated(this.window)
		}
		await saveConfig(config)
		this.emitTabs()
	}

	removeView(view: WebContentsView) {
		this.window?.contentView.removeChildView(view)
	}

	createTray() {
		const trayIconPath = path.join(
			app.isPackaged ? process.resourcesPath : app.getAppPath(),
			process.platform === "darwin" ? "trayTemplate@2x.png" : "tray@2x.png",
		)
		this.tray = new Tray(nativeImage.createFromPath(trayIconPath))
		this.updateTrayMenu()
	}

	hasTray() {
		return !this.config?.disable_tray
	}

	updateTrayMenu() {
		const items: MenuItemConstructorOptions[] = this.views.entries().map(([id, view]) => ({
			label: this.views.size === 1 ? "Open gomuks" : `Open: ${view.config.displayname || id}`,
			click: view.focus,
		})).toArray()
		items.push({
			label: "Check for updates",
			click: () => {
				autoUpdater.once("update-not-available", () => {
					dialog.showMessageBox({
						type: "info",
						title: "gomuks",
						message: "No updates found",
					})
				})
				autoUpdater.checkForUpdates()
			},
			enabled: app.isPackaged && (process.platform === "darwin" || process.platform === "win32"),
		})
		items.push({
			label: "Quit gomuks",
			click: app.quit,
		})
		this.tray?.setContextMenu(Menu.buildFromTemplate(items))
	}

	public setFocused(view: GomuksView) {
		this.activeView = view
	}

	public getTabs(): TabInfo[] {
		return this.views.values().map(view => view.tabInfo).toArray()
	}

	public emitTabs() {
		const tabs = this.getTabs()
		for (const view of this.views.values()) {
			view.emitTabs(tabs)
		}
		// console.debug("Sent tabs", tabs)
	}

	public initialize() {
		if (!this.config) {
			throw new Error("Config not loaded")
		}
		for (const backend of this.config.backends) {
			if (this.views.has(backend.id)) {
				throw new Error(`Duplicate backend name: ${backend.id}`)
			}
			const view = new GomuksView(backend, this)
			this.views.set(backend.id, view)
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
