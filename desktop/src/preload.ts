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
import { contextBridge, ipcRenderer } from "electron"
import { TabInfo } from "./tabinfo.ts"

let subscriber = (_tabs: TabInfo[]) => {}
let cache: TabInfo[] | null  = null
let currentTabID: string = ""
let disableNotifications: boolean = false

contextBridge.exposeInMainWorld("gomuksDesktop", {
	isDesktop: true,
	getTabID() {
		return currentTabID
	},
	getDisableNotifications() {
		return disableNotifications
	},
	setNotificationCount: (count: number) => {
		ipcRenderer.send("set-notification-count", count)
	},
	subscribeToTabs: (listener: (tabs: TabInfo[]) => void) => {
		subscriber = listener
		if (cache) {
			listener(cache)
		}
		console.log("Tab subscriber updated, current cache:", cache)
	},
	switchTab: (tab: string) => {
		console.log("Sending tab switch request", tab)
		ipcRenderer.send("switch-tab", tab)
	},
	restartBackend: () => {
		ipcRenderer.send("restart-backend")
	},
	quitApp: () => {
		ipcRenderer.send("quit-app")
	},
})

ipcRenderer.on("open-matrix-uri", (_evt, url: string) => {
	if (!url.startsWith("matrix:")) {
		console.warn("Received non-matrix URI from main process:", url)
		return
	}
	console.log("Received matrix: URI from main process:", url)
	location.hash = `#/uri/${encodeURIComponent(url)}`
})

ipcRenderer.on("disable-notifications", () => {
	disableNotifications = true
})

ipcRenderer.on("tab-id", (_evt, name: string) => {
	currentTabID = name
})

ipcRenderer.on("update-tabs", (_evt, tabs) => {
	console.log("Received tab update", tabs)
	cache = tabs
	subscriber(tabs)
})
