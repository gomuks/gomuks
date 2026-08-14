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
import type { DeviceID, UserID } from "./mxtypes.ts"

export type Scope =
	"openid"
	| "email"
	| "urn:matrix:client:api:*"
	| `urn:matrix:client:device:${DeviceID}`

export type GrantType =
	"authorization_code" | "refresh_token" | "client_credentials"
	| "urn:ietf:params:oauth:grant-type:device_code"

export type ResponseType = "code"

export type AuthMethod = "none"

// TODO proper type
//eslint-disable-next-line @typescript-eslint/no-explicit-any
export type OAuthServerMetadata = Record<string, any>

export interface OAuthClientMetadata {
	client_id: string

	application_type: "web" | "native"

	client_name?: string
	client_uri?: string
	logo_uri?: string
	policy_uri?: string
	tos_uri?: string

	grant_types?: GrantType[]
	redirect_uris?: string[]
	response_types?: ResponseType[]
	token_endpoint_auth_method?: AuthMethod
}

export type OAuthClientMetadataRequest = Omit<OAuthClientMetadata, "client_id">

export interface OAuthGetAuthorizationURLParams {
	homeserver_url: string
	redirect_uri: string
	scopes: Scope[]
	user_id_hint?: UserID
	client_id?: string
	response_mode: "query" | "fragment"
}

export interface OAuthAuthorizationState {
	state: string
	code_verifier: string
	url: string
}

export interface OAuthExchangeTokenParams {
	homeserver_url: string
	code_verifier: string
	code: string
	redirect_uri: string
	client_id: string
}

export interface OAuthGenerateDeviceCodeParams {
	homeserver_url: string
	scopes: Scope[]
	client_id?: string
	user_id_hint?: UserID
}

export interface OAuthDeviceCodeResponse {
	device_code: string
	user_code: string
	verification_uri: string
	verification_uri_complete?: string
	expires_in: number
	interval?: number
}
