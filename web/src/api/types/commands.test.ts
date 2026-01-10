import { readFileSync, readdirSync } from "fs"
import { expect, suite, test } from "vitest"
import { WrappedBotCommand, parseQuoted, stringToCommandArgs } from "./commands.ts"

interface QuoteTestData {
	name: string
	input: string
	output: [string, string, boolean]
}

const readJSON = (path: string) => JSON.parse(readFileSync(path, "utf-8"))

suite("parseQuoted", () => {
	const testdata: QuoteTestData[] = readJSON(`${__dirname}/commandtestdata/parse_quote.json`)
	for (const item of testdata) {
		test(item.name, () => {
			expect(parseQuoted(item.input)).toEqual(item.output)
		})
	}
})

interface CommandTest {
	name: string
	input: string
	error?: boolean
	output: Record<string, unknown> | null
}

interface CommandTestData {
	spec: WrappedBotCommand
	tests: CommandTest[]
}

suite("stringToCommandArgs", () => {
	const testdir = `${__dirname}/commandtestdata/commands`
	for (const file of readdirSync(testdir)) {
		const testdata: CommandTestData = readJSON(`${testdir}/${file}`)
		suite(file.replace(".json", ""), () => {
			for (const item of testdata.tests) {
				test(item.name, () => {
					expect(stringToCommandArgs(testdata.spec, item.input)).toEqual(item.output)
				})
			}
		})
	}
})
