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
import { app, Menu, MenuItemConstructorOptions } from "electron"
import electronDl from "electron-dl"
import path from "node:path"
import started from "electron-squirrel-startup"
import { updateElectronApp, UpdateSourceType } from "update-electron-app"
import { EmbeddedBackend } from "./backend.ts"
import { GomuksWindow } from "./mainwindow.ts"
import { loadConfig } from "./config.ts"
import buildInfo from "./build-info.ts"

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

const mainWindow = new GomuksWindow()

function prepareMenu() {
	const isMac = process.platform === "darwin"
	Menu.setApplicationMenu(Menu.buildFromTemplate([
		...(isMac ? [{ role: "appMenu" }] as MenuItemConstructorOptions[] : []),
		{ role: "fileMenu" },
		{ role: "editMenu" },
		{
			label: "View",
			submenu: [
				{ role: "reload" },
				{ role: "forceReload" },
				{
					// https://github.com/electron/electron/pull/49356 might remove the need to override this
					label: "Toggle Developer Tools",
					accelerator: isMac ? "Alt+Command+I" : "Ctrl+Shift+I",
					click: mainWindow.toggleDevTools,
				},
				{ type: "separator" },
				{ role: "resetZoom" },
				{ role: "zoomIn" },
				{ role: "zoomOut" },
				{ type: "separator" },
				{ role: "togglefullscreen" },
			],
		},
		{ role: "windowMenu" },
	]))
}

app.on("window-all-closed", () => {
	if (!EmbeddedBackend.runningInstances) {
		app.quit()
	}
})

app.on("before-quit", evt => {
	mainWindow.quitting = true
	if (EmbeddedBackend.runningInstances) {
		evt.preventDefault()
		EmbeddedBackend.stopAll().then(() => app.quit())
	}
})

electronDl({
	saveAs: process.platform !== "darwin",
})

app.whenReady().then(async () => {
	mainWindow.config = await loadConfig()
	mainWindow.initialize()
	prepareMenu()
	mainWindow.open()
	mainWindow.createTray()
	const lastArg = process.argv[process.argv.length - 1]
	if (lastArg.startsWith("matrix:")) {
		mainWindow.handleMatrixURI(lastArg)
	}

	if (buildInfo.ci && buildInfo.updateChannel) {
		updateElectronApp({
			updateSource: {
				type: UpdateSourceType.StaticStorage,
				baseUrl: `https://update.gomuks.app/desktop-${buildInfo.updateChannel}/${process.platform}/${process.arch}`,
			},
			updateInterval: "1 hour",
			notifyUser: true,
		})
	}
})
