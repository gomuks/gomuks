import { contextBridge, ipcRenderer } from "electron"

contextBridge.exposeInMainWorld("gomuksDesktop", true)

ipcRenderer.on("open-matrix-uri", (evt, url: string) => {
	if (!url.startsWith("matrix:")) {
		console.warn("Received non-matrix URI from main process:", url)
		return
	}
	console.log("Received matrix: URI from main process:", url)
	location.hash = `#/uri/${encodeURIComponent(url)}`
})

ipcRenderer.on("disable-notifications", () => {
	contextBridge.exposeInMainWorld("gomuksDesktopNotifications", true)
})
