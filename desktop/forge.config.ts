import type { ForgeConfig } from "@electron-forge/shared-types"
import { MakerSquirrel } from "@electron-forge/maker-squirrel"
import { MakerDMG } from "@electron-forge/maker-dmg"
import { MakerDeb } from "@electron-forge/maker-deb"
import { VitePlugin } from "@electron-forge/plugin-vite"
import { FusesPlugin } from "@electron-forge/plugin-fuses"
import { FuseV1Options, FuseVersion } from "@electron/fuses"
import path from "node:path"
import fs from "node:fs/promises"
import pkg from "./package.json" with { type: "json" }

const commit = process.env.CI_COMMIT_SHA
const tag = process.env.CI_COMMIT_TAG

let debVersion

if (commit && !tag) {
	debVersion = `${pkg.version}~git${commit.slice(0, 7)}`
}

const config: ForgeConfig = {
	packagerConfig: {
		asar: true,
		protocols: [
			{
				name: "matrix",
				schemes: ["matrix"],
			},
		],
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
		packageAfterCopy: async (forgeConfig, buildPath, _electronVersion, platform, arch) => {
			const binaryName = platform === "win32" ? "gomuks.exe" : "gomuks"
			const resourcesDir = path.resolve(buildPath, "..")
			const dest = path.join(resourcesDir, binaryName)

			await fs.copyFile(path.join(__dirname, "..", binaryName), dest)
			if (platform === "darwin") {
				await fs.copyFile(path.join(__dirname, "..", "libolm.3.dylib"), path.join(resourcesDir, "libolm.3.dylib"))
			}
			if (platform !== "win32") {
				await fs.chmod(dest, 0o755)
			}
		},
	},
	rebuildConfig: {},
	makers: [
		new MakerSquirrel({}),
		new MakerDMG({}),
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
			renderer: [],
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
