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
import { WebContents } from "electron"
import path from "node:path"

export function loadPage(webContents: WebContents, file: string) {
	if (MAIN_WINDOW_VITE_DEV_SERVER_URL) {
		return webContents.loadURL(`${MAIN_WINDOW_VITE_DEV_SERVER_URL}/src/chrome/${file}`)
	} else {
		return webContents.loadFile(path.join(__dirname, `../renderer/${MAIN_WINDOW_VITE_NAME}/src/chrome/${file}`))
	}
}
