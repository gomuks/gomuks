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
import { describe, expect, test } from "vitest"
import {
	ensureArray,
	ensureNumber,
	ensureString,
	ensureStringArray,
	ensureTypedArray,
	getDisplayname,
	getLegacyMSC1767Text,
	getLocalpart,
	getRelatesTo,
	getServerName,
	getThreadRoot,
	isEventID,
	isMXC,
	isRoomAlias,
	isRoomID,
	isServerName,
	isString,
	isThread,
	isUserID,
	lessNoisyEncodeURIComponent,
	matrixToToMatrixURI,
	onlyIfString,
	parseMatrixURI,
	parseMXC,
	validated,
} from "./validation"

describe("validated", () => {
	test("returns false for undefined", () => {
		expect(validated(undefined, () => true)).toBe(false)
	})

	test("returns true when validator passes", () => {
		expect(validated(5, v => v > 3)).toBe(true)
	})

	test("returns false when validator fails", () => {
		expect(validated(2, v => v > 3)).toBe(false)
	})
})

describe("isServerName", () => {
	test("accepts plain hostnames", () => {
		expect(isServerName("example.com")).toBe(true)
		expect(isServerName("example.com:8448")).toBe(true)
		expect(isServerName("127.0.0.1")).toBe(true)
	})

	test("rejects non-strings", () => {
		expect(isServerName(5)).toBe(false)
		expect(isServerName(null)).toBe(false)
		expect(isServerName(undefined)).toBe(false)
	})

	test("rejects strings with invalid characters", () => {
		expect(isServerName("bad server")).toBe(false)
		expect(isServerName("")).toBe(false)
		expect(isServerName("server/path")).toBe(false)
	})
})

describe("isEventID", () => {
	test("accepts $-prefixed strings without requiring a server", () => {
		expect(isEventID("$abc123")).toBe(true)
		expect(isEventID("$")).toBe(true)
		expect(isEventID("$abc:example.com")).toBe(true)
	})

	test("rejects wrong sigil and non-strings", () => {
		expect(isEventID("abc123")).toBe(false)
		expect(isEventID("@abc")).toBe(false)
		expect(isEventID(5)).toBe(false)
		expect(isEventID(null)).toBe(false)
	})
})

describe("isUserID", () => {
	test("accepts valid user IDs", () => {
		expect(isUserID("@user:example.com")).toBe(true)
		expect(isUserID("@user:example.com:8448")).toBe(true)
	})

	test("rejects user IDs without a server part", () => {
		expect(isUserID("@user")).toBe(false)
		expect(isUserID("@")).toBe(false)
	})

	test("accepts empty localpart as long as a colon follows the sigil", () => {
		expect(isUserID("@:example.com")).toBe(true)
	})

	test("rejects invalid server characters", () => {
		expect(isUserID("@user:bad server")).toBe(false)
		expect(isUserID("@user:server/path")).toBe(false)
	})

	test("rejects wrong sigil and non-strings", () => {
		expect(isUserID("user:example.com")).toBe(false)
		expect(isUserID("!user:example.com")).toBe(false)
		expect(isUserID(42)).toBe(false)
	})
})

describe("isRoomID", () => {
	test("accepts !-prefixed strings without requiring a server", () => {
		expect(isRoomID("!abc123")).toBe(true)
		expect(isRoomID("!abc:example.com")).toBe(true)
		expect(isRoomID("!")).toBe(true)
	})

	test("rejects wrong sigil and non-strings", () => {
		expect(isRoomID("abc")).toBe(false)
		expect(isRoomID("#abc:example.com")).toBe(false)
		expect(isRoomID(undefined)).toBe(false)
	})
})

describe("isRoomAlias", () => {
	test("accepts valid aliases", () => {
		expect(isRoomAlias("#room:example.com")).toBe(true)
	})

	test("rejects aliases without a server part", () => {
		expect(isRoomAlias("#room")).toBe(false)
		expect(isRoomAlias("#")).toBe(false)
	})

	test("rejects invalid server characters", () => {
		expect(isRoomAlias("#room:bad server")).toBe(false)
	})

	test("accepts empty localpart as long as a colon follows the sigil", () => {
		expect(isRoomAlias("#:example.com")).toBe(true)
	})

	test("rejects wrong sigil and non-strings", () => {
		expect(isRoomAlias("!room:example.com")).toBe(false)
		expect(isRoomAlias({})).toBe(false)
	})
})

