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
import React, { JSX, useCallback, useEffect, useLayoutEffect, useReducer, useRef } from "react"
import { ModalCloseContext, ModalContext, ModalState } from "./contexts.ts"

const ModalWrapper = ({ children }: { children: React.ReactNode }) => {
	const [state, setState] = useReducer((prevState: ModalState | null, newState: ModalState | null) => {
		prevState?.onClose?.()
		return newState
	}, null)
	const onClickWrapper = useCallback((evt?: React.MouseEvent) => {
		if (evt && evt.target !== evt.currentTarget) {
			return
		}
		setState(null)
		if (history.state?.modal) {
			history.back()
		}
	}, [])
	const onKeyWrapper = (evt: React.KeyboardEvent<HTMLDivElement>) => {
		if (evt.key === "Escape") {
			setState(null)
			if (history.state?.modal) {
				history.back()
			}
		}
		evt.stopPropagation()
	}
	const openModal = useCallback((newState: ModalState) => {
		if (!history.state?.modal && newState.captureInput !== false) {
			history.pushState({ ...(history.state ?? {}), modal: true }, "")
		}
		setState(newState)
	}, [])
	const wrapperRef = useRef<HTMLDivElement>(null)
	useLayoutEffect(() => {
		if (wrapperRef.current && (!document.activeElement || !wrapperRef.current.contains(document.activeElement))) {
			wrapperRef.current.focus()
		}
	}, [state])
	useEffect(() => {
		window.closeModal = onClickWrapper
		const listener = (evt: PopStateEvent) => {
			if (!evt.state?.modal) {
				setState(null)
			}
		}
		window.addEventListener("popstate", listener)
		return () => window.removeEventListener("popstate", listener)
	}, [onClickWrapper])
	let modal: JSX.Element | null = null
	if (state) {
		let content = <ModalCloseContext value={onClickWrapper}>{state.content}</ModalCloseContext>
		if (state.boxed) {
			content = <div className={`modal-box ${state.boxClass ?? ""}`}>
				<div className={`modal-box-inner ${state.innerBoxClass ?? ""}`}>
					{content}
				</div>
			</div>
		}
		if (state.captureInput !== false) {
			modal = <div
				className={`overlay modal ${state.dimmed ? "dimmed" : ""}`}
				onClick={onClickWrapper}
				onKeyDown={onKeyWrapper}
				tabIndex={-1}
				ref={wrapperRef}
			>
				{content}
			</div>
		} else {
			modal = content
		}
	}
	return <ModalContext value={openModal}>
		{children}
		{modal}
	</ModalContext>
}

export default ModalWrapper
