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
import { RoomStateStore } from "@/api/statestore"
import { KeyRestoreProgress, RoomID } from "@/api/types"
import { NonNullCachedEventDispatcher, useEventAsState } from "@/util/eventdispatcher.ts"
import { ModalContext } from "../modal"

export interface KeyRestoreStatus {
	progress: KeyRestoreProgress
	connected: boolean
	done?: "ok" | string
}

const KeyRestoreProgressModal = ({ evt }: { evt: NonNullCachedEventDispatcher<KeyRestoreStatus> }) => {
	const status = useEventAsState(evt)
	const prog = status.progress
	let statusMessage: string = "Unknown status"
	let handledCountMessage: string = ""

	const decryptedCount = prog.decrypted + prog.decryption_failed + prog.import_failed
	const statusMax = prog.total * 3 - (prog.decryption_failed * 2) - (prog.import_failed * 2)
	const statusValue = prog.stage === "fetching"
		? undefined
		: decryptedCount + prog.saved + prog.post_processed

	if (prog.stage === "fetching") {
		statusMessage = "Fetching keys from server"
	} else if (prog.stage === "decrypting") {
		statusMessage = "Decrypting keys"
		handledCountMessage = `Decrypted ${prog.decrypted} / ${prog.total} keys`
	} else if (prog.stage === "saving") {
		statusMessage = "Saving decrypted keys"
		handledCountMessage = `Saved ${prog.saved} / ${prog.decrypted} keys`
	} else if (prog.stage === "postprocessing") {
		statusMessage = "Decrypting pending messages"
		handledCountMessage = `Post-processed ${prog.post_processed} / ${prog.decrypted} keys`
	} else if (prog.stage === "done") {
		statusMessage = "Restore completed"
		handledCountMessage = `Successfully restored ${prog.post_processed} / ${prog.total} keys`
	}
	if (status.done && status.done !== "ok") {
		statusMessage = status.done
	} else if (!status.connected) {
		statusMessage = "Connecting to server"
	}
	return <>
		<div className="status">
			{statusMessage}
		</div>
		{prog.current_room_id && !status.done ? <div className="active-room-id">
			Currently processing <code>{prog.current_room_id}</code>
		</div> : null}
		<progress id="key-backup-restore-progress" value={statusValue} max={statusMax}/>

		<label htmlFor="key-backup-restore-progress">
			<div>{handledCountMessage}</div>
			{prog.decryption_failed ? <div>Failed to decrypt {prog.decryption_failed} keys</div> : null}
			{prog.import_failed ? <div>Failed to import {prog.import_failed} keys</div> : null}
		</label>
	</>
}

interface KeyExportViewProps {
	room?: RoomStateStore
}

const KeyExportView = ({ room }: KeyExportViewProps) => {
	const [passphrase, setPassphrase] = useState("")
	const [hasFile, setHasFile] = useState(false)
	const openModal = use(ModalContext)
	const importBackup = (roomID?: RoomID) => {
		let path = "_gomuks/keys/restorebackup"
		if (roomID) {
			path += `/${encodeURIComponent(roomID)}`
		}
		const evtSource = new EventSource(path)
		let progress: KeyRestoreProgress = {
			stage: "fetching",
			current_room_id: "",
			decrypted: 0,
			decryption_failed: 0,
			import_failed: 0,
			saved: 0,
			post_processed: 0,
			total: 0,
		}
		let connected = false
		const disp = new NonNullCachedEventDispatcher<KeyRestoreStatus>({
			progress,
			connected,
		})
		evtSource.addEventListener("progress", evt => {
			progress = JSON.parse(evt.data)
			connected = true
			disp.emit({ progress, connected })
		})
		evtSource.addEventListener("done", evt => {
			disp.emit({ progress, connected, done: evt.data })
			evtSource.close()
		})
		evtSource.addEventListener("error", () => {
			disp.emit({ progress, connected, done: "Failed to connect to server" })
			evtSource.close()
		})
		evtSource.addEventListener("close", () => {
			if (!disp.current.done) {
				disp.emit({ progress, connected, done: "Connection closed unexpectedly" })
			}
			evtSource.close()
		})
		openModal({
			dimmed: true,
			boxed: true,
			content: <KeyRestoreProgressModal evt={disp}/>,
			innerBoxClass: "key-restore-modal",
			boxClass: "key-restore-modal-wrapper",
		})
	}
	return <div className="key-export">
		<h3>Key export/import</h3>
		<input
			className="passphrase"
			type="password"
			value={passphrase}
			onChange={evt => setPassphrase(evt.target.value)}
			placeholder="Passphrase"
		/>
		<form
			className="import-buttons"
			action="_gomuks/keys/import"
			encType="multipart/form-data"
			method="post"
			target="_blank"
		>
			<input type="password" name="passphrase" hidden readOnly value={passphrase} />
			<input
				className="import-file"
				type="file"
				accept="text/plain"
				name="export"
				defaultValue=""
				onChange={evt => setHasFile(!!evt.target.files?.length)}
			/>
			<button type="submit" disabled={passphrase == "" || !hasFile}>Import file</button>
		</form>
		<div className="export-buttons">
			<form action="_gomuks/keys/export" method="post" target="_blank">
				<input type="password" name="passphrase" hidden readOnly value={passphrase} />
				<button type="submit" disabled={passphrase == ""}>Export all keys</button>
			</form>
			{room && (
				<form action={`_gomuks/keys/export/${encodeURIComponent(room.roomID)}`} method="post" target="_blank">
					<input type="password" name="passphrase" hidden readOnly value={passphrase} />
					<button type="submit" disabled={passphrase == ""}>Export room keys</button>
				</form>
			)}
		</div>
		<hr/>
		<div className="key-backup-buttons">
			{room && <button onClick={() => importBackup(room.roomID)}>Import room backup</button>}
			<button onClick={() => importBackup()}>Import entire backup</button>
		</div>
	</div>
}

export default KeyExportView
