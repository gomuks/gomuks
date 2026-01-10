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
import React, { JSX } from "react"
import { RoomStateStore } from "@/api/statestore"
import {
	BotArgumentValue,
	BotParameter,
	SingleBotArgumentValue,
	commandArgsToString,
	unpackExtensibleText,
} from "@/api/types"
import { ComposerState } from "./MessageComposer.tsx"

interface CommandArgumentProps {
	index: number
	spec: BotParameter
	value: BotArgumentValue
	setValue: (value: BotArgumentValue) => void
}

function renderArgumentContent(
	spec: BotParameter,
	value: BotArgumentValue,
	setValue: (value: SingleBotArgumentValue) => void, description: string,
	contentID: string,
	autoFocus: boolean,
	onKeyDown: (evt: React.KeyboardEvent) => void,
	key?: number,
): JSX.Element {
	if (spec.schema.schema_type === "primitive" && spec.schema.type === "boolean") {
		return <input
			key={key}
			id={contentID}
			autoFocus={autoFocus}
			type="checkbox"
			checked={(value ?? false) as boolean}
			onChange={evt => setValue(evt.target.checked)}
			onKeyDown={onKeyDown}
		/>
	} else if (spec.schema.schema_type === "primitive" && spec.schema.type === "integer") {
		return <input
			key={key}
			id={contentID}
			autoFocus={autoFocus}
			type="number"
			value={(value ?? 0) as number}
			onChange={evt => {
				const val = parseInt(evt.target.value)
				setValue(isNaN(val) ? 0 : val)
			}}
			onKeyDown={onKeyDown}
			placeholder={description}
		/>
	}/* else if (spec.type === "enum") {
		return <select
			key={key}
			id={contentID}
			autoFocus={autoFocus}
			value={value as string}
			onChange={evt => setValue(evt.target.value)}
			onKeyDown={onKeyDown}
		>
			{spec.enum.map(option => <option key={option} value={option}>{option}</option>)}
		</select>
	} */ else {
		return <input
			key={key}
			id={contentID}
			autoFocus={autoFocus}
			type="text"
			value={(value ?? "") as string}
			onChange={evt => setValue(evt.target.value)}
			onKeyDown={onKeyDown}
			placeholder={description}
		/>
	}
}

const CommandArgument = ({ index, spec, value, setValue }: CommandArgumentProps) => {
	const description = unpackExtensibleText(spec.description) || spec.key
	const contentID = `cmd-arg-${index}`
	const onKeyDown = (evt: React.KeyboardEvent) => {
		if (evt.key === "Enter") {
			evt.preventDefault()
			const next = document.getElementById(`cmd-arg-${index + 1}`)
			if (next) {
				next.focus()
			} else {
				document.getElementById("message-composer")?.focus()
			}
		}
	}
	let content: JSX.Element
	if (spec.schema.schema_type === "array") {
		const valueSetter = (itemIdx: number) => (itemVal: SingleBotArgumentValue) => {
			const newArr = [...value as SingleBotArgumentValue[]]
			newArr[itemIdx] = itemVal
			setValue(newArr)
		}
		content = <div className="variadic-items">
			{(value as SingleBotArgumentValue[]).map((item, itemIdx) =>
				renderArgumentContent(
					spec, item, valueSetter(itemIdx), description, contentID, false, onKeyDown, itemIdx,
				))}
		</div>
	} else {
		content = renderArgumentContent(spec, value, setValue, description, contentID, false, onKeyDown)
	}
	return <>
		<label htmlFor={contentID} title={description}>{spec.key}</label>
		{content}
	</>
}

export interface CommandInputProps {
	room: RoomStateStore
	state: ComposerState
	setState: (state: Partial<ComposerState>) => void
}

const CommandInput = ({ state, setState }: CommandInputProps) => {
	const cmd = state.command!
	return <div className="command-arguments">
		{cmd.spec.parameters?.map((spec, index) => {
			return <CommandArgument
				key={index}
				index={index}
				spec={spec}
				value={cmd.inputArgs[spec.key]}
				setValue={val => {
					const inputArgs = {
						...cmd.inputArgs,
						[spec.key]: val,
					}
					setState({
						text: commandArgsToString(cmd.spec, inputArgs),
						command: {
							...cmd,
							inputArgs,
						},
					})
				}}
			/>
		})}
		{!cmd.spec.parameters?.length ? "Selected /" + cmd.spec.command : null}
	</div>
}

export default CommandInput
