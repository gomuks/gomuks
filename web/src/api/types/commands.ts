// gomuks - A Matrix client written in Go.
// Copyright (C) 2024 Tulir Asokan
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
import {
	ParsedMatrixURI,
	ensureString,
	ensureStringArray,
	isEventID,
	isRoomAlias,
	isRoomID,
	isServerName,
	isUserID,
	lessNoisyEncodeURIComponent,
	matrixToToMatrixURI,
	parseMatrixURI,
} from "@/util/validation.ts"
import {
	BotArgumentEventIDValue,
	BotArgumentRoomIDValue,
	BotArgumentValue,
	BotCommand,
	BotParameter,
	BotParameterPrimitiveType,
	BotParameterSchema,
	BotParameterSchemaType,
	ExtensibleTextContainer,
	SingleBotArgumentValue,
	UnknownEventContent,
	UserID,
} from "./mxtypes.ts"

export interface WrappedBotCommand extends BotCommand {
	parameters: BotParameter[]
	source: UserID | "@gomuks"
	fake?: boolean
}

function isValidBotParameterType(type: unknown): type is BotParameterPrimitiveType {
	switch (type) {
	case "string":
	case "integer":
	case "boolean":
	case "server_name":
	case "user_id":
	case "room_id":
	case "room_alias":
	case "event_id":
		return true
	default:
		return false
	}
}

function isValidBotParameterSchema(schema: BotParameterSchema, parent?: BotParameterSchemaType): boolean {
	if (typeof schema !== "object" || typeof schema?.schema_type !== "string") {
		return false
	}
	switch (schema.schema_type) {
	case "primitive":
		return isValidBotParameterType(schema.type)
	case "literal":
		return typeof schema.value === "string"
			|| typeof schema.value === "number"
			|| typeof schema.value === "boolean"
			|| (typeof schema.value === "object"
				&& (schema.value.type === "room_id" || schema.value.type === "event_id"))
	case "union":
		if (parent !== undefined && parent !== "array") {
			return false
		}
		return Array.isArray(schema.variants) &&
			schema.variants.every(x => isValidBotParameterSchema(x, schema.schema_type))
	case "array":
		if (parent !== undefined) {
			return false
		}
		return isValidBotParameterSchema(schema.items, schema.schema_type)
	default:
		return false
	}
}

function sanitizeParameters(args: BotParameter[] | undefined): BotParameter[] | null {
	if (args === undefined) {
		return []
	} else if (!Array.isArray(args)) {
		return null
	}
	for (const arg of args) {
		if (!isValidBotParameterSchema(arg.schema)) {
			return null
		}
	}
	return args
}

export function sanitizeCommand(stateKey: UserID, content?: UnknownEventContent): WrappedBotCommand | null {
	if (!content || typeof content.command !== "string") {
		return null
	}
	const parameters = sanitizeParameters(content.parameters)
	if (parameters === null) {
		return null
	}
	return {
		source: stateKey,
		fake: false,
		command: content.command,
		aliases: ensureStringArray(content["aliases"]),
		parameters,
		description: content.description,
	}
}

function quote(val: string): string
function quote(val: string | null): string | null
function quote(val: string | null): string | null {
	if (val === null) {
		return null
	}
	if (!val) {
		return `""`
	}
	val = val
		.replaceAll("\\", "\\\\")
		.replaceAll(`"`, `\\"`)
	if (val.includes(" ") || val.includes("\\") || val.startsWith(ARRAY_OPENER) || val.includes(ARRAY_CLOSER)) {
		val = `"${val}"`
	}
	return val
}

const ARRAY_OPENER = "<"
const ARRAY_CLOSER = ">"

export function parseQuoted(val: string): [string | null, string, boolean] {
	if (!val) {
		return [null, "", false]
	}
	if (!val.startsWith(`"`)) {
		const spaceIdx = val.indexOf(" ")
		if (spaceIdx === -1) {
			return [val, "", false]
		}
		return [val.slice(0, spaceIdx), val.slice(spaceIdx).trimStart(), false]
	}
	val = val.slice(1)
	const out = []
	while (true) {
		const quoteIdx = val.indexOf(`"`)
		const escapeIdx = val.slice(0, quoteIdx).indexOf(`\\`)
		if (escapeIdx >= 0) {
			out.push(val.slice(0, escapeIdx))
			out.push(val.charAt(escapeIdx+1))
			val = val.slice(escapeIdx+2)
		} else if (quoteIdx >= 0) {
			out.push(val.slice(0, quoteIdx))
			val = val.slice(quoteIdx + 1)
			break
		} else if (!out.length) {
			// Unterminated quote, no escape characters, val is the whole input
			return [val, "", true]
		} else {
			// Unterminated quote, but there were escape characters previously
			out.push(val)
			val = ""
			break
		}
	}
	return [out.join(""), val.trimStart(), true]
}

