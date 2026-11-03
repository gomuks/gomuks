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
	"go.mau.fi/gomuks/pkg/rpc/store"
	"go.mau.fi/gomuks/tui/debug"
)

type SpaceSwitcherModal = FuzzyPickerModal

func NewSpaceSwitcherModal(mainView *MainView, width int, height int) *SpaceSwitcherModal {
	rawSpaces := mainView.matrix.GetSpaceList()
	spaceList := make([]*store.SpaceEntry, 0, len(rawSpaces)+1)
	spaceList = append(spaceList, &store.SpaceEntry{
		RoomID: "",
		Name:   "All Rooms",
	})
	spaceList = append(spaceList, rawSpaces...)

	titles := make([]string, len(spaceList))
	for i, space := range spaceList {
		titles[i] = space.Name
	}
	return NewFuzzyPickerModal(mainView, "Space Switcher", titles, func(idx int) {
		if idx >= 0 && idx < len(spaceList) {
			selectedSpace := spaceList[idx]
			debug.Print("Space Switcher: Selected", selectedSpace.Name, selectedSpace.RoomID)
			mainView.SwitchToSpace(selectedSpace.RoomID)
		}
	}, width, height)
}
