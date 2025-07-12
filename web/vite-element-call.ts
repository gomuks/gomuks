import fs from "fs"
import path from "path"
import type { Plugin } from "vite"

function extensionToMime(ext: string): string {
	switch (ext) {
	case ".html":
		return "text/html"
	case ".js":
		return "application/javascript"
	case ".css":
		return "text/css"
	case ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

const elementCallPlugin: Plugin = {
	name: "copy-element-call",
	configureServer(server) {
		server.middlewares.use("/element-call-embedded", (req, res, next) => {
			const filePath = req.url
				?.replace("/element-call-embedded", "")
				.replace(/\?.+/, "") || "/index.html"
			const fullPath = path.join("./node_modules/@element-hq/element-call-embedded/dist", filePath)

			try {
				if (fs.statSync(fullPath).isFile()) {
					res.setHeader("Content-Type", extensionToMime(path.extname(filePath)))
					res.end(fs.readFileSync(fullPath))
				} else {
					next()
				}
			} catch {
				next()
			}
		})
	},
	writeBundle() {
		const srcDir = "./node_modules/@element-hq/element-call-embedded/dist"
		const destDir = "./dist/element-call-embedded"

		const copyDir = (src: string, dest: string) => {
			fs.mkdirSync(dest, { recursive: true })
			for (const file of fs.readdirSync(src)) {
				if (
					file.endsWith(".map")
					|| file.endsWith(".woff")
					|| file.endsWith(".mp3")
					|| file.startsWith("matrix_sdk_crypto")
					|| file.startsWith("matrix-sdk-crypto")
				) {
					continue
				}
				const srcPath = path.join(src, file)
				const destPath = path.join(dest, file)
				if (fs.statSync(srcPath).isDirectory()) {
					copyDir(srcPath, destPath)
				} else {
					fs.copyFileSync(srcPath, destPath)
				}
			}
		}

		copyDir(srcDir, destDir)
	},
}

export default elementCallPlugin