function getDefaultArgumentForSchema(schema: BotParameterSchema): BotArgumentValue {
	switch (schema?.schema_type) {
	case "literal":
		return schema.value
	case "primitive":
		switch (schema.type) {
		case "boolean":
			return false
		case "integer":
			return 0
		default:
			return ""
		}
	case "union":
		return getDefaultArgumentForSchema(schema.variants[0])
	case "array":
		return []
	}
}

function getDefaultArgument(schema: BotParameter): BotArgumentValue {
	if (schema.optional) {
		return null
	}
	return getDefaultArgumentForSchema(schema.schema)
}

export function getDefaultArguments(spec: WrappedBotCommand): Record<string, BotArgumentValue> {
	return Object.fromEntries(spec.parameters?.map(param => {
		return [param.key, param["fi.mau.default_value"] ?? getDefaultArgument(param)]
	}) ?? [])
}

function roomIDArgumentsEqual(
	a: BotArgumentRoomIDValue | BotArgumentEventIDValue | null | undefined,
	b: BotArgumentRoomIDValue | BotArgumentEventIDValue | null | undefined,
): boolean {
	if (!a || !b) {
		return a === b
	}
	return a.type === b.type
		&& a.id === b.id
		&& (a.type === "event_id" && b.type === "event_id" ? (a.event_id === b.event_id) : true)
		&& a.via?.length === b.via?.length
		&& Boolean(a.via?.every((server, idx) => server === b.via?.[idx]))
}

const markdownLinkRegex = /^\[.+]\(([^)]+)\)$/

const omitEmptyArray = <T>(arr: T[]): T[] | undefined => {
	if (arr.length === 0) {
		return undefined
	}
	return arr
}

function stringToIdentifierArgument(type: BotParameterPrimitiveType, val?: string): SingleBotArgumentValue | null {
	if (!val) {
		return null
	}
	if (val.startsWith("[") && val.includes("](") && val.endsWith(")")) {
		val = val.match(markdownLinkRegex)?.[1]
		if (!val) {
			return null
		}
	}
	let parsed: ParsedMatrixURI | undefined
	if (!val.startsWith("matrix:") && !val.startsWith("https://matrix.to/")) {
		parsed = {
			identifier: val,
			params: new URLSearchParams(),
		}
	} else {
		parsed = parseMatrixURI(matrixToToMatrixURI(val) ?? val)
	}
	if (!parsed) {
		return null
	}
	switch (type) {
	case "room_id":
		return isRoomID(parsed.identifier) ? {
			type: "room_id",
			id: parsed.identifier,
			via: omitEmptyArray(parsed.params.getAll("via")),
		} : null
	case "event_id":
		return isRoomID(parsed.identifier) && isEventID(parsed.eventID) ? {
			type: "event_id",
			id: parsed.identifier,
			event_id: parsed.eventID,
			via: omitEmptyArray(parsed.params.getAll("via")),
		} : null
	case "user_id":
		return isUserID(parsed.identifier) ? parsed.identifier : null
	case "room_alias":
		return isRoomAlias(parsed.identifier) ? parsed.identifier : null
	default:
		return null
	}
}

const truePrefixes = ["t", "T", "y", "Y", "1"]
const falsePrefixes = ["f", "F", "n", "N", "0"]

function parseBoolean(val?: string): boolean | null {
	const firstChar = val?.charAt(0) ?? ""
	if (truePrefixes.includes(firstChar)) {
		return true
	} else if (falsePrefixes.includes(firstChar)) {
		return false
	}
	return null
}

function parseIntOrNull(val?: string): number | null {
	const intVal = parseInt(val ?? "")
	return isNaN(intVal) ? null : intVal
}

function stringToPrimitiveArgument(type: BotParameterPrimitiveType, val?: string): SingleBotArgumentValue | null {
	switch (type) {
	case "string":
		return val === undefined ? null : val
	case "boolean":
		return parseBoolean(val)
	case "integer":
		return parseIntOrNull(val)
	case "user_id":
	case "room_id":
	case "room_alias":
	case "event_id":
		return stringToIdentifierArgument(type, val)
	case "server_name":
		return isServerName(val) ? val : null
	default:
		return null
	}
}

