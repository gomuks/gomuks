// gomuks - A Matrix client written in Go.
// Copyright (C) 2026 Logan Devine
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
import { ScaleLoader } from "react-spinners"
import Client from "@/api/client.ts"
import { SanitizedBio, UserID } from "@/api/types"
import { ModalCloseContext, ModalContext } from "../modal"
import { SanitizedHTMLView } from "../timeline/content/TextMessageBody"

interface UserProfileNotesProps {
	note: SanitizedBio
	client: Client
	refreshProfile: () => void
	userID: UserID
	editByDefault?: boolean
	loadingGlobalProfile?: boolean
}

const UserProfileFullNotes = ({
	note,
	client,
	userID,
	refreshProfile,
	editByDefault,
}: UserProfileNotesProps) => {
	const closeModal = use(ModalCloseContext)
	const [editing, setEditing] = useState<string | undefined>(editByDefault ? note.edit_source ?? "" : undefined)
	const startEdit = () => {
		setEditing(note.edit_source ?? "")
	}
	const saveEdit = () => {
		client.rpc.setProfileAnnotation(userID, editing).then(
			() => {
				refreshProfile()
				closeModal()
			},
			err => window.alert(`Failed to save notes: ${err}`),
		)
	}
	const cancelEdit = () => {
		setEditing(undefined)
		if (editByDefault) {
			closeModal()
		}
	}

	if (editing === undefined) {
		return <>
			<SanitizedHTMLView html={note.html} />
			<div className="buttons">
				<div/>
				<button onClick={startEdit}>Edit</button>
			</div>
		</>
	}

	return <>
		<textarea
			placeholder="User notes"
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

const UserProfileNotes = ({ note, loadingGlobalProfile, ...rest }: UserProfileNotesProps) => {
	const openModal = use(ModalContext)
	const editByDefault = !note.html
	const viewFull = () => {
		openModal({
			content: <UserProfileFullNotes note={note} {...rest} editByDefault={editByDefault} />,
			boxed: true,
			dimmed: true,
			innerBoxClass: "profile-notes-modal",
			boxClass: "profile-notes-modal-wrapper",
		})
	}

	if (loadingGlobalProfile) {
		return <div className="profile-notes">
			<h4>Notes</h4>
			<ScaleLoader className="user-info-loader" color="var(--primary-color)"/>
		</div>
	}

	return <div className="profile-notes-wrapper">
		<h4>Notes</h4>
		<div className="profile-notes">
			<SanitizedHTMLView html={note.html} />
			<button onClick={viewFull}>View/Edit notes</button>
		</div>
	</div>
}

export default UserProfileNotes
