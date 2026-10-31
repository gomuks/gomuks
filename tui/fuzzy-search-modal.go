// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2020 Tulir Asokan
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

package tui

import (
	"go.mau.fi/gomuks/tui/debug"
)

type FuzzySearchModal = FuzzyPickerModal

func NewFuzzySearchModal(mainView *MainView, width int, height int) *FuzzySearchModal {
	roomList := mainView.matrix.ReversedRoomList.Current()
	titles := make([]string, len(roomList))
	for i, room := range roomList {
		titles[i] = room.Name
	}
	return NewFuzzyPickerModal(mainView, "Quick Room Switcher", titles, func(idx int) {
		if idx >= 0 && idx < len(roomList) {
			debug.Print("Fuzzy Selected Room:", roomList[idx].Name)
			mainView.SwitchRoom(roomList[idx].RoomID)
		}
	}, width, height)
}
