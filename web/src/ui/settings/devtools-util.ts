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

export enum EventKind {
	None,
	Message,
	State,
	AccountData,
	RoomAccountData,
	Profile,
	PushRules,
}

export function kindName(kind: EventKind): string {
	switch (kind) {
	case EventKind.None:
		return ""
	case EventKind.Message:
		return "Message"
	case EventKind.State:
		return "Room State"
	case EventKind.AccountData:
		return "Account Data"
	case EventKind.RoomAccountData:
		return "Room Account Data"
	case EventKind.Profile:
		return "Profile"
	case EventKind.PushRules:
		return "Push Rules"
	}
}

export type DoneCallback = (
	kind: EventKind, type: string, stateKey: string | undefined, content: unknown | undefined,
) => void
