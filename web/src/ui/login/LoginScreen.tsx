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
import React, { use, useCallback, useEffect, useRef, useState } from "react"
import type Client from "@/api/client.ts"
import { hasTabs } from "@/api/tabs.ts"
import type {
	ClientState,
	OAuthClientMetadataRequest,
	OAuthDeviceCodeResponse,
	OAuthServerMetadata,
} from "@/api/types"
import { NestableModalContext } from "../modal"
import BackendManager from "../settings/BackendManager.tsx"
import BeeperLogin from "./BeeperLogin.tsx"
import CheckIcon from "@/icons/check.svg?react"
import CopyIcon from "@/icons/copy.svg?react"
import "./LoginScreen.css"

export interface LoginScreenProps {
	client: Client
	clientState: ClientState
}

const beeperServerRegex = /^https:\/\/matrix\.(beeper(?:-dev|-staging)?\.com)$/

const generateDeviceID = () => {
	let deviceID = Math.random().toString(36).slice(2, 12).toUpperCase()
	if (deviceID.length < 10) {
		deviceID += Math.random().toString(36).slice(2, 12 - deviceID.length).toUpperCase()
	}
	return deviceID
}

const standardClientRegistrationParams: OAuthClientMetadataRequest = {
	application_type: "native",
	client_name: "gomuks web",
	client_uri: "https://gomuks.app/",
	logo_uri: "https://gomuks.app/favicon.png",
	grant_types: ["refresh_token", "urn:ietf:params:oauth:grant-type:device_code"],
	token_endpoint_auth_method: "none",
}

const clientURI = window.location.origin + window.location.pathname
const isLocalhost = window.location.hostname === "localhost"
const isSupportedRedirectURI = window.location.protocol === "https:" || isLocalhost

const redirectClientRegistrationParams: OAuthClientMetadataRequest = {
	application_type: "native",
	client_name: "gomuks web",
	client_uri: isLocalhost ? "https://gomuks.app" : clientURI,
	logo_uri: isLocalhost ? "https://gomuks.app/favicon.png" : (clientURI + "gomuks.png"),
	grant_types: ["refresh_token", "authorization_code", "urn:ietf:params:oauth:grant-type:device_code"],
	token_endpoint_auth_method: "none",
	response_types: ["code"],
	redirect_uris: [isLocalhost ? "http://localhost" + window.location.pathname : clientURI],
}

