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

const dayNames = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"]
const monthNames = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"]

const week = 1000 * 60 * 60 * 24 * 7

function isLessThanWeekAgo(time: Date, now: Date) {
	if (time > now) {
		return false
	}
	const weekAgo = new Date(now.getTime() - week)
	const isWeekAgo = time.getDate() === weekAgo.getDate() && time.getMonth() === weekAgo.getMonth()
	return time > weekAgo && !isWeekAgo
}

export const formatPreviewTime = (time: Date) => {
	const now = new Date()
	if (
		time.getFullYear() === now.getFullYear()
		&& time.getMonth() === now.getMonth()
		&& time.getDate() == now.getDate()
	) {
		return formatShortTime(time)
	} else if (isLessThanWeekAgo(time, now)) {
		return dayNames[time.getDay()]
	} else if (now.getFullYear() === time.getFullYear()) {
		return `${monthNames[time.getMonth()]} ${time.getDate().toString().padStart(2, "0")}`
	} else {
		return time.getFullYear().toString()
	}
}
