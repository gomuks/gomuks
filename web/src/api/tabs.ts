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
import { useSyncExternalStore } from "react"

// This should match desktop/src/tabinfo.ts
export interface TabInfo {
	id: string
	displayname: string
	icon?: string
	unread: number
	exited: boolean
}

let tabsCache: TabInfo[] = []
let tabListeners: (() => void)[] = []

const noopFunc = () => {}

function subscribeTabs(fn: () => void) {
	if (!window.gomuksDesktop) {
		return noopFunc
	}
	tabListeners.push(fn)
	return () => {
		tabListeners = tabListeners.filter(l => l !== fn)
	}
}

function getTabs() {
	return tabsCache
}

const noTabs = [[], "", 0, () => {}] as const

export function useTabs() {
	const tabs = useSyncExternalStore(subscribeTabs, getTabs)
	if (!window.gomuksDesktop) {
		return noTabs
	}
	const currentTabID = window.gomuksDesktop.getTabID() ?? ""
	const totalUnreads = tabs.reduce((acc, t) => acc + (t.id !== currentTabID ? t.unread : 0), 0)
	return [tabs, currentTabID, totalUnreads, window.gomuksDesktop.switchTab] as const
}

window.gomuksDesktop?.subscribeToTabs((tabs: TabInfo[]) => {
	tabsCache = tabs
	tabListeners.forEach(l => l())
})
