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
import { use, useState } from "react"
import ClientContext from "../ClientContext.ts"

const currentVersion = (
	document.querySelector("meta[name=gomuks-version-description]") as HTMLMetaElement
)?.content

const MiscButtons = () => {
	const client = use(ClientContext)!
	const onClickLogout = () => {
		if (window.confirm("Really log out and delete all local data?")) {
			client.logout().then(
				() => console.info("Successfully logged out"),
				err => window.alert(`Failed to log out: ${err}`),
			)
		}
	}
	const onClickOpenCSSApp = () => {
		client.rpc.requestOpenIDToken().then(
			resp => window.open(
				`https://css.gomuks.app/login?token=${resp.access_token}&server_name=${resp.matrix_server_name}`,
				"_blank",
				"noreferrer noopener",
			),
			err => window.alert(`Failed to request OpenID token: ${err}`),
		)
	}
	const [clearing, setClearing] = useState(false)
	const clearCache = () => {
		setClearing(true)
		client.store.deleteCache().then(
			() => {
				console.log("Cleared state cache, reloading")
				window.location.reload()
			},
			err => window.alert(`Failed to clear cache: ${err}`),
		).finally(() => setClearing(false))
	}
	return <div className="misc-buttons">
		<button onClick={onClickOpenCSSApp}>Sign into css.gomuks.app</button>
		{window.Notification && !window.gomuksAndroid && <button onClick={client.requestNotificationPermission}>
			Request notification permission
		</button>}
		{!window.gomuksAndroid &&
			<button onClick={client.registerURIHandler}>Register <code>matrix:</code> URI handler</button>
		}
		{client.store.anyStateCache ? <button onClick={clearCache} disabled={clearing}>
			{clearing ? "Clearing cache..." : "Clear cache and reload"}
		</button> : null}
		<p>
			State cache status: {client.store.stateCacheStatus}
		</p>
		<p>
			{currentVersion || "Version unknown"}
		</p>
		<div className="spacer" />
		<button className="logout" onClick={onClickLogout}>Logout</button>
	</div>
}

export default MiscButtons
