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
import { describe, expect, test } from "vitest"
import {
	escapeHTML,
	escapeMarkdown,
	escapeMarkdownAndURI,
	makeMentionMarkdown,
	makeRoomMentionMarkdown,
} from "./markdown"

describe("escapeHTML", () => {
	test("escapes all HTML-special characters", () => {
		expect(escapeHTML(`&<>"'`)).toBe("&amp;&lt;&gt;&quot;&#039;")
	})

	test("leaves plain text unchanged", () => {
		expect(escapeHTML("hello world 123")).toBe("hello world 123")
	})

	test("escapes repeated occurrences", () => {
		expect(escapeHTML("a&b&<c")).toBe("a&amp;b&amp;&lt;c")
	})

	test("escapes ampersands before other characters to avoid double escaping", () => {
		expect(escapeHTML("&lt;")).toBe("&amp;lt;")
	})
})

describe("escapeMarkdown", () => {
	test("escapes all markdown-special characters with a backslash", () => {
		const expected = ["\\\\", "\\`", "\\*", "\\_", "\\[", "\\]", "\\(", "\\)"].join("")
		expect(escapeMarkdown("\\`*_[]()")).toBe(expected)
	})

	test("escapes angle brackets as HTML entities", () => {
		expect(escapeMarkdown("<b>")).toBe("&lt;b&gt;")
	})

	test("leaves plain text unchanged", () => {
		expect(escapeMarkdown("plain text")).toBe("plain text")
	})
})

describe("escapeMarkdownAndURI", () => {
	test("URI-encodes first, then markdown-escapes", () => {
		expect(escapeMarkdownAndURI("a b")).toBe("a%20b")
	})

	test("parentheses survive URI encoding and get markdown-escaped", () => {
		expect(escapeMarkdownAndURI("x(1)")).toBe("x\\(1\\)")
	})

	test("user IDs are fully encoded", () => {
		expect(escapeMarkdownAndURI("@user:example.org")).toBe("%40user%3Aexample.org")
	})
})

describe("makeMentionMarkdown", () => {
	test("produces a matrix.to mention link with a trailing space", () => {
		expect(makeMentionMarkdown("Alice", "@alice:example.org"))
			.toBe("[Alice](https://matrix.to/#/%40alice%3Aexample.org) ")
	})

	test("escapes markdown characters in the displayname", () => {
		expect(makeMentionMarkdown("Bob *star*", "@bob:example.org"))
			.toBe("[Bob \\*star\\*](https://matrix.to/#/%40bob%3Aexample.org) ")
	})

	test("replaces the first newline in the displayname with a space", () => {
		expect(makeMentionMarkdown("Multi\nLine\nName", "@m:example.org"))
			.toBe("[Multi Line\nName](https://matrix.to/#/%40m%3Aexample.org) ")
	})
})

describe("makeRoomMentionMarkdown", () => {
	test("mentions a room by ID with via parameters", () => {
		expect(makeRoomMentionMarkdown("Room", "!room:example.org", ["srv1", "srv2"]))
			.toBe("[Room](https://matrix.to/#/!room%3Aexample.org?via=srv1&via=srv2)")
	})

	test("omits the query for an alias even with via parameters", () => {
		expect(makeRoomMentionMarkdown("Room", "#room:example.org", ["srv1"]))
			.toBe("[Room](https://matrix.to/#/%23room%3Aexample.org)")
	})

	test("omits the query for an empty via list", () => {
		expect(makeRoomMentionMarkdown("Room", "!room:example.org", []))
			.toBe("[Room](https://matrix.to/#/!room%3Aexample.org)")
	})

	test("omits the query when via is undefined", () => {
		expect(makeRoomMentionMarkdown("Room", "!room:example.org"))
			.toBe("[Room](https://matrix.to/#/!room%3Aexample.org)")
	})

	test("escapes the room name and replaces its first newline", () => {
		expect(makeRoomMentionMarkdown("A [B]\nC", "#a:example.org"))
			.toBe("[A \\[B\\] C](https://matrix.to/#/%23a%3Aexample.org)")
	})
})