describe("isMXC", () => {
	test("accepts valid mxc URIs", () => {
		expect(isMXC("mxc://example.com/abc123")).toBe(true)
		expect(isMXC("mxc://example.com/Ab-_9")).toBe(true)
	})

	test("rejects invalid mxc URIs", () => {
		expect(isMXC("mxc://example.com/")).toBe(false)
		expect(isMXC("mxc://example.com")).toBe(false)
		expect(isMXC("mxc:///abc")).toBe(false)
		expect(isMXC("mxc://example.com/ab/cd")).toBe(false)
		expect(isMXC("https://example.com/abc")).toBe(false)
		expect(isMXC("")).toBe(false)
	})

	test("rejects non-strings", () => {
		expect(isMXC(7)).toBe(false)
		expect(isMXC(null)).toBe(false)
	})
})

describe("getRelatesTo", () => {
	test("returns undefined for null/undefined event", () => {
		expect(getRelatesTo(undefined)).toBeUndefined()
		expect(getRelatesTo(null)).toBeUndefined()
	})

	test("returns m.relates_to from content", () => {
		const rel = { rel_type: "m.annotation", event_id: "$abc" }
		const evt = { content: { "m.relates_to": rel } } as never
		expect(getRelatesTo(evt)).toBe(rel)
	})

	test("prefers orig_content over content", () => {
		const origRel = { rel_type: "m.thread", event_id: "$orig" }
		const rel = { rel_type: "m.annotation", event_id: "$new" }
		const evt = { content: { "m.relates_to": rel }, orig_content: { "m.relates_to": origRel } } as never
		expect(getRelatesTo(evt)).toBe(origRel)
	})

	test("returns undefined when no m.relates_to present", () => {
		const evt = { content: {} } as never
		expect(getRelatesTo(evt)).toBeUndefined()
	})
})

describe("getThreadRoot / isThread", () => {
	test("returns event_id for valid thread relation", () => {
		expect(getThreadRoot({ rel_type: "m.thread", event_id: "$abc" })).toBe("$abc")
		expect(isThread({ rel_type: "m.thread", event_id: "$abc" })).toBe(true)
	})

	test("returns undefined for non-thread relation types", () => {
		expect(getThreadRoot({ rel_type: "m.annotation", event_id: "$abc" })).toBeUndefined()
		expect(isThread({ rel_type: "m.annotation", event_id: "$abc" })).toBe(false)
	})

	test("returns undefined for invalid event_id", () => {
		expect(getThreadRoot({ rel_type: "m.thread", event_id: "abc" })).toBeUndefined()
		expect(isThread({ rel_type: "m.thread", event_id: "abc" })).toBe(false)
	})

	test("returns undefined for undefined relation", () => {
		expect(getThreadRoot(undefined)).toBeUndefined()
		expect(isThread(undefined)).toBe(false)
	})
})

describe("matrixToToMatrixURI", () => {
	test("converts room alias URL", () => {
		expect(matrixToToMatrixURI("https://matrix.to/#/%23room:example.com")).toBe("matrix:r/room:example.com")
	})

	test("converts user URL", () => {
		expect(matrixToToMatrixURI("https://matrix.to/#/@user:example.com")).toBe("matrix:u/user:example.com")
	})

	test("converts room ID URL", () => {
		expect(matrixToToMatrixURI("https://matrix.to/#/!abc:example.com")).toBe("matrix:roomid/abc:example.com")
	})

	test("converts room ID + event URL", () => {
		expect(matrixToToMatrixURI("https://matrix.to/#/!abc:example.com/$event123"))
			.toBe("matrix:roomid/abc:example.com/e/event123")
	})

	test("preserves query parameters", () => {
		expect(matrixToToMatrixURI("https://matrix.to/#/@user:example.com?via=example.org"))
			.toBe("matrix:u/user:example.com?via=example.org")
	})

	test("re-encodes non-colon percent sequences after decoding", () => {
		// decodeURIComponent gives @user@host:..., then encodeURIComponent re-encodes the @
		expect(matrixToToMatrixURI("https://matrix.to/#/@user%40host:example.com"))
			.toBe("matrix:u/user%40host:example.com")
	})

	test("returns null for non-matrix.to URLs", () => {
		expect(matrixToToMatrixURI("https://example.com/#/@user:example.com")).toBeNull()
	})

	test("returns null for unrecognized sigils", () => {
		expect(matrixToToMatrixURI("https://matrix.to/#/foo")).toBeNull()
		expect(matrixToToMatrixURI("https://matrix.to/#/")).toBeNull()
	})

	test("ignores second path part for non-event sigils", () => {
		expect(matrixToToMatrixURI("https://matrix.to/#/!abc:example.com/notanevent"))
			.toBe("matrix:roomid/abc:example.com")
	})
})

