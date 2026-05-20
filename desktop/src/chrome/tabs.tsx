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
import React, { useSyncExternalStore } from "react"
import ReactDOM from "react-dom/client"
import "./tabs.css"

export interface TabInfo {
	name: string
	active: boolean
	unread: number
}

declare global {
	interface Window {
		tabAPI: {
			subscribe: (fn: (tabs: TabInfo[]) => void) => void
			switchTo: (tab: string) => void
		}
	}
}

let tabsCache: TabInfo[] = []
let tabListeners: (() => void)[] = []

function subscribeTabs(fn: () => void) {
	tabListeners.push(fn)
	return () => {
		tabListeners = tabListeners.filter(l => l !== fn)
	}
}

function getTabs() {
	return tabsCache
}

tabAPI.subscribe((tabs: TabInfo[]) => {
	tabsCache = tabs
	tabListeners.forEach(l => l())
})

const TabBar = () => {
	const tabs: TabInfo[] = useSyncExternalStore(subscribeTabs, getTabs)
	return <>
		{tabs.map(tab => <button
			key={tab.name}
			className={tab.active ? "active" : ""}
			onClick={() => tabAPI.switchTo(tab.name)}
		>{tab.name} {tab.unread}</button>)}
		{tabs.length === 0 ? "No tabs :(" : null}
	</>
}

ReactDOM.createRoot(document.querySelector("nav")!).render(<React.StrictMode><TabBar /></React.StrictMode>)
