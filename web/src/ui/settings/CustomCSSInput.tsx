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
import React, { Suspense, lazy, use, useRef, useState } from "react"
import { ScaleLoader } from "react-spinners"
import Client from "@/api/client.ts"
import { RoomStateStore, usePreference } from "@/api/statestore"
import { PreferenceContext, preferenceContextToInt } from "@/api/types/preferences"
import useEvent from "@/util/useEvent.ts"
import ClientContext from "../ClientContext.ts"
import type { SetPrefFunc } from "./SettingsView.tsx"

function getActiveCSSContext(client: Client, room?: RoomStateStore): PreferenceContext {
	if (room?.localPreferenceCache.custom_css !== undefined) {
		return PreferenceContext.RoomDevice
	} else if (room?.serverPreferenceCache.custom_css !== undefined) {
		return PreferenceContext.RoomAccount
	} else if (client.store.localPreferenceCache.custom_css !== undefined) {
		return PreferenceContext.Device
	} else {
		return PreferenceContext.Account
	}
}

const Monaco = lazy(() => import("../util/monaco.tsx"))

interface CustomCSSInputProps {
	setPref: SetPrefFunc
	room?: RoomStateStore
}

const CustomCSSInput = ({ setPref, room }: CustomCSSInputProps) => {
	const client = use(ClientContext)!
	usePreference(client.store, room ?? null, "custom_css")
	const appliedContext = getActiveCSSContext(client, room)
	const [context, setContext] = useState(appliedContext)
	const getContextText = (context: PreferenceContext) => {
		if (context === PreferenceContext.Account) {
			return client.store.serverPreferenceCache.custom_css
		} else if (context === PreferenceContext.Device) {
			return client.store.localPreferenceCache.custom_css
		} else if (context === PreferenceContext.RoomAccount && room) {
			return room.serverPreferenceCache.custom_css
		} else if (context === PreferenceContext.RoomDevice && room) {
			return room.localPreferenceCache.custom_css
		}
	}
	const origText = getContextText(context)
	const [text, setText] = useState(origText ?? "")
	const onChangeContext = (evt: React.ChangeEvent<HTMLSelectElement>) => {
		const newContext = evt.target.value as PreferenceContext
		setContext(newContext)
		setText(getContextText(newContext) ?? "")
	}
	const onChangeText = (evt: React.ChangeEvent<HTMLTextAreaElement>) => {
		setText(evt.target.value)
	}
	const onSave = useEvent(() => {
		if (vscodeOpen) {
			setText(vscodeContentRef.current)
			setPref(context, "custom_css", vscodeContentRef.current)
		} else {
			setPref(context, "custom_css", text)
		}
	})
	const onDelete = () => {
		setPref(context, "custom_css", undefined)
		setText("")
	}
	const [vscodeOpen, setVSCodeOpen] = useState(false)
	const vscodeContentRef = useRef("")
	const vscodeInitialContentRef = useRef("")
	const onClickVSCode = () => {
		vscodeContentRef.current = text
		vscodeInitialContentRef.current = text
		setVSCodeOpen(true)
	}
	const closeVSCode = useEvent(() => {
		setVSCodeOpen(false)
		setText(vscodeContentRef.current)
		vscodeContentRef.current = ""
	})
	return <div className="custom-css-input">
		<div className="header">
			<h3>Custom CSS</h3>
			<select value={context} onChange={onChangeContext}>
				<option value={PreferenceContext.Account}>Account</option>
				<option value={PreferenceContext.Device}>Device</option>
				{room && <>
					<option value={PreferenceContext.RoomAccount}>Room (account)</option>
					<option value={PreferenceContext.RoomDevice}>Room (device)</option>
				</>}
			</select>
			{preferenceContextToInt(context) < preferenceContextToInt(appliedContext) &&
				<span className="warning">
					&#x26a0;&#xfe0f; This context will not be applied, <code>{appliedContext}</code> has content
				</span>}
		</div>
		{vscodeOpen ? <div className="vscode-wrapper">
			<Suspense fallback={
				<div className="loader"><ScaleLoader width={40} height={80} color="var(--primary-color)"/></div>
			}>
				<Monaco
					initData={vscodeInitialContentRef.current}
					onClose={closeVSCode}
					onSave={onSave}
					contentRef={vscodeContentRef}
				/>
			</Suspense>
		</div> : <textarea value={text} onChange={onChangeText}/>}
		<div className="buttons">
			<button onClick={onClickVSCode}>Open in VS Code</button>
			{origText !== undefined && <button className="delete" onClick={onDelete}>Delete</button>}
			<button className="save primary-color-button" onClick={onSave} disabled={origText === text}>Save</button>
		</div>
	</div>
}

export default CustomCSSInput
