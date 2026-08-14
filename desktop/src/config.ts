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
import fs from "node:fs/promises"
import path from "node:path"
import { app, safeStorage } from "electron"
import type { BackendConfig, TabInfoUpdate } from "./webview.ts"

export interface GomuksConfig {
	backends: BackendConfig[]
	disable_tray?: boolean
}

async function fileExists(path: string): Promise<boolean> {
	try {
		await fs.access(path, fs.constants.F_OK)
		return true
	} catch {
		return false
	}
}

const configPath = path.join(app.getPath("userData"), "gomuks-desktop.json")
const validTabIDRegex = /^[a-zA-Z0-9_-]{1,32}$/

export function isValidTabID(id: unknown): id is string {
	return typeof id === "string" && !!id && validTabIDRegex.test(id)
}

export async function loadConfig(): Promise<GomuksConfig> {
	if (!await fileExists(configPath)) {
		console.log("Generating new default config")
		const config: GomuksConfig = {
			backends: [{
				type: "embedded",
				id: "backend",
				displayname: "Default Profile",
				disable_notifications: false,
			}],
			disable_tray: false,
		}
		await saveConfig(config)
		return config
	}
	console.log("Reading config from", configPath)
	const file = await fs.readFile(configPath, { encoding: "utf8" })
	const parsed = JSON.parse(file)
	let doSave = false
	const config = {
		//eslint-disable-next-line @typescript-eslint/no-explicit-any
		backends: await Promise.all(parsed.backends.map(async (backend: any) => {
			const backendID = backend.id ?? backend.name
			if (!isValidTabID(backendID)) {
				throw new Error(`Invalid backend config: name must match ${validTabIDRegex}`)
			}
			if (backend.type === "embedded") {
				return {
					type: "embedded",
					id: backendID,
					displayname: backend.displayname ?? backendID,
					icon: backend.icon,
					env: backend.env,
					disable_notifications: Boolean(backend.disable_notifications),
				} as BackendConfig
			} else if (backend.type === "remote") {
				if (
					typeof backend.address !== "string"
					|| typeof backend.username !== "string"
					|| (typeof backend.password !== "string" && typeof backend.password_encrypted !== "string")
				) {
					throw new Error("Invalid backend config: remote backends must have address, username and password")
				}
				let password = backend.password
				if (backend.password_encrypted) {
					const passwd =
						await safeStorage.decryptStringAsync(Buffer.from(backend.password_encrypted, "base64"))
					if (passwd.shouldReEncrypt) {
						doSave = true
					}
					password = passwd.result
				}
				return {
					type: "remote",
					id: backendID,
					displayname: backend.displayname ?? backendID,
					icon: backend.icon,
					address: backend.address,
					username: backend.username,
					password,
					disable_notifications: Boolean(backend.disable_notifications),
				} as BackendConfig
			} else {
				throw new Error(`Invalid backend config: unknown type ${backend.type}`)
			}
		})),
		disable_tray: Boolean(parsed.disable_tray),
	}
	if (doSave) {
		await saveConfig(config)
	}
	return config
}

export async function saveConfig(config: GomuksConfig) {
	console.log("Saving config to", configPath)
	const canEncrypt = await safeStorage.isAsyncEncryptionAvailable()
	if (!canEncrypt) {
		console.warn("Config encryption not available")
	}
	await fs.writeFile(
		configPath,
		JSON.stringify({
			backends: await Promise.all(config.backends.map(async backend => {
				if (backend.type === "remote" && canEncrypt) {
					return {
						...backend,
						password: undefined,
						password_encrypted: (await safeStorage.encryptStringAsync(backend.password)).toString("base64"),
					}
				}
				return backend
			})),
		}, null, 2),
		{ encoding: "utf8", mode: 0o600 },
	)
}

function isValidEnv(env: unknown): env is Record<string, string> {
	if (typeof env !== "object" || env === null) {
		return false
	}
	for (const [key, value] of Object.entries(env)) {
		if (typeof key !== "string" || typeof value !== "string") {
			return false
		}
	}
	return true
}

export function configNeedsRecreate(oldConfig: BackendConfig | undefined, newConfig: BackendConfig): boolean {
	if (!oldConfig || oldConfig.id !== newConfig.id) {
		return true
	}
	if (oldConfig.type === "embedded" && newConfig.type === "embedded") {
		return JSON.stringify(oldConfig.env ?? {}) !== JSON.stringify(newConfig.env ?? {})
	} else if (oldConfig.type === "remote" && newConfig.type === "remote") {
		return oldConfig.address !== newConfig.address
			|| oldConfig.username !== newConfig.username
			|| oldConfig.password !== newConfig.password
	}
	return false
}

export function tabInfoToConfig(
	tab: TabInfoUpdate,
	oldConfig?: BackendConfig,
): BackendConfig {
	if (!isValidTabID(tab.id)) {
		throw new Error("Invalid tab ID")
	} else if (typeof tab.displayname !== "string" || !tab.displayname) {
		throw new Error("Invalid tab displayname")
	} else if (tab.icon !== undefined && typeof tab.icon !== "string") {
		throw new Error("Invalid tab icon")
	} else if (typeof tab.disable_notifications !== "boolean") {
		throw new Error("Invalid tab disable_notifications")
	}
	const baseTab = {
		disable_notifications: tab.disable_notifications,
		id: tab.id,
		displayname: tab.displayname,
		icon: tab.icon,
	}
	if (tab.type === "embedded") {
		if (tab.env !== undefined && !isValidEnv(tab.env)) {
			throw new Error("Invalid tab env")
		}
		return {
			type: "embedded",
			...baseTab,
			env: tab.env,
		}
	} else if (tab.type === "remote") {
		if (tab.password === undefined) {
			if (oldConfig?.type === "remote") {
				tab.password = oldConfig.password
			}
		}
		if (typeof tab.address !== "string" || !tab.address) {
			throw new Error("Invalid tab address")
		} else if (typeof tab.username !== "string") {
			throw new Error("Invalid tab username")
		} else if (typeof tab.password !== "string") {
			throw new Error("Invalid tab password")
		}
		return {
			type: "remote",
			...baseTab,
			address: tab.address,
			username: tab.username,
			password: tab.password,
		}
	} else {
		throw new Error("Invalid tab type")
	}
}