export const LoginScreen = ({ client }: LoginScreenProps) => {
	const [username, setUsername] = useState("")
	const [password, setPassword] = useState("")
	const [homeserverURL, setHomeserverURL] = useState("")
	const [loginFlows, setLoginFlows] = useState<string[] | null>(null)
	const [oauthServerMeta, setOAuthServerMeta] = useState<OAuthServerMetadata | null>(null)
	const [deviceCode, setDeviceCode] = useState<OAuthDeviceCodeResponse | null>(null)
	const [copySuccess, setCopySuccess] = useState(false)
	const cancelDeviceCodePoll = useRef<(() => void) | null>(null)
	const skipServerResolution = useRef(false)
	const [loading, setLoading] = useState<boolean>(false)
	const [error, setError] = useState("")
	const openNestableModal = use(NestableModalContext)

	const loginSSOAsync = async () => {
		const clientMeta = await client.rpc.oauthRegisterClient(homeserverURL, standardClientRegistrationParams)
		const deviceID = generateDeviceID()
		const resp = await client.rpc.oauthGenerateDeviceCode({
			homeserver_url: homeserverURL,
			scopes: ["urn:matrix:client:api:*", `urn:matrix:client:device:${deviceID}`],
			user_id_hint: username,
			client_id: clientMeta.client_id,
		})
		setDeviceCode(resp)
		localStorage.pendingDeviceCodeLogin = JSON.stringify({
			homeserver_url: homeserverURL,
			username: username,
			client_id: clientMeta.client_id,
			code_resp: resp,
			login_flows: loginFlows,
			server_metadata: oauthServerMeta,
		})
		await loginSSOPollDeviceCode(homeserverURL, clientMeta.client_id, resp.device_code, resp.interval || 5)
	}
	const loginSSORedirect = async () => {
		const clientMeta = await client.rpc.oauthRegisterClient(homeserverURL, redirectClientRegistrationParams)
		const deviceID = generateDeviceID()
		const resp = await client.rpc.oauthGetAuthorizationURL({
			homeserver_url: homeserverURL,
			scopes: ["urn:matrix:client:api:*", `urn:matrix:client:device:${deviceID}`],
			user_id_hint: username,
			client_id: clientMeta.client_id,
			redirect_uri: clientURI,
			response_mode: "fragment",
		})
		localStorage.pendingAuthorizationCodeLogin = JSON.stringify({
			state: resp.state,
			code_verifier: resp.code_verifier,
			redirect_uri: clientURI,
			homeserver_url: homeserverURL,
			client_id: clientMeta.client_id,
		})
		if (window.gomuksWebWasm) {
			client.rpc.stop()
		}
		window.location.href = resp.url
	}
	const loginSSOPollDeviceCode = useCallback((
		homeserverURL: string, clientID: string, code: string, interval: number,
	) => new Promise<void>((resolve, reject) => {
		let cancelled = false
		let pollTimeout: ReturnType<typeof setTimeout>
		const cancel = () => {
			cancelled = true
			clearTimeout(pollTimeout)
			delete localStorage.pendingDeviceCodeLogin
			reject(new Error("Login cancelled"))
		}
		const pollFunc = () => client.rpc.oauthPollDeviceCode(homeserverURL, code, clientID).then(() => {
			console.log("OAuth device code login successful")
			if (!cancelled) {
				resolve()
			}
		}, err => {
			if (cancelled) {
				return
			}
			const errStr = err.toString()
			if (errStr.includes("authorization_pending")) {
				pollTimeout = setTimeout(pollFunc, interval*1000)
			} else if (errStr.includes("slow_down")) {
				interval += 5
				console.log(`Increasing polling interval to ${interval} seconds due to ${err}`)
				pollTimeout = setTimeout(pollFunc, interval*1000)
			} else {
				reject(err)
				cancel()
			}
		})
		pollTimeout = setTimeout(pollFunc, 0)
		cancelDeviceCodePoll.current?.()
		cancelDeviceCodePoll.current = cancel
	}), [client])

	const loginAnySSOAsync = () => {
		if (supportsCodeSSO) {
			return loginSSOAsync()
		} else if (supportsRedirectSSO) {
			return loginSSORedirect()
		} else {
			return Promise.reject(new Error("No supported SSO method"))
		}
	}

	const loginSSO = () => {
		setLoading(true)
		loginAnySSOAsync()
			.catch(err => setError(err.toString()))
			.finally(() => {
				setLoading(false)
				setDeviceCode(null)
			})
	}

	const login = (evt: React.SubmitEvent) => {
		evt.preventDefault()
		if (!loginFlows) {
			// do nothing
		} else if (!loginFlows.includes("m.login.password")) {
			loginSSO()
		} else {
			setLoading(true)
			client.rpc.login(homeserverURL, username, password).then(
				() => {
					client.passwordCache = password
				},
				err => setError(err.toString()),
			).finally(() => setLoading(false))
		}
	}

	const resolveLoginFlows = useCallback((serverURL: string) => {
		client.rpc.getLoginFlows(serverURL).then(
			resp => {
				setLoginFlows(resp.flows?.map(flow => flow.type) ?? [])
				setOAuthServerMeta(resp.oauth ?? null)
				setError("")
			},
			err => setError(`Failed to get login flows: ${err}`),
		)
	}, [client])
	const resolveHomeserver = useCallback(() => {
		client.rpc.discoverHomeserver(username).then(
			resp => {
				const url = resp["m.homeserver"].base_url
				setLoginFlows(null)
				setOAuthServerMeta(null)
				setHomeserverURL(url)
				resolveLoginFlows(url)
			},
			err => setError(`Failed to resolve homeserver: ${err}`),
		)
	}, [client, username, resolveLoginFlows])

	useEffect(() => {
		if (localStorage.pendingDeviceCodeLogin) {
			const data = JSON.parse(localStorage.pendingDeviceCodeLogin)
			skipServerResolution.current = true
			setDeviceCode(data.code_resp)
			setLoginFlows(data.login_flows)
			setOAuthServerMeta(data.server_metadata)
			setHomeserverURL(data.homeserver_url)
			setUsername(data.username)
			setLoading(true)
			loginSSOPollDeviceCode(
				data.homeserver_url, data.client_id, data.code_resp.device_code, data.code_resp.interval,
			).catch(err => {
				setError(err.toString())
			}).finally(() => {
				skipServerResolution.current = false
				setLoading(false)
				setDeviceCode(null)
			})
		} else if (localStorage.pendingAuthorizationCodeLogin && window.location.hash) {
			const cache = JSON.parse(localStorage.pendingAuthorizationCodeLogin)
			const params = new URLSearchParams(window.location.hash.slice(1))
			const code = params.get("code")
			if (params.get("state") === cache.state && code) {
				setLoading(true)
				client.rpc.oauthExchangeToken({ ...cache, code }).then(
					() => {
						console.log("OAuth authorization code login successful")
						delete localStorage.pendingAuthorizationCodeLogin
						const newURL = new URL(window.location.href)
						newURL.hash = ""
						history.replaceState({}, "", newURL.toString())
					},
					err => {
						console.error("OAuth authorization code login failed", err)
						setError(err.toString())
					},
				).finally(() => setLoading(false))
			}
		}
	}, [loginSSOPollDeviceCode, client])
	useEffect(() => {
		if (
			!username.startsWith("@")
			|| !username.includes(":")
			|| !username.includes(".")
			|| skipServerResolution.current
		) {
			return
		}
		const timeout = setTimeout(resolveHomeserver, 500)
		return () => {
			clearTimeout(timeout)
		}
	}, [username, resolveHomeserver])
	useEffect(() => {
		if (loginFlows !== null || loginFlows === "resolving" || !homeserverURL) {
			return
		}
		const timeout = setTimeout(() => resolveLoginFlows(homeserverURL), 500)
		return () => {
			clearTimeout(timeout)
		}
	}, [homeserverURL, loginFlows, resolveLoginFlows])
	const onChangeHomeserverURL = (evt: React.ChangeEvent<HTMLInputElement>) => {
		setLoginFlows(null)
		setHomeserverURL(evt.target.value)
	}
	const copyToClipboard = (evt: React.MouseEvent) => {
		evt.stopPropagation()
		evt.preventDefault()
		const code = evt.currentTarget.getAttribute("data-code")
		navigator.clipboard.writeText(code!).then(
			() => {
				setCopySuccess(true)
				setTimeout(() => setCopySuccess(false), 2000)
			},
			err => console.error("Failed to copy to clipboard", err),
		)
	}
	const openBackendManager = () => {
		openNestableModal({
			boxed: true,
			dimmed: true,
			boxClass: "backend-manager-modal",
			content: <BackendManager />,
		})
	}

	const supportsPassword = loginFlows?.includes("m.login.password")
	const beeperDomain = homeserverURL.match(beeperServerRegex)?.[1]
	const supportsCodeSSO = !!oauthServerMeta?.device_authorization_endpoint
	const supportsRedirectSSO = !!oauthServerMeta?.authorization_endpoint
		// Redirects are a pain on wrapped apps, so don't allow it there
		&& !window.gomuksDesktop && !window.gomuksAndroid && isSupportedRedirectURI
	const supportsAnySSO = supportsCodeSSO || supportsRedirectSSO
	return <main className="matrix-login">
		<h1>gomuks web</h1>
		<form onSubmit={login}>
			<input
				type="text"
				id="mxlogin-username"
				placeholder="User ID (@user:example.com)"
				value={username}
				onChange={evt => setUsername(evt.target.value)}
				disabled={loading}
				autoComplete="username"
			/>
			<input
				type="text"
				id="mxlogin-homeserver-url"
				placeholder="Homeserver URL (will autofill)"
				value={homeserverURL}
				onChange={onChangeHomeserverURL}
				disabled={loading}
				autoComplete="url"
			/>
			{supportsPassword && <input
				type="password"
				id="mxlogin-password"
				placeholder="Password"
				value={password}
				onChange={evt => setPassword(evt.target.value)}
				disabled={loading}
				autoComplete="current-password"
			/>}
			<div className="buttons">
				{supportsAnySSO && !deviceCode && <button
					className="mx-login-button primary-color-button"
					type={supportsPassword ? "button" : "submit"}
					disabled={loading}
					onClick={supportsPassword ? loginSSO : undefined}
				>Login with OAuth</button>}
				{supportsPassword && <button
					className="mx-login-button primary-color-button"
					type="submit"
					disabled={loading}
				>Login{supportsAnySSO || beeperDomain ? " with password" : ""}</button>}
				{loginFlows && !supportsAnySSO && !supportsRedirectSSO && !supportsPassword && !beeperDomain ? <button
					className="mx-login-button primary-color-button"
					type="button"
					disabled
				>No supported login methods</button> : null}
			</div>
			{error && <div className="error">
				{error}
			</div>}
		</form>

		{deviceCode && <div className="oauth-device-code">
			{deviceCode.verification_uri_complete ? <>
				<div className="instructions">
					Click the code below to log in:
				</div>
				<a target="_blank" className="device-code" href={deviceCode.verification_uri_complete}>
					{deviceCode.user_code}
					<button onClick={copyToClipboard} data-code={deviceCode.user_code}>
						{copySuccess ? <CheckIcon /> : <CopyIcon />}
					</button>
				</a>
				<div className="instructions sub">
					(or open <a target="_blank" href={deviceCode.verification_uri}>{deviceCode.verification_uri}</a> and
					enter it there)
				</div>
			</> : <>
				<div className="instructions">
					Open <a target="_blank" href={deviceCode.verification_uri}>{deviceCode.verification_uri}</a> and
					enter the code below:
				</div>
				<div className="device-code">
					{deviceCode.user_code}
					<button onClick={copyToClipboard} data-code={deviceCode.user_code}>
						{copySuccess ? <CheckIcon /> : <CopyIcon />}
					</button>
				</div>
			</>}
			<button
				className="mx-login-button"
				onClick={() => cancelDeviceCodePoll.current?.()}
			>Cancel login</button>
		</div>}

		{beeperDomain && <>
			<hr/>
			<BeeperLogin domain={beeperDomain} client={client}/>
		</>}

		{hasTabs() && <>
			<hr/>
			<button onClick={openBackendManager} className="mx-login-button">Open backend manager</button>
		</>}
	</main>
}
