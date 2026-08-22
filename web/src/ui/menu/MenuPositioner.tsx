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
import React, { CSSProperties, ComponentType, useLayoutEffect, useRef, useState } from "react"
import { getWindowMargins } from "@/util/cssparse.ts"
import { getModalStyleFromButton, getRightOpeningModalStyleFromButton } from "./util.ts"

interface ChildProps<T extends HTMLElement> {
	ref: React.Ref<T>
	style: CSSProperties
}

interface ExactPosition {
	x: number
	y: number
	element?: never
	anchor: "click" | "touch"
}

interface ElementPosition {
	x?: never
	y?: never
	element: HTMLElement
	anchor: "left" | "right"
}

type PositioningProps = ExactPosition | ElementPosition

type MenuPositionerProps<T extends object, P extends HTMLElement> = T & PositioningProps & {
	Child: ComponentType<Omit<T, "Child" | keyof PositioningProps> & ChildProps<P>>
}

const defaultStyle: CSSProperties = { visibility: "hidden" }

export const MenuPositioner = <T extends object, P extends HTMLElement>({
	Child, x, y, element, anchor, ...props
}: MenuPositionerProps<T, P>) => {
	const ref = useRef<P>(null)
	const [style, setStyle] = useState<CSSProperties>(defaultStyle)
	useLayoutEffect(() => {
		if (!ref.current) {
			return
		}
		const [topMargin, bottomMargin] = getWindowMargins()
		const width = ref.current.offsetWidth
		const height = ref.current.offsetHeight
		const finalStyle: CSSProperties = {}
		if (anchor === "click") {
			if (x + width + 4 > window.innerWidth) {
				finalStyle.right = "4px"
			} else {
				finalStyle.left = x
			}
			if (y + height + bottomMargin > window.innerHeight) {
				finalStyle.bottom = window.innerHeight - y
			} else {
				finalStyle.top = y
			}
		} else if (anchor === "touch") {
			const standardLeft = Math.max(x - width - 16, 4)
			finalStyle.left = standardLeft
			const standardTop = Math.max(y - height / 2, 4) + topMargin
			let transformTop = standardTop
			if (standardTop + height + bottomMargin > window.innerHeight) {
				finalStyle.bottom = "calc(var(--window-bottom-margin) + 4px)"
				transformTop = window.innerHeight - bottomMargin - height - 4
			} else {
				finalStyle.top = standardTop
			}
			if (!window.matchMedia("(prefers-reduced-motion)").matches) {
				finalStyle.animation = "modal-grow 100ms linear"
			}
			finalStyle.transformOrigin = `${x - standardLeft}px ${y - transformTop}px`
		} else if (element) {
			if (anchor === "left") {
				setStyle(getModalStyleFromButton(element, height))
			} else {
				setStyle(getRightOpeningModalStyleFromButton(element, height))
			}
			return
		}
		setStyle(finalStyle)
	}, [x, y, element, anchor])
	return <Child ref={ref} style={style} {...props} />
}
