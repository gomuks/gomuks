/// <reference types="vite/client" />
/// <reference types="vite-plugin-svgr/client" />

import type Client from "@/api/client.ts"
import type { GCSettings } from "@/api/statestore"
import type { AndroidAPI, DesktopAPI } from "@/api/tabs.ts"
import type { MainScreenContextFields } from "@/ui/MainScreenContext.ts"
import type { openNestableModal, openNonNestableModal } from "@/ui/modal/contexts.ts"
import type { RoomContextData } from "@/ui/roomview/roomcontext.ts"

declare global {
	interface Window {
		client: Client
		activeRoomContext?: RoomContextData
		mainScreenContext: MainScreenContextFields
		openLightbox: (params: { src: string, alt: string }) => void
		gcSettings: GCSettings
		hackyOpenEventContextMenu?: string
		closeModal: () => void
		closeNestableModal: () => void
		openModal: openNonNestableModal
		openNestableModal: openNestableModal
		gomuksAndroid?: true | AndroidAPI
		gomuksDesktop?: DesktopAPI
		gomuksWebWasm?: boolean
		vapidPublicKey?: string
	}
}
