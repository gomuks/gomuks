import { app, BrowserWindow, Menu, nativeImage, Notification, shell, Tray } from "electron"
import electronDl from "electron-dl"
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

interface NotificationUser {
	id: string
	name: string
	avatar?: string
}

interface PushNewMessage {
	desktop_notification: true
	timestamp: number
	event_id: string
	event_rowid: number

	room_id: string
	room_name: string
	room_avatar?: string
	sender: NotificationUser
	self: NotificationUser

	text: string
	image?: string
	mention?: true
	reply?: true
	sound?: true
}

const openNotifications = new Map<string, Map<number, Notification>>()

function getNotifMap(roomID: string) {
	let map = openNotifications.get(roomID)
	if (!map) {
		map = new Map()
		openNotifications.set(roomID, map)
	}
	return map
}

function onPushNotification(data: PushNewMessage) {
	const notif = new Notification({
		body: data.text,
		title: data.sender.name === data.room_name ? data.sender.name : `${data.sender.name} (${data.room_name})`,
		silent: !data.sound,
		// TODO this doesn't support webp
		// icon: data.sender.avatar,
		groupId: data.room_id,
		groupTitle: data.room_name,
	})
	getNotifMap(data.room_id).set(data.event_rowid, notif)
	notif.on("close", () => getNotifMap(data.room_id).delete(data.event_rowid))
	notif.on("click", () => {
		const targetURI = `matrix:roomid/${encodeURIComponent(data.room_id.slice(1))}/e/${encodeURIComponent(data.event_id.slice(1))}`
		console.log("Opening", targetURI, "after notification click")
		onFocus()
		handleMatrixURI(targetURI)
	})
	console.log("Displaying notification for", data.event_id, "in", data.room_id)
	notif.show()
}

function onDismissNotification(roomID: string) {
	const map = openNotifications.get(roomID)
	if (map?.size) {
		console.log("Clearing active notifications in", roomID)
		for (const notif of map.values()) {
			notif.close()
		}
		map.clear()
	}
}

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
			GOMUKS_LOGS_HOME: app.getPath("logs"),
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
					backendProc?.off("exit", exitHandler)
					resolve(data.address)
				} else if (data.desktop_notification) {
					onPushNotification(data.desktop_notification)
				} else if (data.dismiss_notification) {
					onDismissNotification(data.dismiss_notification)
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
	const trayIconPath = path.join(
		app.isPackaged ? process.resourcesPath : app.getAppPath(),
		process.platform === "darwin" ? "trayTemplate@2x.png" : "tray@2x.png",
	)
	tray = new Tray(nativeImage.createFromPath(trayIconPath))
	tray.setContextMenu(Menu.buildFromTemplate([
		{
			label: "Open gomuks",
			click: onFocus,
		},
		{
			label: "Quit gomuks",
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
		if (details.url.startsWith(`${serverURL}/_gomuks/media/`)) {
			console.log("Downloading", details.url)
			mainWindow.webContents.downloadURL(details.url)
		} else {
			console.log("Opening", details.url, "externally")
			shell.openExternal(details.url)
		}
		return { action: "deny" }
	})

	let serverURL: string | null = externalAddress || null
	mainWindow.webContents.on("login", (event, authenticationResponseDetails, authInfo, callback) => {
		event.preventDefault()
		if (serverURL && authenticationResponseDetails.url.startsWith(`${serverURL}/_gomuks/auth`)) {
			if (externalAddress) {
				if (!externalUsername || !externalPassword) {
					console.warn("External backend requires authentication but username or password not set in environment variables")
				}
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
			return mainWindow.loadURL(serverURL)
		})
		mainWindow.webContents.send("disable-notifications")
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

electronDl({
	saveAs: process.platform !== "darwin",
})

app.whenReady().then(() => {
	startBackend()
	createWindow()
	createTrayIcon()
	const lastArg = process.argv[process.argv.length - 1]
	if (lastArg.startsWith("matrix:")) {
		handleMatrixURI(lastArg)
	}
})
