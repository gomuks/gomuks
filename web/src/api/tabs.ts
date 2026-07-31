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

export interface WrapperAPI {
	getTabID(): string
	isEmbedded(): boolean
	setNotificationCount: (count: number) => void
	switchTab: (tab: string) => void
	updateTab: (tab: TabInfoUpdate) => Promise<void>
	deleteTab: (tab: string) => Promise<void>
	restartBackend: () => void
}

export interface DesktopAPI extends WrapperAPI {
	isDesktop: true
	getDisableNotifications(): boolean
	subscribeToTabs: (listener: (tabs: TabInfo[]) => void) => void
	quitApp: () => void
}

export interface AndroidAPI extends WrapperAPI {
	isAndroid: true
}

let tabsCache: readonly TabInfo[] = []
let tabListeners: (() => void)[] = []

const noopFunc = () => {}

const api = (() => {
	if (window.gomuksDesktop) {
		return window.gomuksDesktop
	}
	if (window.gomuksAndroid && typeof window.gomuksAndroid === "object") {
		return window.gomuksAndroid
	}
	return null
})()

function subscribeTabs(fn: () => void) {
	if (!api) {
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
	updateTab: (update: TabInfoUpdate) => Promise<void>
	deleteTab: (id: string) => Promise<void>
	hasTabs: boolean
}

const noTabs: UseTabsValue = {
	tabs: [],
	currentTabID: "",
	totalUnreads: 0,
	switchTab: () => {},
	updateTab: async () => {},
	deleteTab: async () => {},
	hasTabs: false,
}

export function hasTabs(): boolean {
	return Boolean(api)
}

export function getTabsAPI(): WrapperAPI | null {
	return api
}

export function useTabs(): UseTabsValue {
	const tabs = useSyncExternalStore(subscribeTabs, getTabs)
	if (!api) {
		return noTabs
	}
	const currentTabID = api.getTabID() ?? ""
	const totalUnreads = tabs.reduce((acc, t) => acc + (t.id !== currentTabID ? t.unread : 0), 0)
	return {
		tabs, currentTabID, totalUnreads,
		hasTabs: true,
		switchTab: api.switchTab,
		updateTab: api.updateTab,
		deleteTab: api.deleteTab,
	}
}

function onTabUpdate(tabs: TabInfo[]) {
	tabsCache = tabs
	tabListeners.forEach(l => l())
}

window.gomuksDesktop?.subscribeToTabs(onTabUpdate)

if (window.gomuksAndroid && typeof window.gomuksAndroid === "object") {
	window.addEventListener("GomuksAndroidTabUpdate", (evt: CustomEventInit<string>) => {
		onTabUpdate(JSON.parse(evt.detail!))
	})
}
