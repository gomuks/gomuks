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
import { CreateEventContent, MemDBEvent, PowerLevelEventContent, UserID } from "@/api/types"

export const preV12 = new Set([undefined, "", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"])

export function getUserLevel(
	pls: PowerLevelEventContent | undefined,
	createEvt: MemDBEvent | undefined | null,
	userID: UserID,
): number {
	const create = createEvt?.content as CreateEventContent | undefined
	if (
		createEvt
		&& create
		&& !preV12.has(create.room_version)
		&& (createEvt.sender === userID || create.additional_creators?.includes(userID))
	) {
		return Infinity
	}
	return pls?.users?.[userID] ?? pls?.users_default ?? 0
}

export function getEventLevel(
	pls: PowerLevelEventContent | undefined,
	eventType: string,
	state: boolean = false,
): number {
	return pls?.events?.[eventType] ?? (state ? pls?.state_default : pls?.events_default) ?? (state ? 50 : 0)
}
