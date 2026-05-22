import data from "./build-info.json"

export interface BuildInfo {
	ci?: boolean
	commit?: string
	tag?: string
	version?: string
	updateChannel: "stable" | "nightly"
	builtAt?: string
}

export default data as BuildInfo
