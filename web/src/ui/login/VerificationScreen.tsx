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
import React, { JSX, useState } from "react"
import { ScaleLoader } from "react-spinners"
import { RecoveryKeyResponse } from "@/api/types"
import { LoginScreenProps } from "./LoginScreen.tsx"
import "./LoginScreen.css"

enum ResetState {
	None,
	Preparing,
	Ready,
}

export const VerificationScreen = ({ client, clientState }: LoginScreenProps) => {
	if (!clientState.is_logged_in) {
		throw new Error("Invalid state")
	}
	const [recoveryKey, setRecoveryKey] = useState("")
	const [generatedRecoveryKey, setGeneratedRecoveryKey] = useState<RecoveryKeyResponse | null>(null)
	const [confirmReset, setConfirmReset] = useState(ResetState.None)
	const [error, setError] = useState("")
	const vs = clientState.verification_state

	const verify = (evt: React.SubmitEvent) => {
		evt.preventDefault()
		client.rpc.verify(recoveryKey).then(
			() => {},
			err => setError(err.toString()),
		)
	}

	const logout = () => {
		client.rpc.logout().then(
			() => {},
			err => setError(err.toString()),
		)
	}

	const startReset = () => {
		client.rpc.generateRecoveryKey().then(
			key => {
				setGeneratedRecoveryKey(key)
				setConfirmReset(ResetState.Preparing)
				setTimeout(() => setConfirmReset(val => val === ResetState.Preparing ? ResetState.Ready : val), 5000)
			},
			err => setError(err.toString()),
		)
	}

	const doReset = () => {
		if (vs.has_cross_signing && !window.confirm("Really reset cross-signing keys and key backup?")) {
			return
		}
		client.rpc.resetEncryption(generatedRecoveryKey!, client.passwordCache).then(
			() => {},
			err => setError(err.toString()),
		)
	}
	const cancelReset = () => {
		setGeneratedRecoveryKey(null)
		setConfirmReset(ResetState.None)
	}

	let content: JSX.Element
	if (!vs.state_checked) {
		content = <center>Checking encryption state... <ScaleLoader barCount={8} color="var(--primary-color)"/></center>
	} else if (confirmReset) {
		content = <>
			<p>Store the recovery key below safely, then click confirm to proceed</p>
			<input
				type="text"
				id="mxlogin-recoverykey"
				value={generatedRecoveryKey!.recovery_key}
				disabled
			/>
			<div className="buttons">
				<button onClick={cancelReset} type="button" className="mx-login-button">
					Cancel
				</button>
				<button
					onClick={confirmReset === ResetState.Ready ? doReset : () => {}}
					disabled={confirmReset !== ResetState.Ready}
					type="button"
					className={`mx-login-button ${vs.has_cross_signing ? "reset" : "primary-color"}-button`}
				>
					{vs.has_cross_signing ? "Confirm reset" : "Confirm setup"}
				</button>
			</div>
		</>
	} else {
		content = <>
			{vs.has_ssss ? <>
				<input
					type="text"
					autoComplete="off"
					id="mxlogin-recoverykey"
					placeholder="Recovery key or passphrase"
					value={recoveryKey}
					onChange={evt => setRecoveryKey(evt.target.value)}
				/>
				<button className="mx-login-button primary-color-button" type="submit">Verify</button>
			</> : <p>SSSS not set up, can't verify using recovery key</p>}
			<div className="buttons">
				<button onClick={logout} type="button" className="mx-login-button thin-button">
					Log out
				</button>
				<button
					onClick={startReset}
					type="button"
					className={`mx-login-button ${vs.has_cross_signing ? "thin-button reset" : "primary-color"}-button`}
				>
					{vs.has_cross_signing ? "Reset encryption identity" : "Set up encryption"}
				</button>
			</div>
		</>
	}

	return <main className="matrix-login">
		<h1>gomuks web</h1>
		<form onSubmit={verify}>
			<p>Successfully logged in as <code>{clientState.user_id}</code></p>
			{content}
		</form>
		{error && <div className="error">
			{error}
		</div>}
	</main>
}
