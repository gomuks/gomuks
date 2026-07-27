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
import { JSX, use, useCallback, useState } from "react"
import { RoomStateStore } from "@/api/statestore"
import {
	PreferenceContext,
	PreferenceValueType,
	Preferences,
} from "@/api/types/preferences"
import ClientContext from "../ClientContext.ts"
import BackendManager from "./BackendManager.tsx"
import CustomCSSInput from "./CustomCSSInput.tsx"
import EncryptionSettings from "./EncryptionSettings.tsx"
import MiscButtons from "./MiscButtons.tsx"
import RoomSettings from "./RoomSettings.tsx"
import SettingsDeck from "./SettingsDeck.tsx"
import "./SettingsView.css"

export type SetPrefFunc =
	(context: PreferenceContext, key: keyof Preferences, value: PreferenceValueType | undefined) => void

interface SettingsViewProps {
	room?: RoomStateStore
}

enum SettingsTab {
	RoomSettings,
	Preferences,
	CustomCSS,
	Encryption,
	MiscButtons,
	BackendManager
}

function getContent(tab: SettingsTab, setPref: SetPrefFunc, room?: RoomStateStore): JSX.Element {
	switch (tab) {
	case SettingsTab.RoomSettings:
		if (!room) {
			return <div className="settings-tab">Missing room</div>
		}
		return <RoomSettings room={room} />
	case SettingsTab.Preferences:
		return <SettingsDeck setPref={setPref} room={room} />
	case SettingsTab.CustomCSS:
		return <CustomCSSInput setPref={setPref} room={room} />
	case SettingsTab.Encryption:
		return <EncryptionSettings room={room} />
	case SettingsTab.MiscButtons:
		return <MiscButtons />
	case SettingsTab.BackendManager:
		return <BackendManager />
	default:
		return <div className="settings-tab">Unknown tab</div>
	}
}

const SettingsView = ({ room }: SettingsViewProps) => {
	const client = use(ClientContext)!
	const setPref = useCallback((
		context: PreferenceContext, key: keyof Preferences, value: PreferenceValueType | undefined,
	) => {
		if (context === PreferenceContext.Account) {
			client.rpc.setAccountData("fi.mau.gomuks.preferences", {
				...client.store.serverPreferenceCache,
				[key]: value,
			})
		} else if (context === PreferenceContext.Device) {
			if (value === undefined) {
				delete client.store.localPreferenceCache[key]
			} else {
				(client.store.localPreferenceCache[key] as PreferenceValueType) = value
			}
			if (key === "web_push") {
				client.registerWebPush()
			}
			if (key === "low_bandwidth" && !value) {
				client.store.deleteCache().then(
					() => console.log("Cleared cache after disabling low bandwidth mode"),
					err => console.error("Failed to clear cache after disabling low bandwidth mode:", err),
				)
			}
		} else if (context === PreferenceContext.RoomAccount && room) {
			client.rpc.setAccountData("fi.mau.gomuks.preferences", {
				...room.serverPreferenceCache,
				[key]: value,
			}, room.roomID)
		} else if (context === PreferenceContext.RoomDevice && room) {
			if (value === undefined) {
				delete room.localPreferenceCache[key]
			} else {
				(room.localPreferenceCache[key] as PreferenceValueType) = value
			}
		}
	}, [client, room])
	const [tab, setTab] = useState<SettingsTab>(room ? SettingsTab.RoomSettings : SettingsTab.Preferences)
	const makeTabButton = (toTab: SettingsTab, text: JSX.Element | string) => {
		return <button onClick={() => setTab(toTab)} className={toTab === tab ? "active" : ""}>
			{text}
		</button>
	}
	return <>
		<nav className="tabs">
			{room && makeTabButton(SettingsTab.RoomSettings, "Room settings")}
			{makeTabButton(SettingsTab.Preferences, "Preferences")}
			{makeTabButton(SettingsTab.CustomCSS, "Custom CSS")}
			{makeTabButton(SettingsTab.Encryption, "Encryption")}
			{makeTabButton(SettingsTab.MiscButtons, "Misc buttons")}
			{window.gomuksDesktop && makeTabButton(SettingsTab.BackendManager, "Backend manager")}
		</nav>
		<div className="settings-tab">
			{getContent(tab, setPref, room)}
		</div>
	</>
}

export default SettingsView
