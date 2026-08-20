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
import React, { useRef } from "react"

interface SwipeState {
	startX: number
	startY: number
	triggered: boolean
}

const hasHorizontalScroller = (target: HTMLElement | null, parent: HTMLDivElement) => {
	if (target === parent || !target) {
		return false
	}
	if (target.scrollWidth > target.clientWidth) {
		const style = window.getComputedStyle(target)
		if (style.overflowX === "scroll" || style.overflowX === "auto") {
			return true
		}
	}
	return hasHorizontalScroller(target.parentElement, parent)
}

export interface UseSwipeParams {
	startThreshold: number
	verticalLimit: number
	maxDistance: number
	minTriggerDistance: number
	left: boolean
	enabled: boolean
	onTrigger: () => void
}

const noopFunc = () => {}

export const useHorizontalSwipe = ({
	startThreshold, verticalLimit, minTriggerDistance, maxDistance, left, onTrigger, enabled,
}: UseSwipeParams) => {
	const swipeState = useRef<SwipeState | null>(null)
	if (!enabled || window.ontouchstart === undefined) {
		return [noopFunc, noopFunc, noopFunc, noopFunc] as const
	}
	const onEndSwipe = (target: HTMLDivElement) => {
		if (swipeState.current) {
			target.style.transition = swipeState.current.triggered ? "translate 0.1s linear" : "none"
			target.style.translate = ""
			swipeState.current = null
		}
	}
	const onTouchStart = (evt: React.TouchEvent<HTMLDivElement>) => {
		if (evt.touches.length === 1 && !hasHorizontalScroller(evt.target as HTMLElement, evt.currentTarget)) {
			swipeState.current = {
				startX: evt.touches[0].clientX,
				startY: evt.touches[0].clientY,
				triggered: false,
			}
			evt.currentTarget.style.transition = "none"
		} else {
			onEndSwipe(evt.currentTarget)
		}
	}
	const onTouchMove = (evt: React.TouchEvent<HTMLDivElement>) => {
		if (!swipeState.current) {
			return
		}
		const deltaX = (evt.touches[0].clientX - swipeState.current.startX) * (left ? -1 : 1)
		const deltaY = Math.abs(evt.touches[0].clientY - swipeState.current.startY)
		if (!swipeState.current.triggered) {
			if (deltaX > startThreshold) {
				swipeState.current.triggered = true
			} else if (deltaY > verticalLimit) {
				onEndSwipe(evt.currentTarget)
				return
			} else {
				return
			}
		}
		const translate = Math.min(Math.max(deltaX - startThreshold, 0), maxDistance)
		evt.currentTarget.style.translate = `${left ? "-" : ""}${translate}px 0`
	}
	const onTouchEnd = (evt: React.TouchEvent<HTMLDivElement>) => {
		if (swipeState.current?.triggered) {
			const deltaX = (evt.changedTouches[0].clientX - swipeState.current.startX) * (left ? -1 : 1)
			if (deltaX > minTriggerDistance + startThreshold) {
				onTrigger()
			}
		}
		onEndSwipe(evt.currentTarget)
	}
	const onTouchCancel = (evt: React.TouchEvent<HTMLDivElement>) => {
		onEndSwipe(evt.currentTarget)
	}
	return  [
		onTouchStart,
		onTouchMove,
		onTouchEnd,
		onTouchCancel,
	] as const
}
