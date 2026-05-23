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
import { type BackendConfig } from "./webview.ts"
import path from "node:path"
import { app, safeStorage } from "electron"

export interface GomuksConfig {
	backends: BackendConfig[]
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

export async function loadConfig(): Promise<GomuksConfig> {
	if (!await fileExists(configPath)) {
		console.log("Generating new default config")
		const config: GomuksConfig = {
			backends: [{
				type: "embedded",
				name: "backend",
				displayname: "Default Profile",
			}]
		}
		await saveConfig(config)
		return config
	}
	console.log("Reading config from", configPath)
	const file = await fs.readFile(configPath, { encoding: "utf8" })
	const parsed = JSON.parse(file)
	let doSave = false
	const config = {
		backends: await Promise.all(parsed.backends.map(async (backend: any) => {
			if (typeof backend.name !== "string") {
				throw new Error("Invalid backend config: name must be a string")
			}
			if (backend.type === "embedded") {
				return {
					type: "embedded",
					name: backend.name,
					displayname: backend.displayname,
					icon: backend.icon,
					env: backend.env,
				} as BackendConfig
			} else if (backend.type === "remote") {
				if (typeof backend.address !== "string" || typeof backend.username !== "string" || (typeof backend.password !== "string" && typeof backend.password_encrypted !== "string")) {
					throw new Error("Invalid backend config: remote backends must have address, username and password")
				}
				let password = backend.password
				if (backend.password_encrypted) {
					const passwd = await safeStorage.decryptStringAsync(Buffer.from(backend.password_encrypted, "base64"))
					if (passwd.shouldReEncrypt) {
						doSave = true
					}
					password = passwd.result
				}
				return {
					type: "remote",
					name: backend.name,
					displayname: backend.displayname,
					icon: backend.icon,
					address: backend.address,
					username: backend.username,
					password,
				} as BackendConfig
			} else {
				throw new Error(`Invalid backend config: unknown type ${backend.type}`)
			}
		}))
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
			}))
		}, null, 2),
		{ encoding: "utf8", mode: 0o600 },
	)
}