function literalEqual(expected: unknown, val: string): boolean {
	if (typeof expected === "string") {
		return expected === val
	} else if (typeof expected === "number") {
		return parseIntOrNull(val) === expected
	} else if (typeof expected === "boolean") {
		return parseBoolean(val) === expected
	} else if (
		typeof expected === "object" && expected !== null && "type" in expected
		&& (expected.type === "room_id" || expected.type === "event_id")
	) {
		const parsed = stringToIdentifierArgument(expected.type, val)
		return roomIDArgumentsEqual(
			parsed as BotArgumentRoomIDValue | BotArgumentEventIDValue,
			expected as BotArgumentRoomIDValue | BotArgumentEventIDValue,
		)
	} else {
		return false
	}
}

function stringToArgument(spec: BotParameterSchema, val: string | null): SingleBotArgumentValue | null {
	if (val === null) {
		return null
	}
	switch (spec.schema_type) {
	case "literal":
		return literalEqual(spec.value, val) ? spec.value : null
	case "primitive":
		return stringToPrimitiveArgument(spec.type, val)
	case "union":
		for (const variant of spec.variants) {
			const casted = stringToArgument(variant, val)
			if (casted !== null) {
				return casted
			}
		}
		return null
	case "array":
		return stringToArgument(spec.items, val)
	default:
		return null
	}
}

function singleArgumentToString(arg: BotArgumentValue): string | null {
	if (arg === null || arg === undefined || Array.isArray(arg)) {
		return null
	}
	if (typeof arg === "object") {
		let generatedURL = `matrix:roomid/${lessNoisyEncodeURIComponent(arg.id)}`
		if (arg.type === "event_id") {
			generatedURL += `/e/${lessNoisyEncodeURIComponent(arg.event_id)}`
		} else if (arg.type !== "room_id") {
			return null
		}
		if (Array.isArray(arg.via) && arg.via.length > 0) {
			generatedURL += "?" + new URLSearchParams(
				ensureStringArray(arg.via).map(server => ["via", server]),
			).toString()
		}
		return generatedURL
	} else if (typeof arg === "number" || typeof arg === "boolean" || typeof arg === "string") {
		return arg.toString()
	} else {
		return null
	}
}

function arrayArgumentToString(arg: BotArgumentValue, isLast: boolean): string | null {
	if (!Array.isArray(arg)) {
		return null
	}
	const parts = arg
		.map(part => quote(singleArgumentToString(part)))
		.filter(part => part !== null)
	if (isLast && parts.length > 0) {
		return parts.join(" ")
	}
	return ARRAY_OPENER + parts.join(" ") + ARRAY_CLOSER
}

function validateCommandPrefix(spec: WrappedBotCommand, input: string): string | null {
	const prefix = "/" + spec.command
	if (!input.startsWith(prefix)) {
		return null
	} else if (input === prefix) {
		return prefix
	}
	input = input.slice(prefix.length)
	if (input.startsWith(" ")) {
		return prefix + " "
	} else if (input === spec.source) {
		return prefix + spec.source
	} else if (input.startsWith(spec.source + " ")) {
		return prefix + spec.source + " "
	}
	return null
}

function schemaAllowsPrimitive(schema: BotParameterSchema, primitive: BotParameterPrimitiveType): boolean {
	switch (schema.schema_type) {
	case "primitive":
		return schema.type === primitive
	case "union":
		return schema.variants.some(variant => schemaAllowsPrimitive(variant, primitive))
	case "array":
		return schemaAllowsPrimitive(schema.items, primitive)
	default:
		return false
	}
}

