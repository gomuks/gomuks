import { app, BrowserWindow, Menu, nativeImage, shell, Tray } from "electron"
import path from "node:path"
import { ChildProcess, spawn } from "node:child_process"
import { randomBytes } from "node:crypto"
import started from "electron-squirrel-startup"

if (started) {
	app.quit()
	process.exit(0)
}

if (process.defaultApp) {
	if (process.argv.length >= 2) {
		app.setAsDefaultProtocolClient("matrix", process.execPath, [path.resolve(process.argv[1])])
	}
} else {
	app.setAsDefaultProtocolClient("matrix")
}

if (!app.requestSingleInstanceLock()) {
	app.quit()
	process.exit(0)
}

function backendBinaryPath() {
	const binaryName = "gomuks" + (process.platform === "win32" ? ".exe" : "")
	if (app.isPackaged) {
		return path.join(process.resourcesPath, binaryName)
	}
	return binaryName
}

const externalAddress = process.env.GOMUKS_DESKTOP_BACKEND_ADDR
const externalUsername = process.env.GOMUKS_DESKTOP_BACKEND_USERNAME
const externalPassword = process.env.GOMUKS_DESKTOP_BACKEND_PASSWORD
const desktopKey = randomBytes(32).toString("hex")
let backendProc: ChildProcess | null = null
let serverAddrPromise: Promise<string> | null = null

function startBackend() {
	if (externalAddress) {
		return
	}
	const binaryPath = backendBinaryPath()
	console.log("Spawning", binaryPath, "--desktop")
	backendProc = spawn(binaryPath, ["--desktop"], {
		stdio: ["ignore", "pipe", "inherit"],
		windowsHide: true,
		env: {
			GOMUKS_ROOT: path.join(app.getPath("sessionData"), "backend"),
			...process.env,
			GOMUKS_DESKTOP_KEY: desktopKey,
		},
	})
	backendProc.on("exit", code => {
		backendProc = null
		if (code !== 0) {
			console.error(`Backend exited with code ${code}`)
		} else {
			console.log("Backend exited normally")
		}
		app.quit()
	})
	serverAddrPromise = new Promise((resolve, reject) => {
		const stdout = backendProc?.stdout
		if (!stdout) {
			reject(new Error("Failed to start backend: no stdout"))
			return
		}
		let exitHandler = (code: number | null) => {
			reject(new Error(`Backend exited with status ${code}`))
		}
		let handler = (output: string) => {
			try {
				const data = JSON.parse(output)
				if (data.started === true && data.address) {
					console.info("Got status from backend:", data)
					stdout.off("data", handler)
					backendProc?.off("exit", exitHandler)
					resolve(data.address)
				} else {
					console.warn("Unexpected backend output:", data)
				}
			} catch (err) {
				console.error("Failed to parse backend output:", output.toString())
			}
		}
		stdout.on("data", handler)
		backendProc?.on("exit", exitHandler)
	})
}

let triedToQuit = false

const onClickQuit = () => {
	if (backendProc) {
		console.log("Sending", triedToQuit ? "SIGKILL" : "SIGTERM", "to backend")
		backendProc.kill(triedToQuit ? "SIGKILL" : "SIGTERM")
		triedToQuit = true
	} else {
		app.quit()
	}
}

const onFocus = () => {
	if (BrowserWindow.getAllWindows().length === 0) {
		createWindow()
	} else {
		if (activeMainWindow.isMinimized()) {
			activeMainWindow.restore()
		}
		activeMainWindow.focus()
	}
}

let tray: Tray | null = null

function createTrayIcon() {
	const trayIconPath = path.join(app.isPackaged ? process.resourcesPath : app.getAppPath(), "icon.png")
	tray = new Tray(nativeImage.createFromPath(trayIconPath))
	tray.setContextMenu(Menu.buildFromTemplate([
		{
			label: "Open",
			click: onFocus,
		},
		{
			label: "Quit",
			click: onClickQuit,
		},
	]))
}

let activeMainWindow: BrowserWindow

function createWindow() {
	const mainWindow = new BrowserWindow({
		width: 1280,
		height: 720,
		autoHideMenuBar: true,
		webPreferences: {
			preload: path.join(__dirname, "preload.js"),
		},
	})
	activeMainWindow = mainWindow

	mainWindow.webContents.setWindowOpenHandler(details => {
		console.log("Opening", details.url, "externally")
		shell.openExternal(details.url)
		return { action: "deny" }
	})

	let serverURL: string | null = null
	mainWindow.webContents.on("login", (event, authenticationResponseDetails, authInfo, callback) => {
		event.preventDefault()
		if (serverURL && authenticationResponseDetails.url.startsWith(`${serverURL}/_gomuks/auth`)) {
			if (externalAddress) {
				callback(externalUsername, externalPassword)
			} else {
				callback("desktop-key", desktopKey)
			}
		} else {
			console.warn("Unexpected auth request from", authenticationResponseDetails.url)
			callback()
		}
	})

	if (externalAddress) {
		mainWindow.loadURL(externalAddress)
	} else if (serverAddrPromise) {
		serverAddrPromise.then(addr => {
			serverURL = `http://${addr}`
			mainWindow.loadURL(serverURL)
		})
	} else {
		throw new Error("Server address not available")
	}
	if (process.env.NODE_ENV === "development") {
		mainWindow.webContents.openDevTools()
	}
}

function handleMatrixURI(uri: string) {
	console.log("Handling external matrix URI", uri)
	activeMainWindow?.webContents.send("open-matrix-uri", uri)
}

app.on("window-all-closed", () => {
	if (!backendProc) {
		app.quit()
	}
})

app.on("before-quit", evt => {
	if (backendProc) {
		evt.preventDefault()
		onClickQuit()
	}
})

app.on("activate", onFocus)

app.on("second-instance", (event, commandLine, workingDirectory) => {
	console.log("Got second instance with", commandLine)
	onFocus()

	const uri = commandLine.pop()
	if (uri?.startsWith("matrix:")) {
		handleMatrixURI(uri)
	}
})

app.on("open-url", (event, url) => {
	handleMatrixURI(url)
})

app.whenReady().then(() => {
	startBackend()
	createWindow()
	createTrayIcon()
})
