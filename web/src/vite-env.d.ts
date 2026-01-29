/// <reference types="vite/client" />
/// <reference types="vite-plugin-svgr/client" />

import type Client from "@/api/client.ts"
import type { GCSettings, RoomStateStore } from "@/api/statestore"
import type { MainScreenContextFields } from "@/ui/MainScreenContext.ts"
import type { openNestableModal, openNonNestableModal } from "@/ui/modal/contexts.ts"
import type { RoomContextData } from "@/ui/roomview/roomcontext.ts"

declare global {
	interface Window {
		client: Client
		activeRoom?: RoomStateStore | null
		activeRoomContext?: RoomContextData
		mainScreenContext: MainScreenContextFields
		openLightbox: (params: { src: string, alt: string }) => void
		gcSettings: GCSettings
		hackyOpenEventContextMenu?: string
		closeModal: () => void
		closeNestableModal: () => void
		openModal: openNonNestableModal
		openNestableModal: openNestableModal
		gomuksAndroid?: true
		gomuksDesktop?: boolean
		gomuksWebWasm?: boolean
		vapidPublicKey?: string
	}
	interface Element {
		moveBefore?(newNode: ChildNode, referenceNode: ChildNode | null): ChildNode
	}
	// TODO remove after typescript 5.10/6.0
	interface Uint8ArrayConstructor {
		fromBase64(
			string: string,
			options?: {
				alphabet?: "base64" | "base64url" | undefined;
				lastChunkHandling?: "loose" | "strict" | "stop-before-partial" | undefined;
			},
		): Uint8Array<ArrayBuffer>
	}
}
