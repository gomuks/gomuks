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
import { CSSProperties, use } from "react"
import { TabInfo } from "@/api/tabs.ts"
import { ModalCloseContext } from "../modal"

interface ProfileSwitcherProps {
	tabs: readonly TabInfo[]
	currentTabID: string
	switchTab: (tabID: string) => void
	style: CSSProperties
}

const ProfileSwitcher = ({ tabs, currentTabID, switchTab, style }: ProfileSwitcherProps) => {
	const closeModal = use(ModalCloseContext)
	return <div className="context-menu profile-switcher-menu" style={style}>
		{tabs.map(tab => tab.id !== currentTabID ? <button
			key={tab.id}
			onClick={() => {
				switchTab(tab.id)
				closeModal()
			}}
		>
			<div>
				{tab.icon && <img src={tab.icon} className="avatar" alt="" />}
				{tab.displayname || tab.id}
			</div>
			{tab.unread > 0 && <div className="room-entry-unreads">
				<div className="unread-count notified">
					{tab.unread > 999 ? "99+" : tab.unread}
				</div>
			</div>}
		</button> : null)}
		{window.gomuksDesktop && <button onClick={() => window.gomuksDesktop?.quitApp()}>
			Quit
		</button>}
	</div>
}

export default ProfileSwitcher
