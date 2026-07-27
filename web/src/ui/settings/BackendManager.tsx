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
import React, { Fragment, use, useState } from "react"
import { TabInfo, TabInfoUpdate, useTabs } from "@/api/tabs.ts"
import { ModalCloseContext, ModalContext } from "@/ui/modal"
import "./BackendManager.css"

interface BackendInfoProps {
	tab?: TabInfo
	isCurrent?: boolean
	updateTab: (tab: TabInfoUpdate) => Promise<void>
	deleteTab: (tabID: string) => Promise<void>
}

const BackendInfo = ({ tab, isCurrent, updateTab, deleteTab }: BackendInfoProps) => {
	const [type, setType] = useState(tab?.type ?? "embedded")
	const [id, setID] = useState(tab?.id ?? "")
	const [name, setName] = useState(tab?.displayname ?? "")
	const [disableNotifications, setDisableNotifications] = useState(tab?.disable_notifications ?? false)
	const [address, setAddress] = useState(tab?.address || "")
	const [username, setUsername] = useState(tab?.username || "")
	const [password, setPassword] = useState(tab?.password || "")
	const [loading, setLoading] = useState(false)
	const isNew = !tab
	const closeModal = isNew ? use(ModalCloseContext) : undefined

	const onClickDelete = () => {
		const message = tab?.type === "embedded"
			? `Really delete ${name}? Embedded backend data will be permanently deleted.`
			: `Really delete ${name}?`
		if (!window.confirm(message)) {
			return
		}
		setLoading(true)
		deleteTab(tab!.id).
			catch(err => window.alert(`Failed to delete backend: ${err}`)).
			finally(() => setLoading(false))
	}
	const onClickSave = (evt: React.SubmitEvent) => {
		evt.preventDefault()
		setLoading(true)
		updateTab({
			id,
			...tab,
			type,
			displayname: name || id,
			disable_notifications: disableNotifications,
			address,
			username,
			password: !password && tab ? undefined : password,
		}).then(
			() => {
				console.info("Updated tab", id)
				closeModal?.()
			},
			err => window.alert(`Failed to update backend: ${err}`),
		).finally(() => setLoading(false))
	}

	const tabID = tab?.id ?? "new-tab"
	const hasNecessaryFields = Boolean(id && (type === "embedded" || address))
	return <form className="backend" onSubmit={onClickSave}>
		<div className="fields">
			<label htmlFor={`backend-id-${tabID}`}>ID</label>
			<input
				type="text"
				value={id}
				onChange={e => setID(e.target.value)}
				disabled={!isNew}
				id={`backend-id-${tabID}`}
				pattern="^(?:\w|-){1,32}$"
				required
				placeholder="mynewbackend"
			/>
			<label htmlFor={`backend-name-${tabID}`}>Name</label>
			<input
				type="text"
				value={name}
				onChange={e => setName(e.target.value)}
				id={`backend-name-${tabID}`}
				placeholder="My New Backend"
			/>
			<label htmlFor={`backend-notifications-${tabID}`}>Notifications</label>
			<input
				type="checkbox"
				checked={!disableNotifications}
				onChange={e => setDisableNotifications(!e.target.checked)}
				id={`backend-notifications-${tabID}`}
			/>
			<label htmlFor={`backend-type-${tabID}`}>Type</label>
			<select
				value={type}
				onChange={e => setType(e.target.value as TabInfo["type"])}
				disabled={isCurrent}
				id={`backend-type-${tabID}`}
			>
				<option value="embedded">Embedded</option>
				<option value="remote">Remote</option>
			</select>
			{type === "remote" && <>
				<label htmlFor={`backend-address-${tabID}`}>Address</label>
				<input
					type="text"
					value={address}
					onChange={e => setAddress(e.target.value)}
					id={`backend-address-${tabID}`}
					placeholder="e.g. http://localhost:29325"
					required
				/>
				<label htmlFor={`backend-username-${tabID}`}>Username</label>
				<input
					type="text"
					value={username}
					onChange={e => setUsername(e.target.value)}
					id={`backend-username-${tabID}`}
					placeholder="basic auth username"
				/>
				<label htmlFor={`backend-password-${tabID}`}>Password</label>
				<input
					type="password"
					value={password}
					onChange={e => setPassword(e.target.value)}
					id={`backend-password-${tabID}`}
					placeholder={tab?.type === "remote" ? "unchanged" : "correct horse battery staple"}
				/>
			</>}
		</div>
		<div className="buttons">
			{!isCurrent && tab && <button
				onClick={onClickDelete}
				disabled={loading}
				type="button"
				className="delete-button"
			>Delete</button>}
			<button disabled={loading || !hasNecessaryFields} type="submit">{isNew ? "Create" : "Save"}</button>
		</div>
	</form>
}

const BackendManager = () => {
	const tabsData = useTabs()
	const openModal = use(ModalContext)
	if (!tabsData.hasTabs) {
		return <div className="backend-manager unsupported">
			This wrapper doesn't support multiple backends.
		</div>
	}
	const { tabs, currentTabID, updateTab, deleteTab } = tabsData
	const onClickNewBackend = () => {
		openModal({
			boxed: true,
			dimmed: true,
			content: <BackendInfo updateTab={updateTab} deleteTab={deleteTab} />,
		})
	}
	return <div className="backend-manager">
		<button onClick={onClickNewBackend} className="new-backend">New backend</button>
		{tabs.map(tab => <Fragment key={tab.id}>
			<hr/>
			<BackendInfo
				isCurrent={tab.id === currentTabID}
				tab={tab}
				updateTab={updateTab}
				deleteTab={deleteTab}
			/>
		</Fragment>)}
	</div>
}

export default BackendManager
