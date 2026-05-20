/// <reference types="vite/client" />

import { type TabInfo } from "./tabs.tsx"

declare global {
	interface Window {
		tabAPI: {
			subscribe: (fn: (tabs: TabInfo[]) => void) => void
			switchTo: (tab: string) => void
		}
	}
}
