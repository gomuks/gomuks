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
	"fmt"
	"slices"
	"strconv"
	"sync"

	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/rpc/store"
	"go.mau.fi/gomuks/tui/widget"
)

type RoomList struct {
	lock sync.RWMutex

	parent *MainView

	rooms       []*store.RoomListEntry
	activeSpace id.RoomID
	selected    id.RoomID

	scrollOffset int
	height       int
	width        int

	// The item main text color.
	mainTextColor tcell.Color
	// The text color for selected items.
	selectedTextColor tcell.Color
	// The background color for selected items.
	selectedBackgroundColor tcell.Color
}

func NewRoomList(parent *MainView) *RoomList {
	list := &RoomList{
		parent: parent,

		scrollOffset: 0,

		mainTextColor:           tcell.ColorDefault,
		selectedTextColor:       tcell.ColorWhite,
		selectedBackgroundColor: tcell.ColorDarkGreen,
	}
	return list
}

func (list *RoomList) SetSelected(roomID id.RoomID) {
	list.selected = roomID
	pos := list.index(roomID)
	if pos <= list.scrollOffset {
		list.scrollOffset = pos - 1
	} else if pos >= list.scrollOffset+list.height {
		list.scrollOffset = pos - list.height + 1
	}
	if list.scrollOffset < 0 {
		list.scrollOffset = 0
	}
}

func (list *RoomList) HasSelected() bool {
	return list.selected != ""
}

func (list *RoomList) SelectedRoom() id.RoomID {
	return list.selected
}

func (list *RoomList) updateRoomsLocked() {
	allRooms := list.parent.matrix.ReversedRoomList.Current()
	if list.activeSpace == "" {
		list.rooms = allRooms
		return
	}
	filtered := make([]*store.RoomListEntry, 0, len(allRooms))
	for _, room := range allRooms {
		if list.parent.matrix.IsRoomInSpace(list.activeSpace, room.RoomID) {
			filtered = append(filtered, room)
		}
	}
	list.rooms = filtered
}

func (list *RoomList) SetActiveSpace(spaceID id.RoomID) {
	list.lock.Lock()
	defer list.lock.Unlock()
	list.activeSpace = spaceID
	list.updateRoomsLocked()
	list.scrollOffset = 0
}

func (list *RoomList) GetActiveSpace() id.RoomID {
	list.lock.RLock()
	defer list.lock.RUnlock()
	return list.activeSpace
}

func (list *RoomList) Previous() id.RoomID {
	list.lock.Lock()
	defer list.lock.Unlock()
	list.updateRoomsLocked()
	idx := list.index(list.selected)
	if idx > 0 && idx < len(list.rooms) {
		return list.rooms[idx-1].RoomID
	}
	return ""
}

func (list *RoomList) Next() id.RoomID {
	list.lock.Lock()
	defer list.lock.Unlock()
	list.updateRoomsLocked()
	if len(list.rooms) == 0 {
		return ""
	}
	if list.selected == "" {
		return list.rooms[0].RoomID
	}
	idx := list.index(list.selected)
	if idx >= 0 && idx < len(list.rooms)-1 {
		return list.rooms[idx+1].RoomID
	}
	return ""
}

func (list *RoomList) NextWithActivity() id.RoomID {
	list.lock.Lock()
	defer list.lock.Unlock()
	list.updateRoomsLocked()
	for _, room := range list.rooms {
		if room.UnreadHighlights > 0 || room.UnreadMessages > 0 || room.MarkedUnread {
			return room.RoomID
		}
	}
	return ""
}

func (list *RoomList) index(roomID id.RoomID) int {
	return slices.IndexFunc(list.rooms, func(entry *store.RoomListEntry) bool {
		return entry.RoomID == roomID
	})
}

func (list *RoomList) OnKeyEvent(_ mauview.KeyEvent) bool {
	return false
}

func (list *RoomList) OnPasteEvent(_ mauview.PasteEvent) bool {
	return false
}

func (list *RoomList) OnMouseEvent(event mauview.MouseEvent) bool {
	if event.HasMotion() {
		return false
	}
	switch event.Buttons() {
	case tcell.WheelUp:
		list.addScrollOffset(-WheelScrollOffsetDiff)
		return true
	case tcell.WheelDown:
		list.addScrollOffset(WheelScrollOffsetDiff)
		return true
	case tcell.Button1:
		_, y := event.Position()
		list.lock.RLock()
		defer list.lock.RUnlock()
		y += list.scrollOffset
		if y < 0 || y > len(list.rooms) {
			return false
		}
		list.parent.SwitchRoom(list.rooms[y].RoomID)
		return true
	}
	return false
}

func (list *RoomList) addScrollOffset(offset int) {
	list.scrollOffset += offset
	if list.scrollOffset > len(list.rooms)-list.height {
		list.scrollOffset = len(list.rooms) - list.height
	}
	if list.scrollOffset < 0 {
		list.scrollOffset = 0
	}
}

func (list *RoomList) Focus() {}
func (list *RoomList) Blur()  {}

func (list *RoomList) Draw(screen mauview.Screen) {
	list.lock.Lock()
	list.updateRoomsLocked()
	list.width, list.height = screen.Size()
	roomSlice := list.rooms[min(len(list.rooms), list.scrollOffset):min(len(list.rooms), list.scrollOffset+list.height)]
	list.lock.Unlock()

	for y, room := range roomSlice {
		style := tcell.StyleDefault.
			Foreground(list.mainTextColor).
			Bold(room.MarkedUnread || room.UnreadNotifications > 0 || room.UnreadHighlights > 0)
		if room.RoomID == list.selected {
			style = style.
				Foreground(list.selectedTextColor).
				Background(list.selectedBackgroundColor)
		}

		widget.WriteLinePadded(screen, mauview.AlignLeft, room.Name, 0, y, list.width, style)

		if room.UnreadMessages > 0 {
			unreadMessageCount := "99+"
			if room.UnreadMessages < 1000 {
				unreadMessageCount = strconv.Itoa(room.UnreadMessages)
			}
			if room.UnreadHighlights > 0 {
				unreadMessageCount += "!"
			}
			unreadMessageCount = fmt.Sprintf("(%s)", unreadMessageCount)
			widget.WriteLine(screen, mauview.AlignRight, unreadMessageCount, list.width-7, y, 7, style)
		}
	}
}
