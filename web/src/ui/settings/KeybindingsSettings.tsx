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
import { useEffect, useState } from "react"
import { keyToString } from "../keybindings.ts"
import {
	ACTION_LABELS,
	ActionName,
	BINDABLE_ACTIONS,
	DEFAULT_KEYBINDINGS,
	buildEffectiveKeymap,
	clearKeybindOverrides,
	loadKeybindOverrides,
	saveKeybindOverrides,
} from "../keyconfig.ts"

function isModifierOnly(evt: KeyboardEvent): boolean {
	return evt.key === "Control" || evt.key === "Shift" || evt.key === "Alt" || evt.key === "Meta"
}

const KeybindingsSettings = () => {
	const [overrides, setOverrides] = useState<Record<string, string>>(loadKeybindOverrides)
	const [recording, setRecording] = useState<ActionName | null>(null)
	const [error, setError] = useState<string | null>(null)

	const effective = buildEffectiveKeymap(overrides)

	useEffect(() => {
		if (!recording) {
			return
		}
		const onKeyDown = (evt: KeyboardEvent) => {
			if (isModifierOnly(evt)) {
				return
			}
			evt.preventDefault()
			evt.stopPropagation()
			const key = keyToString(evt)
			const next = { ...loadKeybindOverrides(), [recording]: key }
			try {
				saveKeybindOverrides(next)
				setOverrides(next)
				setError(null)
				setRecording(null)
			} catch (err) {
				setError(err instanceof Error ? err.message : String(err))
				setRecording(null)
			}
		}
		window.addEventListener("keydown", onKeyDown, true)
		return () => window.removeEventListener("keydown", onKeyDown, true)
	}, [recording])

	const resetDefaults = () => {
		clearKeybindOverrides()
		setOverrides({})
		setError(null)
		setRecording(null)
	}

	return <div className="keybindings-settings">
		<h3>Keyboard shortcuts</h3>
		<p>
			Click a shortcut, then press the new key. Enter and Tab stay reserved for the composer.
			Reply shortcuts still require the “Use Ctrl+Arrow to reply” preference.
		</p>
		{error && <div className="keybind-error" role="alert">{error}</div>}
		<div className="keybind-table">
			<div className="name">Action</div>
			<div className="name">Shortcut</div>
			{BINDABLE_ACTIONS.map(action => [
				<div className="name" key={`${action}-label`}>{ACTION_LABELS[action]}</div>,
				<button
					key={`${action}-key`}
					className={recording === action ? "recording" : ""}
					onClick={() => {
						setError(null)
						setRecording(action)
					}}
				>
					{recording === action ? "Press a key…" : effective[action]}
				</button>,
			])}
		</div>
		<div className="keybind-actions">
			<button onClick={resetDefaults} disabled={Object.keys(overrides).length === 0}>
				Reset to defaults
			</button>
			<code>{DEFAULT_KEYBINDINGS.close_panel} closes panels first, then the room.</code>
		</div>
	</div>
}

export default KeybindingsSettings
