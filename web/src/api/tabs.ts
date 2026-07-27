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

// This should match desktop/src/webview.ts
export interface TabInfo {
	type: "embedded" | "remote"
	id: string
	displayname: string
	icon?: string
	disable_notifications: boolean

	address?: string
	username?: string
	password?: string

	unread: number
	exited: boolean
}

export type TabInfoUpdate = Omit<TabInfo, "unread" | "exited">

let tabsCache: readonly TabInfo[] = []
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

interface UseTabsValue {
	tabs: readonly TabInfo[]
	currentTabID: string
	totalUnreads: number
	switchTab: (id: string) => void
}

interface NoTabs extends UseTabsValue {
	hasTabs: false
}

interface HasTabs extends UseTabsValue {
	updateTab: (update: TabInfoUpdate) => Promise<void>
	deleteTab: (id: string) => Promise<void>
	hasTabs: true
}

const noTabs: NoTabs = {
	tabs: [],
	currentTabID: "",
	totalUnreads: 0,
	switchTab: () => {},
	hasTabs: false,
}

export function hasTabs(): boolean {
	return Boolean(window.gomuksDesktop)
}

export function useTabs(): HasTabs | NoTabs {
	const tabs = useSyncExternalStore(subscribeTabs, getTabs)
	if (!window.gomuksDesktop) {
		return noTabs
	}
	const currentTabID = window.gomuksDesktop.getTabID() ?? ""
	const totalUnreads = tabs.reduce((acc, t) => acc + (t.id !== currentTabID ? t.unread : 0), 0)
	return {
		tabs, currentTabID, totalUnreads,
		hasTabs: true,
		switchTab: window.gomuksDesktop.switchTab,
		updateTab: window.gomuksDesktop.updateTab,
		deleteTab: window.gomuksDesktop.deleteTab,
	}
}

window.gomuksDesktop?.subscribeToTabs((tabs: TabInfo[]) => {
	tabsCache = tabs
	tabListeners.forEach(l => l())
})
