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
const fullTimeFormatter = new Intl.DateTimeFormat("en-GB", { dateStyle: "full", timeStyle: "medium" })
const dateFormatter = new Intl.DateTimeFormat("en-GB", { dateStyle: "full" })

export const formatShortTime = (time: Date) =>
	`${time.getHours().toString().padStart(2, "0")}:${time.getMinutes().toString().padStart(2, "0")}`
export const formatFullTime = (time: Date) => fullTimeFormatter.format(time)
export const formatDate = (time: Date) => dateFormatter.format(time)
export const newSafeDate = (val: number) => {
	const date = new Date(val)
	if (isNaN(+date)) {
		return new Date(0)
	}
	return date
}
