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
import { describe, expect, test, vi } from "vitest"
import type { RoomStateStore } from "@/api/statestore"
import type { MemDBEvent } from "@/api/types"
import { displayAsRedacted } from "./displayAsRedacted"

const evt = (partial: Partial<MemDBEvent>): MemDBEvent => partial as unknown as MemDBEvent
const memberEvt = (partial: Partial<MemDBEvent>): MemDBEvent => partial as unknown as MemDBEvent

const bannedRedactingMember = memberEvt({
	sender: "@banned:example.com",
	content: {
		membership: "ban",
		"org.matrix.msc4293.redact_events": true,
	},
})

const fakeRoom = (state: Record<string, MemDBEvent | undefined> = {}): RoomStateStore => {
	const getStateEvent = vi.fn((type: string, key: string) => state[`${type}/${key}`])
	return { getStateEvent } as unknown as RoomStateStore
}

describe("displayAsRedacted", () => {
	test("viewing_redacted event is never displayed as redacted", () => {
		expect(displayAsRedacted(evt({ viewing_redacted: true }))).toBe(false)
	})

	test("viewing_redacted takes precedence over redacted_by", () => {
		expect(displayAsRedacted(evt({ viewing_redacted: true, redacted_by: "$redaction" }))).toBe(false)
	})

	test("redacted_by event is displayed as redacted", () => {
		expect(displayAsRedacted(evt({ redacted_by: "$redaction" }))).toBe(true)
	})

	test("plain event is not displayed as redacted", () => {
		expect(displayAsRedacted(evt({}), null, fakeRoom())).toBe(false)
	})

	test("missing member event is not redacted", () => {
		expect(displayAsRedacted(evt({}), undefined, fakeRoom())).toBe(false)
	})

	test("member without ban or redact_events is not redacted", () => {
		const member = memberEvt({
			sender: "@member:example.com",
			content: { membership: "join" },
		})
		expect(displayAsRedacted(evt({}), member, fakeRoom())).toBe(false)

		const unbannedRedacting = memberEvt({
			sender: "@member:example.com",
			content: { membership: "leave", "org.matrix.msc4293.redact_events": true },
		})
		expect(displayAsRedacted(evt({}), unbannedRedacting, fakeRoom())).toBe(false)
	})

	test("banned redacting member without room is displayed as redacted", () => {
		expect(displayAsRedacted(evt({}), bannedRedactingMember, undefined)).toBe(true)
	})

	test("banned redacting member with insufficient power level is not redacted", () => {
		const room = fakeRoom({
			"m.room.power_levels/": evt({
				content: { redact: 50, users: { "@banned:example.com": 25 } },
			}),
		})
		expect(displayAsRedacted(evt({}), bannedRedactingMember, room)).toBe(false)
	})

	test("banned redacting member with sufficient power level is redacted", () => {
		const room = fakeRoom({
			"m.room.power_levels/": evt({
				content: { redact: 50, users: { "@banned:example.com": 50 } },
			}),
		})
		expect(displayAsRedacted(evt({}), bannedRedactingMember, room)).toBe(true)
	})

	test("missing power levels defaults redact to 50 and users to 0", () => {
		const room = fakeRoom({})
		expect(displayAsRedacted(evt({}), bannedRedactingMember, room)).toBe(false)
	})

	test("power levels without redact key default redact PL to 50", () => {
		const room = fakeRoom({
			"m.room.power_levels/": evt({
				content: { users: { "@banned:example.com": 50 } },
			}),
		})
		expect(displayAsRedacted(evt({}), bannedRedactingMember, room)).toBe(true)
	})

	test("power levels without users map use users_default", () => {
		const room = fakeRoom({
			"m.room.power_levels/": evt({
				content: { redact: 75, users_default: 74 },
			}),
		})
		expect(displayAsRedacted(evt({}), bannedRedactingMember, room)).toBe(false)
	})

	test("room creator has infinite power in rooms v12 and above", () => {
		const room = fakeRoom({
			"m.room.power_levels/": evt({
				content: { redact: 9001 },
			}),
			"m.room.create/": evt({
				sender: "@banned:example.com",
				content: { room_version: "12" },
			}),
		})
		expect(displayAsRedacted(evt({}), bannedRedactingMember, room)).toBe(true)
	})

	test("room creator in pre-v12 rooms does not get infinite power", () => {
		const room = fakeRoom({
			"m.room.power_levels/": evt({
				content: { redact: 9001 },
			}),
			"m.room.create/": evt({
				sender: "@banned:example.com",
				content: { room_version: "11" },
			}),
		})
		expect(displayAsRedacted(evt({}), bannedRedactingMember, room)).toBe(false)
	})

	test("room creator without room version does not get infinite power", () => {
		const room = fakeRoom({
			"m.room.power_levels/": evt({
				content: { redact: 9001 },
			}),
			"m.room.create/": evt({
				sender: "@banned:example.com",
				content: {},
			}),
		})
		expect(displayAsRedacted(evt({}), bannedRedactingMember, room)).toBe(false)
	})

	test("additional creator has infinite power in rooms v12 and above", () => {
		const room = fakeRoom({
			"m.room.power_levels/": evt({
				content: { redact: 9001 },
			}),
			"m.room.create/": evt({
				sender: "@creator:example.com",
				content: { room_version: "12", additional_creators: ["@banned:example.com"] },
			}),
		})
		expect(displayAsRedacted(evt({}), bannedRedactingMember, room)).toBe(true)
	})

	test("create event for a different sender without additional_creators uses PLs", () => {
		const room = fakeRoom({
			"m.room.power_levels/": evt({
				content: { redact: 50, users: { "@banned:example.com": 50 } },
			}),
			"m.room.create/": evt({
				sender: "@other:example.com",
				content: { room_version: "12" },
			}),
		})
		expect(displayAsRedacted(evt({}), bannedRedactingMember, room)).toBe(true)
	})
})
