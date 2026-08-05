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
import Client from "@/api/client.ts"
import { SanitizedBio, UserID } from "@/api/types"
import { ModalCloseContext, ModalContext } from "../modal"
import { SanitizedHTMLView } from "../timeline/content/TextMessageBody.tsx"

interface UserProfileBioProps {
	bio: SanitizedBio
	client: Client
	refreshProfile: () => void
	userID: UserID
	editByDefault?: boolean
}

const UserProfileFullBio = ({ bio, client, userID, refreshProfile, editByDefault }: UserProfileBioProps) => {
	const closeModal = use(ModalCloseContext)
	const [editing, setEditing] = useState<string | undefined>(editByDefault ? bio.edit_source ?? "" : undefined)
	const startEdit = () => {
		setEditing(bio.edit_source ?? "")
	}
	const saveEdit = () => {
		client.rpc.setProfileField("_gomuks_bio", editing).then(
			() => {
				refreshProfile()
				closeModal()
			},
			err => window.alert(`Failed to save bio: ${err}`),
		)
	}
	const cancelEdit = () => {
		setEditing(undefined)
		if (editByDefault) {
			closeModal()
		}
	}
	if (editing === undefined || userID !== client.userID) {
		return <>
			<SanitizedHTMLView html={bio.html} />
			<div className="buttons">
				<div/>
				{userID === client.userID && <button onClick={startEdit}>Edit</button>}
			</div>
		</>
	}
	return <>
		<textarea
			placeholder="Profile biography"
			value={editing}
			onChange={evt => setEditing(evt.target.value)}
			rows={10}
		/>
		<div className="buttons">
			<button onClick={cancelEdit}>Cancel</button>
			<button onClick={saveEdit}>Save</button>
		</div>
	</>
}

export const UserProfileSmallBio = ({ bio, ...rest }: UserProfileBioProps) => {
	const openModal = use(ModalContext)
	const editByDefault = rest.userID === rest.client.userID && !bio.html
	const viewFull = () => {
		openModal({
			content: <UserProfileFullBio bio={bio} {...rest} editByDefault={editByDefault} />,
			boxed: true,
			dimmed: true,
			innerBoxClass: "profile-full-bio-modal",
			boxClass: "profile-full-bio-modal-wrapper",
		})
	}
	if (!editByDefault && !bio.html) {
		return null
	}
	return <div className="profile-small-bio">
		<SanitizedHTMLView html={bio.html} />
		<button onClick={viewFull}>{editByDefault ? "Edit bio" : "View full bio"}</button>
	</div>
}