export function stringToCommandArgs(
	spec: WrappedBotCommand, input: string,
): Record<string, BotArgumentValue> | null {
	const prefix = validateCommandPrefix(spec, input)
	if (!prefix) {
		return null
	}
	input = input.slice(prefix.length)
	const args: Record<string, BotArgumentValue> = {}
	const processParam = (param: BotParameter, isLast: boolean, isNamed: boolean) => {
		let nextVal: string | null
		let wasQuoted: boolean
		const origInput = input
		if (param.schema.schema_type === "array") {
			const hasOpener = input.startsWith(ARRAY_OPENER)
			let arrayClosed = false
			if (hasOpener) {
				input = input.slice(1)
				if (input.startsWith(ARRAY_CLOSER)) {
					input = input.slice(1).trimStart()
					arrayClosed = true
				}
			}
			const collector = []
			while (input.length > 0 && !arrayClosed) {
				[nextVal, input, wasQuoted] = parseQuoted(input)
				if (!wasQuoted && hasOpener && nextVal?.endsWith(ARRAY_CLOSER)) {
					// The value wasn't quoted and has the array delimiter at the end, close the array
					nextVal = nextVal.slice(0, -1)
					arrayClosed = true
				} else if (hasOpener && input.startsWith(ARRAY_CLOSER)) {
					// The value was quoted or there was a space, and the next character is the
					// array delimiter, close the array
					input = input.slice(1).trimStart()
					arrayClosed = true
				} else if (!hasOpener && !isLast) {
					// For array arguments in the middle without the <> delimiters, stop after the first item
					arrayClosed = true
				}
				const arg = stringToArgument(param.schema, nextVal)
				if (arg !== null) {
					collector.push(arg)
				}
			}
			args[param.key] = collector.length ? collector : param["fi.mau.default_value"] ?? getDefaultArgument(param)
		} else {
			[nextVal, input, wasQuoted] = parseQuoted(input)
			if (isLast && !wasQuoted && input) {
				// If the last argument is not quoted and not variadic, just treat the rest of the string
				// as the argument without escapes (arguments with escapes should be quoted).
				nextVal += " " + input
				input = ""
			}
			// Special case for named boolean parameters: if no value is given, treat it as true
			if (!nextVal && !wasQuoted && isNamed && schemaAllowsPrimitive(param.schema, "boolean")) {
				args[param.key] = true
				return
			}
			const parsedVal = stringToArgument(param.schema, nextVal)
			args[param.key] = parsedVal ?? param["fi.mau.default_value"] ?? getDefaultArgument(param)
			// For optional parameters that fail to parse, restore the input and try passing it as the next parameter
			if (parsedVal === null && param.optional && !isLast && !isNamed) {
				input = origInput.trimStart()
			}
		}
	}
	let i = 0
	const skipParams: boolean[] = []
	for (const param of spec.parameters) {
		while (input.startsWith("--")) {
			const equalsIdx = input.indexOf("=")
			const spaceIdx = input.indexOf(" ")
			let nameEndIdx: number
			if (equalsIdx === -1 && spaceIdx === -1) {
				nameEndIdx = input.length
			} else if (equalsIdx === -1) {
				nameEndIdx = spaceIdx
			} else if (spaceIdx === -1) {
				nameEndIdx = equalsIdx
			} else {
				nameEndIdx = Math.min(equalsIdx, spaceIdx)
			}
			const overrideName = input.slice(2, nameEndIdx)
			const overrideIdx = spec.parameters.findIndex(param => param.key === overrideName)
			if (overrideIdx >= 0) {
				input = input.slice(nameEndIdx)
				// Trim the equals sign, but leave spaces alone to let parseQuoted treat it as empty input
				if (input.startsWith("=")) {
					input = input.slice(1)
				}
				const overrideParam = spec.parameters[overrideIdx]
				skipParams[overrideIdx] = true
				processParam(overrideParam, false, true)
			} else {
				break
			}
		}
		if (skipParams[i]) {
			i++
			continue
		}
		processParam(param, ++i === spec.parameters.length, false)
	}
	return args
}

export function commandArgsToString(
	spec: WrappedBotCommand,
	argValues: Record<string, BotArgumentValue>,
): string {
	// TODO if spec.source is removed, make sure to sync with the caller in Autocompleter.tsx
	const parts = ["/" + spec.command + spec.source]
	let i = 0
	for (const param of spec.parameters) {
		const isLast = ++i === (spec.parameters!.length ?? 0)
		const val = argValues[param.key]
		let stringified: string | null
		if (param.schema.schema_type === "array") {
			stringified = arrayArgumentToString(val, isLast)
				?? arrayArgumentToString(param["fi.mau.default_value"] ?? getDefaultArgument(param), isLast)
		} else {
			stringified = quote(
				singleArgumentToString(val)
				?? singleArgumentToString(param["fi.mau.default_value"]
					?? getDefaultArgument(param)),
			)
		}
		if (stringified) {
			parts.push(stringified)
		}
	}
	return parts.join(" ")
}

export function unpackExtensibleText(text?: ExtensibleTextContainer): string {
	return ensureString(text?.["m.text"]?.find(item => !item.mimetype || item.mimetype === "text/plain")?.body)
}
