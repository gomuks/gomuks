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
import { use, useEffect, useState } from "react"
import { ScaleLoader } from "react-spinners"
import Client from "@/api/client.ts"
import { SanitizedBio, UserID } from "@/api/types"
import { ModalCloseContext, ModalContext } from "../modal"
import { SanitizedHTMLView } from "../timeline/content/TextMessageBody"

interface UserProfileNotesProps {
	client: Client
	userID: UserID
}
interface UserProfileFullNotesProps extends UserProfileNotesProps {
	note: SanitizedBio
	setNote: (note: SanitizedBio) => void
}

export const UserProfileFullNotes = ({
	note,
	client,
	userID,
	setNote,
}: UserProfileFullNotesProps) => {
	const closeModal = use(ModalCloseContext)
	const [editing, setEditing] = useState<string | undefined>(undefined)
	const startEdit = () => {
		setEditing(note.edit_source ?? "")
	}
	const saveEdit = () => {
		client.rpc.setProfileAnnotation(userID, editing).then(
			(newValue) => {
				setNote(newValue.note ?? { html: "" })
				closeModal()
			},
			err => window.alert(`Failed to save notes: ${err}`),
		)
	}
	const cancelEdit = () => {
		setEditing(undefined)
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

export const UserProfileNotes = ({ client, userID }: UserProfileNotesProps) => {
	const [note, setNote] = useState<SanitizedBio | null>(null)
	const [loading, setLoading] = useState(false)

	const initialNoteLoad = () => {
		setLoading(true)
		client.rpc.getProfileAnnotation(userID).then(
			({ note }) => setNote(note ?? { html: "" }),
			err => window.alert(`Failed to load notes: ${err}`),
		).finally(() => setLoading(false))
	}

	useEffect(initialNoteLoad, [client, userID])

	if (loading || note === null) {
		return <div className="profile-notes">
			<h4>Notes</h4>
			<ScaleLoader className="user-info-loader" color="var(--primary-color)"/>
		</div>
	}

	const openModal = use(ModalContext)
	const viewFull = () => {
		openModal({
			content: <UserProfileFullNotes client={client} userID={userID}
				note={note} setNote={setNote} />,
			boxed: true,
			dimmed: true,
			innerBoxClass: "profile-notes-modal",
			boxClass: "profile-notes-modal-wrapper",
		})
	}

	return <div className="profile-notes-wrapper">
		<h4>Notes</h4>
		<div className="profile-notes">
			<SanitizedHTMLView html={note.html} />
			<button onClick={viewFull}>View/Edit notes</button>
		</div>
	</div>
}
