// gomuks - A Matrix client written in Go.
// Copyright (C) 2025 Tulir Asokan
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
import { RefObject } from "react"
import { RoomStateStore } from "@/api/statestore"
import { EventID, MemDBEvent } from "@/api/types"
import { isMobileDevice } from "@/util/ismobile.ts"
import MediaUploadDialog, { UploadFileFunc } from "../composer/MediaUploadDialog.tsx"
import VoiceRecorder from "../composer/VoiceRecorder.tsx"
import CreateRoomView from "../roomview/CreateRoomView.tsx"
import { RoomContext, RoomContextData } from "../roomview/roomcontext.ts"
import RoomStateExplorer from "../settings/RoomStateExplorer.tsx"
import SettingsView from "../settings/SettingsView.tsx"
import EventContextModal from "../timeline/EventContextModal.tsx"
import EventEditHistory from "../timeline/EventEditHistory.tsx"
import JSONView from "../util/JSONView.tsx"
import { ShareModal } from "./ShareModal.tsx"
import { ModalState, NestableModalState, NonNestableModalState } from "./contexts.ts"

export function roomStateExplorer(room: RoomStateStore): ModalState {
	return {
		dimmed: true,
		boxed: true,
		innerBoxClass: "state-explorer-box",
		content: <RoomStateExplorer room={room} />,
	}
}

export function mediaUpload(
	file: File,
	doUploadFile: UploadFileFunc,
	isEncrypted: boolean = false,
	isVoice: boolean = false,
): NonNestableModalState {
	const blobURL = URL.createObjectURL(file)
	return {
		dimmed: true,
		boxed: true,
		innerBoxClass: "media-upload-modal-wrapper",
		onClose: () => URL.revokeObjectURL(blobURL),
		content: <MediaUploadDialog
			file={file}
			blobURL={blobURL}
			doUploadFile={doUploadFile}
			isEncrypted={isEncrypted}
			isVoice={isVoice}
		/>,
	}
}

export function voiceRecorder(
	openFileUploadModal: (file: File, isVoice?: true) => void,
	textInput?: RefObject<HTMLTextAreaElement | null>,
): NonNestableModalState {
	return {
		dimmed: true,
		boxed: true,
		boxClass: "voice-recorder-box",
		innerBoxClass: "voice-recorder",
		content: <VoiceRecorder onFinish={openFileUploadModal} />,
		onClose: () => !isMobileDevice && textInput?.current?.focus(),
	}
}

export function jsonView(data: unknown): NonNestableModalState {
	return {
		dimmed: true,
		boxed: true,
		content: <JSONView data={data}/>,
	}
}

export function shareRoom(room: RoomStateStore): NonNestableModalState {
	return {
		dimmed: true,
		boxed: true,
		content: <ShareModal room={room}/>,
	}
}

export function shareEvent(roomCtx: RoomContextData, evt: MemDBEvent): NonNestableModalState {
	return {
		dimmed: true,
		boxed: true,
		content: <RoomContext value={roomCtx}>
			<ShareModal room={roomCtx.store} evt={evt} />
		</RoomContext>,
	}
}

export function createRoom(): NonNestableModalState {
	return {
		dimmed: true,
		boxed: true,
		boxClass: "create-room-view-modal",
		content: <CreateRoomView />,
	}
}

export function settings(room: RoomStateStore): NestableModalState {
	return {
		dimmed: true,
		boxed: true,
		innerBoxClass: "settings-view",
		content: <SettingsView room={room} />,
		nestable: true,
	}
}

export function eventContext(roomCtx: RoomContextData, eventID: EventID): NestableModalState {
	if (roomCtx.threadParentRoom) {
		roomCtx = roomCtx.threadParentRoom
	}
	return {
		dimmed: true,
		boxed: true,
		boxClass: "event-context-modal",
		content: <EventContextModal roomCtx={roomCtx} eventID={eventID} key={eventID} />,
		nestable: true,
	}
}

export function eventEditHistory(roomCtx: RoomContextData, evt: MemDBEvent): NestableModalState {
	return {
		content: <EventEditHistory evt={evt} roomCtx={roomCtx}/>,
		dimmed: true,
		boxed: true,
		boxClass: "full-screen-mobile event-edit-history-wrapper",
		innerBoxClass: "event-edit-history-modal",
	}
}
