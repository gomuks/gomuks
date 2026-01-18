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
import { use, useMemo } from "react"
import { usePreference } from "@/api/statestore"
import ClientContext from "../ClientContext"
import { RoomContext } from "../roomview/roomcontext"
import LazyWidget from "./LazyWidget"

const elementCallHashParams = "#?" + new URLSearchParams({
	roomId: "$matrix_room_id",
	theme: "$org.matrix.msc2873.client_theme",
	userId: "$matrix_user_id",
	deviceId: "$org.matrix.msc3819.matrix_device_id",
	perParticipantE2EE: "$perParticipantE2EE",
	baseUrl: "$homeserverBaseURL",
	intent: "join_existing",
	hideHeader: "true",
	confineToRoom: "true",
	appPrompt: "false",
	lang: "en",
	fontScale: "1",
	rageshakeSubmitUrl: "https://element.io/bugreports/submit",
	preload: "false",
}).toString().replaceAll("%24", "$")

const embeddedElementCallURL = new URL("./element-call-embedded/index.html#", window.location.href)
embeddedElementCallURL.hash = elementCallHashParams

function makeElementCallURL(customBaseURL: string): string {
	if (customBaseURL) {
		const parsed = new URL(customBaseURL)
		if (!parsed.pathname.endsWith("/")) {
			parsed.pathname += "/"
		}
		parsed.pathname += "room"
		parsed.hash = elementCallHashParams
		return parsed.toString()
	} else {
		return embeddedElementCallURL.toString()
	}
}

const ElementCall = ({ onClose }: { onClose?: () => void }) => {
	const room = use(RoomContext)?.store ?? null
	const client = use(ClientContext)!
	const customBaseURL = usePreference(client.store, room, "element_call_base_url")
	const widgetInfo = useMemo(() => ({
		id: "app.gomuks.call",
		name: `Call in ${room?.meta.current.name}`,
		creatorUserId: client.userID,
		type: "m.call",
		url: makeElementCallURL(customBaseURL),
		waitForIframeLoad: false,
		data: {
			perParticipantE2EE: !!room?.meta.current.encryption_event,
			homeserverBaseURL: client.state.current?.is_logged_in ? client.state.current.homeserver_url : "",
		},
	}), [room, client, customBaseURL])
	if (!room || !client) {
		return null
	}
	return <LazyWidget info={widgetInfo} onClose={onClose} noPermissionPrompt={true} />
}

export default ElementCall
