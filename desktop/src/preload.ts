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

contextBridge.exposeInMainWorld("gomuksDesktop", true)
contextBridge.exposeInMainWorld("gomuksDesktopSetNotificationCounts", (counts: number) => {
	ipcRenderer.send("set-notification-counts", counts)
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
	contextBridge.exposeInMainWorld("gomuksDesktopNotifications", true)
})