describe("parseMatrixURI", () => {
	test("parses user URIs", () => {
		const parsed = parseMatrixURI("matrix:u/user:example.com")
		expect(parsed?.identifier).toBe("@user:example.com")
		expect(parsed?.eventID).toBeUndefined()
	})

	test("parses room alias URIs", () => {
		expect(parseMatrixURI("matrix:r/room:example.com")?.identifier).toBe("#room:example.com")
	})

	test("parses room ID URIs", () => {
		expect(parseMatrixURI("matrix:roomid/abc:example.com")?.identifier).toBe("!abc:example.com")
	})

	test("parses room ID URIs with event", () => {
		const parsed = parseMatrixURI("matrix:roomid/abc:example.com/e/event123")
		expect(parsed?.identifier).toBe("!abc:example.com")
		expect(parsed?.eventID).toBe("$event123")
	})

	test("roomid without e subtype has no eventID", () => {
		expect(parseMatrixURI("matrix:roomid/abc:example.com/x/event123")?.eventID).toBeUndefined()
	})

	test("exposes query params", () => {
		const parsed = parseMatrixURI("matrix:u/user:example.com?via=example.org")
		expect(parsed?.params.get("via")).toBe("example.org")
	})

	test("decodes percent-encoded identifiers", () => {
		expect(parseMatrixURI("matrix:u/user%40host:example.com")?.identifier).toBe("@user@host:example.com")
	})

	test("returns undefined for non-strings", () => {
		expect(parseMatrixURI(5)).toBeUndefined()
		expect(parseMatrixURI(null)).toBeUndefined()
	})

	test("returns undefined for invalid URLs", () => {
		expect(parseMatrixURI("not a url at all")).toBeUndefined()
	})

	test("returns undefined for non-matrix protocols", () => {
		expect(parseMatrixURI("https://example.com/u/user")).toBeUndefined()
	})

	test("returns undefined for unknown types", () => {
		expect(parseMatrixURI("matrix:x/foo")).toBeUndefined()
	})
})

describe("lessNoisyEncodeURIComponent", () => {
	test("keeps the first colon unencoded", () => {
		expect(lessNoisyEncodeURIComponent("user:example.com")).toBe("user:example.com")
	})

	test("encodes other characters", () => {
		expect(lessNoisyEncodeURIComponent("a b")).toBe("a%20b")
		expect(lessNoisyEncodeURIComponent("a#b")).toBe("a%23b")
	})

	test("only replaces the first %3A", () => {
		expect(lessNoisyEncodeURIComponent("a:b:c")).toBe("a:b%3Ac")
	})
})

describe("getLocalpart / getServerName", () => {
	test("extracts localpart", () => {
		expect(getLocalpart("@user:example.com")).toBe("user")
	})

	test("handles user IDs without server part", () => {
		expect(getLocalpart("@user")).toBe("user")
	})

	test("extracts server name", () => {
		expect(getServerName("@user:example.com")).toBe("example.com")
		expect(getServerName("@user:example.com:8448")).toBe("example.com:8448")
	})
})

describe("getDisplayname", () => {
	test("uses trimmed profile displayname", () => {
		expect(getDisplayname("@user:example.com", { displayname: "  Alice  " })).toBe("Alice")
	})

	test("falls back to localpart for whitespace-only displayname", () => {
		expect(getDisplayname("@user:example.com", { displayname: "   " })).toBe("user")
	})

	test("falls back to localpart without profile", () => {
		expect(getDisplayname("@user:example.com")).toBe("user")
		expect(getDisplayname("@user:example.com", null)).toBe("user")
	})

	test("displayname with empty localpart still shows the displayname", () => {
		expect(getDisplayname("@:example.com", { displayname: "Display" })).toBe("Display")
	})

	test("falls back to full user ID when localpart is empty", () => {
		expect(getDisplayname("@:example.com")).toBe("@:example.com")
	})

	test("ignores non-string displayname", () => {
		expect(getDisplayname("@user:example.com", { displayname: 5 } as never)).toBe("user")
	})
})

