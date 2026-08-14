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
import React, { use, useEffect, useRef, useState } from "react"
import { RoomStateStore, usePreferences } from "@/api/statestore"
import {
	Preference,
	PreferenceContext,
	PreferenceValueType,
	Preferences,
	preferences,
} from "@/api/types/preferences"
import ClientContext from "../ClientContext.ts"
import Toggle from "../util/Toggle.tsx"
import type { SetPrefFunc } from "./SettingsView.tsx"
import CloseIcon from "@/icons/close.svg?react"

interface PreferenceCellProps<T extends PreferenceValueType> {
	context: PreferenceContext
	name: keyof Preferences
	pref: Preference<T>
	setPref: SetPrefFunc
	value: T | undefined
	inheritedValue: T
}

const makeRemover = (
	context: PreferenceContext, setPref: SetPrefFunc, name: keyof Preferences, value: PreferenceValueType | undefined,
) => {
	if (value === undefined) {
		return null
	}
	return <button onClick={() => setPref(context, name, undefined)}><CloseIcon /></button>
}

const makeRemoverPacked = (props: PreferenceCellProps<PreferenceValueType>) => {
	return makeRemover(props.context, props.setPref, props.name, props.value)
}

const BooleanPreferenceCell = ({ context, name, setPref, value, inheritedValue }: PreferenceCellProps<boolean>) => {
	return <div className="preference boolean-preference">
		<Toggle checked={value ?? inheritedValue} onChange={evt => setPref(context, name, evt.target.checked)}/>
		{makeRemover(context, setPref, name, value)}
	</div>
}

const useLocalValue = <T extends PreferenceValueType = number | string>(
	{ context, name, setPref, value, inheritedValue }: PreferenceCellProps<T>,
) => {
	const realVal = value ?? inheritedValue
	const [localVal, setLocalVal] = useState(realVal)
	const saveTimeout = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
	useEffect(() => {
		clearTimeout(saveTimeout.current)
		setLocalVal(realVal)
	}, [realVal])
	const onChange = (evt: React.ChangeEvent<HTMLInputElement>) => {
		if (typeof realVal === "number") {
			setLocalVal(evt.target.valueAsNumber as T)
		} else {
			setLocalVal(evt.target.value as T)
		}
		clearTimeout(saveTimeout.current)
		saveTimeout.current = setTimeout(() => {
			save()
		}, 500)
	}
	const save = () => {
		clearTimeout(saveTimeout.current)
		if (localVal !== realVal && (localVal || realVal)) {
			setPref(context, name, localVal)
		}
	}
	return [localVal, onChange, save] as const
}

const TextPreferenceCell = (props: PreferenceCellProps<string>) => {
	const [localVal, onChange, save] = useLocalValue(props)
	return <div className="preference string-preference">
		<input value={localVal} onChange={onChange} onBlur={save} />
		{makeRemoverPacked(props)}
	</div>
}

const NumberPreferenceCell = (props: PreferenceCellProps<number>) => {
	const [localVal, onChange, save] = useLocalValue(props)
	return <div className="preference number-preference">
		<input
			type={props.pref.numberType ?? "number"}
			min={props.pref.minValue}
			max={props.pref.maxValue}
			value={localVal}
			onChange={onChange}
			onBlur={save}
			onMouseUp={props.pref.numberType === "range" ? save : undefined}
		/>
		{makeRemoverPacked(props)}
	</div>
}

const SelectPreferenceCell = ({ context, name, pref, setPref, value, inheritedValue }: PreferenceCellProps<string>) => {
	if (!pref.allowedValues) {
		return null
	}
	return <div className="preference select-preference">
		<select value={value ?? inheritedValue} onChange={evt => setPref(context, name, evt.target.value)}>
			{pref.allowedValues.map((value, i) =>
				<option key={i} value={value}>{pref.valueLabels ? pref.valueLabels[i] : value}</option>)}
		</select>
		{makeRemover(context, setPref, name, value)}
	</div>
}

