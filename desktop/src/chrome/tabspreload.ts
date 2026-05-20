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
import { type TabInfo } from "./tabs.tsx"

let subscriber = (_tabs: TabInfo[]) => {}
let cache: TabInfo[] | null  = null

contextBridge.exposeInMainWorld("tabAPI", {
	subscribe: (fn: (tabs: TabInfo[]) => void) => {
		subscriber = fn
		if (cache) {
			fn(cache)
		}
	},
	switchTo: (tab: string) => {
		console.log("Sending tab switch request", tab)
		ipcRenderer.send("switch-tab", tab)
	},
})


ipcRenderer.on("update-tabs", (_evt, tabs) => {
	cache = tabs
	subscriber(tabs)
	console.log("Received update", tabs)
})

console.log("Tab preload initialized")