describe("parseMXC", () => {
	test("parses valid mxc URIs", () => {
		expect(parseMXC("mxc://example.com/abc123")).toEqual(["example.com", "abc123"])
	})

	test("returns empty array for invalid strings", () => {
		expect(parseMXC("https://example.com/abc")).toEqual([])
		expect(parseMXC("mxc://example.com/")).toEqual([])
		expect(parseMXC("")).toEqual([])
	})

	test("returns empty array for non-strings", () => {
		expect(parseMXC(undefined)).toEqual([])
		expect(parseMXC(9)).toEqual([])
	})
})

describe("ensureNumber", () => {
	test("passes through numbers", () => {
		expect(ensureNumber(5)).toBe(5)
		expect(ensureNumber(0)).toBe(0)
		expect(ensureNumber(-1.5)).toBe(-1.5)
	})

	test("returns 0 for NaN", () => {
		expect(ensureNumber(NaN)).toBe(0)
	})

	test("returns 0 for non-numbers", () => {
		expect(ensureNumber("5")).toBe(0)
		expect(ensureNumber(null)).toBe(0)
		expect(ensureNumber(undefined)).toBe(0)
	})
})

describe("ensureString", () => {
	test("passes through strings", () => {
		expect(ensureString("hello")).toBe("hello")
		expect(ensureString("")).toBe("")
	})

	test("returns empty string for non-strings", () => {
		expect(ensureString(5)).toBe("")
		expect(ensureString(null)).toBe("")
		expect(ensureString(undefined)).toBe("")
		expect(ensureString({})).toBe("")
	})
})

describe("ensureArray", () => {
	test("passes through arrays", () => {
		const arr = [1, 2]
		expect(ensureArray(arr)).toBe(arr)
	})

	test("returns empty array for non-arrays", () => {
		expect(ensureArray("nope")).toEqual([])
		expect(ensureArray(null)).toEqual([])
		expect(ensureArray({ length: 2 })).toEqual([])
	})
})

describe("isString / onlyIfString", () => {
	test("isString detects strings", () => {
		expect(isString("a")).toBe(true)
		expect(isString("")).toBe(true)
		expect(isString(5)).toBe(false)
		expect(isString(null)).toBe(false)
	})

	test("onlyIfString returns string or undefined", () => {
		expect(onlyIfString("a")).toBe("a")
		expect(onlyIfString(5)).toBeUndefined()
		expect(onlyIfString(undefined)).toBeUndefined()
	})
})

describe("ensureStringArray / ensureTypedArray", () => {
	test("returns same array reference when all items match", () => {
		const arr = ["a", "b"]
		expect(ensureStringArray(arr)).toBe(arr)
	})

	test("filters out non-matching items", () => {
		expect(ensureStringArray(["a", 5, "b", null])).toEqual(["a", "b"])
	})

	test("returns empty array for non-arrays", () => {
		expect(ensureStringArray("nope")).toEqual([])
		expect(ensureStringArray(undefined)).toEqual([])
	})

	test("ensureTypedArray works with custom type guards", () => {
		const isNum = (v: unknown): v is number => typeof v === "number"
		const arr = [1, 2]
		expect(ensureTypedArray(arr, isNum)).toBe(arr)
		expect(ensureTypedArray([1, "a", 2], isNum)).toEqual([1, 2])
		expect(ensureTypedArray(null, isNum)).toEqual([])
	})
})

describe("getLegacyMSC1767Text", () => {
	test("returns empty string for missing content", () => {
		expect(getLegacyMSC1767Text(undefined)).toBe("")
	})

	test("returns the text field when present", () => {
		expect(getLegacyMSC1767Text({ "org.matrix.msc1767.text": "hello" })).toBe("hello")
	})

	test("message array items are not strings, so result is empty", () => {
		const content = {
			"org.matrix.msc1767.message": [{ mimetype: "text/plain", body: "hello" }],
		}
		expect(getLegacyMSC1767Text(content)).toBe("")
	})

	test("message array without matching mimetype still returns empty string", () => {
		const content = {
			"org.matrix.msc1767.message": [{ mimetype: "text/html", body: "<b>hi</b>" }],
		}
		expect(getLegacyMSC1767Text(content)).toBe("")
	})

	test("returns empty string when no known fields present", () => {
		expect(getLegacyMSC1767Text({ body: "plain matrix content" } as never)).toBe("")
	})
})
