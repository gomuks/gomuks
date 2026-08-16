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
// GNU General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
import { describe, expect, test } from "vitest"
import { getEventLevel, getUserLevel, preV12 } from "./powerlevel"
import { MemDBEvent, PowerLevelEventContent } from "@/api/types"

const creator: MemDBEvent = {
	sender: "@creator:example.com",
	content: { room_version: "12" },
} as MemDBEvent

describe("preV12", () => {
	test("contains room versions 1-11 and legacy values", () => {
		for (let i = 1; i <= 11; i++) {
			expect(preV12.has(String(i))).toBe(true)
		}
		expect(preV12.has(undefined)).toBe(true)
		expect(preV12.has("")).toBe(true)
	})

	test("does not contain room version 12 or 10-style version 10/11 in other formats", () => {
		expect(preV12.has("12")).toBe(false)
		expect(preV12.has("0")).toBe(false)
		expect(preV12.has("v10")).toBe(false)
	})
})

describe("getUserLevel", () => {
	test("returns Infinity for creator in rooms with room version 12+", () => {
		expect(getUserLevel(undefined, creator, "@creator:example.com")).toBe(Infinity)
	})

	test("does not return Infinity for creator in pre-v12 rooms", () => {
		const oldRoom: MemDBEvent = {
			sender: "@creator:example.com",
			content: { room_version: "10" },
		} as MemDBEvent
		expect(getUserLevel(undefined, oldRoom, "@creator:example.com")).toBe(0)
	})

	test("does not return Infinity when room version is missing (pre-v12 default)", () => {
		const noVersion: MemDBEvent = {
			sender: "@creator:example.com",
			content: {},
		} as MemDBEvent
		expect(getUserLevel(undefined, noVersion, "@creator:example.com")).toBe(0)
	})

	test("returns Infinity for additional creator in v12 rooms", () => {
		const created: MemDBEvent = {
			sender: "@creator:example.com",
			content: { room_version: "12", additional_creators: ["@extra:example.com"] },
		} as unknown as MemDBEvent
		expect(getUserLevel(undefined, created, "@extra:example.com")).toBe(Infinity)
		expect(getUserLevel(undefined, created, "@creator:example.com")).toBe(Infinity)
	})

	test("does not return Infinity for non-creators even in v12 rooms", () => {
		expect(getUserLevel(undefined, creator, "@other:example.com")).toBe(0)
	})

	test("prefers per-user level from power levels", () => {
		const pls = { users: { "@user:example.com": 75 } } as PowerLevelEventContent
		expect(getUserLevel(pls, undefined, "@user:example.com")).toBe(75)
	})

	test("falls back to users_default", () => {
		const pls = { users_default: 25 } as PowerLevelEventContent
		expect(getUserLevel(pls, undefined, "@user:example.com")).toBe(25)
	})

	test("falls back to 0 when neither users nor users_default exist", () => {
		const pls = {} as PowerLevelEventContent
		expect(getUserLevel(pls, undefined, "@user:example.com")).toBe(0)
		expect(getUserLevel(undefined, undefined, "@user:example.com")).toBe(0)
	})

	test("per-user level overrides users_default", () => {
		const pls = { users: { "@user:example.com": 100 }, users_default: 10 } as PowerLevelEventContent
		expect(getUserLevel(pls, undefined, "@user:example.com")).toBe(100)
		expect(getUserLevel(pls, undefined, "@other:example.com")).toBe(10)
	})
})

describe("getEventLevel", () => {
	test("prefers per-event-type level", () => {
		const pls = { events: { "m.room.name": 30 } } as PowerLevelEventContent
		expect(getEventLevel(pls, "m.room.name", true)).toBe(30)
	})

	test("state events default to state_default or 50", () => {
		expect(getEventLevel({} as PowerLevelEventContent, "m.room.topic", true)).toBe(50)
		expect(getEventLevel(undefined, "m.room.topic", true)).toBe(50)
		const pls = { state_default: 25 } as PowerLevelEventContent
		expect(getEventLevel(pls, "m.room.topic", true)).toBe(25)
	})

	test("message events default to events_default or 0", () => {
		expect(getEventLevel({} as PowerLevelEventContent, "m.room.message")).toBe(0)
		expect(getEventLevel(undefined, "m.room.message")).toBe(0)
		const pls = { events_default: 5 } as PowerLevelEventContent
		expect(getEventLevel(pls, "m.room.message")).toBe(5)
	})
})
