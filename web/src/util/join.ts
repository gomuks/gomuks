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
function stripBidi(text: string) {
	return text.replace(/[\u202A-\u202E]/g, "")
}

export function humanJoin(arr: string[], sep: string = ", ", lastSep: string = " and "): string {
	if (arr.length === 0) {
		return ""
	}
	if (arr.length === 1) {
		return stripBidi(arr[0])
	}
	if (arr.length === 2) {
		return arr.map(stripBidi).join(lastSep)
	}
	return arr.slice(0, -1).map(stripBidi).join(sep) + lastSep + stripBidi(arr[arr.length - 1])
}