interface PreferenceRowProps {
	name: keyof Preferences
	pref: Preference
	setPref: SetPrefFunc
	globalServer?: PreferenceValueType
	globalLocal?: PreferenceValueType
	roomServer?: PreferenceValueType
	roomLocal?: PreferenceValueType
	hideRoom: boolean
}

const customUIPrefs = new Set([
	"custom_css",
] as (keyof Preferences)[])

const PreferenceRow = ({
	name, pref, setPref, globalServer, globalLocal, roomServer, roomLocal, hideRoom,
}: PreferenceRowProps) => {
	const prefType = typeof pref.defaultValue
	if (customUIPrefs.has(name)) {
		return null
	}
	const makeContentCell = (
		context: PreferenceContext,
		val: PreferenceValueType | undefined,
		inheritedVal: PreferenceValueType,
	) => {
		if (!pref.allowedContexts.includes(context)) {
			return <div className="empty-cell" />
		}
		if (prefType === "boolean") {
			return <BooleanPreferenceCell
				name={name}
				setPref={setPref}
				context={context}
				pref={pref as Preference<boolean>}
				value={val as boolean | undefined}
				inheritedValue={inheritedVal as boolean}
			/>
		} else if (pref.allowedValues) {
			return <SelectPreferenceCell
				name={name}
				setPref={setPref}
				context={context}
				pref={pref as Preference<string>}
				value={val as string | undefined}
				inheritedValue={inheritedVal as string}
			/>
		} else if (prefType === "string") {
			return <TextPreferenceCell
				name={name}
				setPref={setPref}
				context={context}
				pref={pref as Preference<string>}
				value={val as string | undefined}
				inheritedValue={inheritedVal as string}
			/>
		} else if (prefType === "number") {
			return <NumberPreferenceCell
				name={name}
				setPref={setPref}
				context={context}
				pref={pref as Preference<number>}
				value={val as number | undefined}
				inheritedValue={inheritedVal as number}
			/>
		} else {
			return <div className="empty-cell" />
		}
	}
	let inherit: PreferenceValueType
	return <>
		<div className="name" title={pref.description}>{pref.displayName}</div>
		{makeContentCell(PreferenceContext.Account, globalServer, inherit = pref.defaultValue)}
		{makeContentCell(PreferenceContext.Device, globalLocal, inherit = globalServer ?? inherit)}
		{!hideRoom ? <>
			{makeContentCell(PreferenceContext.RoomAccount, roomServer, inherit = globalLocal ?? inherit)}
			{makeContentCell(PreferenceContext.RoomDevice, roomLocal, inherit = roomServer ?? inherit)}
		</> : <>
			<div className="empty-cell" />
			<div className="empty-cell" />
		</>}
	</>
}

interface SettingsDeckProps {
	setPref: SetPrefFunc
	room?: RoomStateStore
}

const SettingsDeck = ({ room, setPref }: SettingsDeckProps) => {
	const client = use(ClientContext)!
	usePreferences(client.store, room ?? null)
	const globalServer = client.store.serverPreferenceCache
	const globalLocal = client.store.localPreferenceCache
	const roomServer = room?.serverPreferenceCache
	const roomLocal = room?.localPreferenceCache
	return <div className="preference-table">
		<div className="name" style={{ height: "2.5rem" }}>Name</div>
		<div className="name">Account</div>
		<div className="name">Device</div>
		{room ? <>
			<div className="name">Room (account)</div>
			<div className="name">Room (device)</div>
		</> : <>
			<div className="name"></div>
			<div className="name"></div>
		</>}
		{Object.entries(preferences).map(([key, pref]) =>
			!pref.hidden ? <PreferenceRow
				key={key}
				name={key as keyof Preferences}
				pref={pref}
				setPref={setPref}
				globalServer={globalServer[key as keyof Preferences]}
				globalLocal={globalLocal[key as keyof Preferences]}
				roomServer={roomServer?.[key as keyof Preferences]}
				roomLocal={roomLocal?.[key as keyof Preferences]}
				hideRoom={!room}
			/> : null)}
	</div>
}

export default SettingsDeck
