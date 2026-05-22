import type { ForgeConfig } from "@electron-forge/shared-types"
import { MakerSquirrel } from "@electron-forge/maker-squirrel"
import { MakerDMG } from "@electron-forge/maker-dmg"
import { MakerZIP } from "@electron-forge/maker-zip"
import { MakerDeb } from "@electron-forge/maker-deb"
import { VitePlugin } from "@electron-forge/plugin-vite"
import { FusesPlugin } from "@electron-forge/plugin-fuses"
import { FuseV1Options, FuseVersion } from "@electron/fuses"
import path from "node:path"
import fs from "node:fs/promises"
import pkg from "./package.json"
import type { BuildInfo } from "./src/build-info.ts"

const commit = process.env.CI_COMMIT_SHA
const tag = process.env.CI_COMMIT_TAG
let appVersion: string | undefined = undefined
let debVersion

if (commit && !tag) {
	debVersion = `${pkg.version}~git${commit.slice(0, 7)}`
}

const ci = process.env.CI === "true"
if (ci && !process.env.GIT_DESCRIBE) {
	throw new Error("Missing GIT_DESCRIBE in CI")
}

if (!tag && process.env.GIT_DESCRIBE) {
	const descRegex = /^v0\.(\d{2})(\d{2})\.(\d+)(?:-(\d+)-g([0-9a-f]+))?$/
	const [, year, month, minorVersion, commitCount, commitHash] = process.env.GIT_DESCRIBE.match(descRegex) ?? []
	if (commitHash) {
		const paddedMinorVersion = (+minorVersion) * 1000 + (+commitCount)
		appVersion = `${year}.${month}.${paddedMinorVersion}`
		debVersion = `${year}.${month}.${minorVersion}-${commitCount}~git${commitHash.slice(0, 7)}`
		console.log("Calculated app version:", appVersion, "/ debian:", debVersion)
	} else {
		throw new Error("Failed to parse GIT_DESCRIBE")
	}
}
const updateChannel = tag ? "stable" : "nightly"

const config: ForgeConfig = {
	packagerConfig: {
		asar: true,
		protocols: [
			{
				name: "matrix",
				schemes: ["matrix"],
			},
		],
		appVersion,
		icon: "icon",
		osxSign: {},
		osxNotarize: process.env.APPLE_API_KEY_PATH ? {
			appleApiKey: process.env.APPLE_API_KEY_PATH,
			appleApiKeyId: process.env.APPLE_API_KEY_ID!,
			appleApiIssuer: process.env.APPLE_API_ISSUER!,
		} : undefined,
		appBundleId: "app.gomuks.desktop",
		appCategoryType: "public.app-category.social-networking",
		extraResource: ["tray@2x.png", "trayTemplate@2x.png"],
	},
	hooks: {
		readPackageJson: async (_forgeConfig, packageJSON) => {
			if (appVersion) {
				packageJSON.version = appVersion
			}
			return packageJSON
		},
		generateAssets: async () => {
			const buildInfo: BuildInfo = {
				ci,
				commit,
				tag,
				version: appVersion ?? pkg.version,
				updateChannel,
				builtAt: new Date().toISOString(),
			}
			try {
				await fs.writeFile(path.join(__dirname, "src", "build-info.json"), JSON.stringify(buildInfo))
			} catch (err) {
				console.error("Failed to write build-info.json", err)
				throw err
			}
		},
		packageAfterCopy: async (_forgeConfig, buildPath, _electronVersion, platform, _arch) => {
			const binaryName = platform === "win32" ? "gomuks.exe" : "gomuks"
			const resourcesDir = path.resolve(buildPath, "..")
			const dest = path.join(resourcesDir, binaryName)

			try {
				await fs.copyFile(path.join(__dirname, "..", binaryName), dest)
				if (platform === "darwin") {
					const dylibDest = path.join(resourcesDir, "libolm.3.dylib")
					await fs.copyFile(path.join(__dirname, "..", "libolm.3.dylib"), dylibDest)
					await fs.chmod(dylibDest, 0o644)
				}
				if (platform !== "win32") {
					await fs.chmod(dest, 0o755)
				}
			} catch (err) {
				console.error(`Failed to copy binary:`, err)
				throw err
			}
		},
	},
	rebuildConfig: {},
	makers: [
		new MakerSquirrel(_arch => ({
			// remoteReleases: `https://update.gomuks.app/desktop-${updateChannel}/win32/${arch}`,
		})),
		new MakerDMG({}),
		new MakerZIP(arch => ({
			macUpdateManifestBaseUrl: `https://update.gomuks.app/desktop-${updateChannel}/darwin/${arch}`,
			macUpdateReleaseNotes: tag ? undefined : process.env.CI_COMMIT_MESSAGE,
		}), ["darwin", "linux"]),
		new MakerDeb({
			options: {
				version: debVersion,
				section: "net",
				recommends: ["ffmpeg"],
				mimeType: ["x-scheme-handler/matrix"],
				icon: "icon.png",
			},
		}),
	],
	plugins: [
		new VitePlugin({
			// `build` can specify multiple entry builds, which can be Main process, Preload scripts, Worker process, etc.
			// If you are familiar with Vite configuration, it will look really familiar.
			build: [
				{
					// `entry` is just an alias for `build.lib.entry` in the corresponding file of `config`.
					entry: "src/main.ts",
					config: "vite.main.config.ts",
					target: "main",
				},
				{
					entry: "src/preload.ts",
					config: "vite.preload.config.ts",
					target: "preload",
				},
			],
			renderer: [
				{
					name: "main_window",
					config: "vite.renderer.config.ts",
				},
			],
		}),
		// Fuses are used to enable/disable various Electron functionality
		// at package time, before code signing the application
		new FusesPlugin({
			version: FuseVersion.V1,
			[FuseV1Options.RunAsNode]: false,
			[FuseV1Options.EnableCookieEncryption]: true,
			[FuseV1Options.EnableNodeOptionsEnvironmentVariable]: false,
			[FuseV1Options.EnableNodeCliInspectArguments]: false,
			[FuseV1Options.EnableEmbeddedAsarIntegrityValidation]: true,
			[FuseV1Options.OnlyLoadAppFromAsar]: true,
		}),
	],
}

export default config
