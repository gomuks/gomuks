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
import React from "react"

export interface ErrorBoundaryProps {
	thing?: string
	wrapperClassName?: string
	children: React.ReactNode
}

export default class ErrorBoundary extends React.Component<ErrorBoundaryProps, { error?: string }> {
	constructor(props: ErrorBoundaryProps) {
		super(props)
		this.state = { error: undefined }
	}

	static getDerivedStateFromError(error: unknown) {
		return {
			error: `${error}`.replace(/^Error: /, ""),
		}
	}

	renderError(message: string) {
		const inner = <>
			Failed to render {this.props.thing ?? "component"}: {message}
		</>
		if (this.props.wrapperClassName) {
			return <div className={this.props.wrapperClassName}>{inner}</div>
		}
		return inner
	}

	render() {
		if (this.state.error) {
			return this.renderError(this.state.error)
		}
		return this.props.children
	}
}
