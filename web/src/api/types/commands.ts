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
import { ensureString, ensureStringArray } from "@/util/validation.ts"
import { UnknownEventContent } from "./hitypes.ts"
import {
	BotArgument,
	BotArgumentValue,
	BotCommand,
	ExtensibleTextContainer,
	SingleBotArgumentValue,
	UserID,
} from "./mxtypes.ts"

export interface WrappedBotCommand extends BotCommand {
	source: UserID | "@gomuks"
	sigil: string
}

function sanitizeArguments(args: BotArgument[] | undefined): BotArgument[] {
	if (!Array.isArray(args)) {
		return []
	}
	let index = 0
	for (const arg of args) {
		arg.type = typeof arg.type !== "string" ? "string" : arg.type
		if (typeof arg.variadic !== "boolean" || index !== args.length - 1) {
			delete arg.variadic
		}
		if (arg.type === "enum") {
			arg.enum = ensureStringArray(arg.enum)
			if (arg.enum.length === 0) {
				arg.enum = [""]
			}
		}
		index++
	}
	return args
}

export function mapCommandContent(stateKey: UserID, content?: UnknownEventContent): WrappedBotCommand[] {
	if (!content || !Array.isArray(content.commands) || typeof content.sigil !== "string") {
		return []
	}
	return content.commands.map((cmd): WrappedBotCommand | null => {
		if (typeof cmd.syntax !== "string") {
			return null
		}
		return {
			source: stateKey,
			sigil: content.sigil,
			syntax: cmd.syntax,
			arguments: sanitizeArguments(cmd.arguments),
			description: cmd.description,
			"fi.mau.aliases": ensureStringArray(cmd["fi.mau.aliases"]),
		}
	}).filter(x => x !== null)
}

const argNameRegex = /{(.+?)}/g

export function findArgumentNames(syntax: string): string[] {
	return syntax
		.matchAll(argNameRegex)
		.map(item => item[1])
		.toArray()
}

function argToString(arg: BotArgumentValue): string | null {
	switch (typeof arg) {
	case "boolean":
		return arg ? "true" : "false"
	case "number":
		return arg.toString()
	case "string":
		return quote(arg)
	case "object":
		if (arg === null) {
			return null
		} else if (Array.isArray(arg)) {
			return arg.map(argToString).join(" ")
		} else if (typeof arg.id === "string") {
			return quote(arg.id)
		}
	}
	return null
}

function quote(val: string): string {
	if (!val) {
		return `""`
	}
	val = val
		.replaceAll("\\", "\\\\")
		.replaceAll(`"`, `\\"`)
	if (val.includes(" ") || val.includes("\\")) {
		val = `"${val}"`
	}
	return val
}

function parseQuoted(val: string): [string, string] {
	if (!val.startsWith(`"`)) {
		const spaceIdx = val.indexOf(" ")
		if (spaceIdx === -1) {
			return [val, ""]
		}
		return [val.slice(0, spaceIdx), val.slice(spaceIdx)]
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
			return [val, ""]
		} else {
			out.push(val)
			val = ""
			break
		}
	}
	return [out.join(""), val]
}

export function getDefaultArguments(
	spec: WrappedBotCommand, argNames?: string[],
): Record<string, BotArgumentValue> {
	if (!argNames) {
		argNames = findArgumentNames(spec.syntax)
	}
	return Object.fromEntries(spec.arguments?.map((param, index) => {
		let defVal: BotArgumentValue | undefined = param["fi.mau.default_value"]
		if (defVal == undefined) {
			if (param.type === "boolean") {
				defVal = false
			} else if (param.type === "integer") {
				defVal = 0
			} else if (param.type === "enum") {
				defVal = param.enum[0]
			} else {
				defVal = ""
			}
			if (param.variadic) {
				defVal = [defVal]
			}
		}
		return [argNames[index], defVal]
	}) ?? [])
}

function castArgument(spec: BotArgument, val: string): SingleBotArgumentValue {
	if (spec.type === "boolean") {
		return val === "true"
	} else if (spec.type === "integer") {
		const intVal = parseInt(val)
		return isNaN(intVal) ? 0 : intVal
	} else if (spec.type === "enum") {
		if (!spec.enum.includes(val)) {
			return spec.enum[0]
		}
	}
	return val
}

export function parseArgumentValues(
	spec: WrappedBotCommand, input: string,
): Record<string, BotArgumentValue> | null {
	if (!input.startsWith("/")) {
		return null
	}
	input = input.slice(1)
	const args = getDefaultArguments(spec)
	let i = 0
	for (const part of spec.syntax.split(argNameRegex)) {
		if (i === 0) {
			if (!input.startsWith(part)) {
				return null
			}
			input = input.slice(part.length)
		} else if (i % 2 === 0) {
			if (!input.startsWith(part)) {
				return args
			}
			input = input.slice(part.length)
		} else {
			const origInput = input
			let argVal: string
			[argVal, input] = parseQuoted(input)
			const argIdx = Math.floor(i/2)
			const argSpec = spec.arguments![argIdx]
			const isLastArg = argIdx === spec.arguments!.length - 1
			if (argSpec.variadic && isLastArg) {
				args[part] = [castArgument(argSpec, argVal)]
				while (input.startsWith(" ")) {
					[argVal, input] = parseQuoted(input.trimStart())
					args[part].push(castArgument(argSpec, argVal))
				}
			} else if (isLastArg && !origInput.startsWith(`"`)) {
				// If the last argument is not quoted and not variadic, just treat the rest of the string
				// as the argument without escapes (arguments with escapes should be quoted).
				args[part] = castArgument(argSpec, origInput)
			} else {
				args[part] = castArgument(argSpec, argVal)
			}
		}
		i++
	}
	return args
}

export function replaceArgumentValues(
	syntax: string,
	argValues: Record<string, BotArgumentValue>,
): string {
	const parts = syntax.split(argNameRegex)
	for (let i = 1; i < parts.length; i += 2) {
		const key = parts[i]
		const val = argToString(argValues[key])
		if (val !== null) {
			// For extremely weird syntaxes where the parameter isn't separated by a space,
			// we need to always quote it to find where it ends.
			const needsQuote = parts[i+1] && !parts[i+1].startsWith(" ")
			if (needsQuote && !val.startsWith(`"`) && !Array.isArray(argValues[key])) {
				parts[i] = `"${val}"`
			} else {
				parts[i] = val
			}
		}
	}
	return parts.join("")
}

export function unpackExtensibleText(text?: ExtensibleTextContainer): string {
	return ensureString(text?.["m.text"]?.find(item => !item.mimetype || item.mimetype === "text/plain")?.body)
}
