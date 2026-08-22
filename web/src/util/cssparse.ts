// gomuks - A Matrix client written in Go.
// Copyright (C) 2026 Tulir Asokan
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

export function parseSimpleCSSUnit(value: string, decl?: CSSStyleDeclaration): number {
	if (value.endsWith("px")) {
		return +value.slice(0, -2)
	} else if (value.endsWith("rem")) {
		const fontSize = +(decl ?? getComputedStyle(document.body)).getPropertyValue("font-size").replace("px", "")
		return +value.slice(0, -3) * fontSize
	} else {
		return +value
	}
}

export function getWindowMargins(): [number, number] {
	const docStyle = getComputedStyle(document.documentElement)
	const topMargin = parseSimpleCSSUnit(docStyle.getPropertyValue("--window-top-margin"), docStyle)
	const bottomMargin = parseSimpleCSSUnit(docStyle.getPropertyValue("--window-bottom-margin"), docStyle)
	return [topMargin || 0, bottomMargin || 0]
}
