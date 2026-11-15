// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2025 Tulir Asokan
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
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"
	"go.mau.fi/util/ptr"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
	"go.mau.fi/gomuks/pkg/rpc/client"
	"go.mau.fi/gomuks/pkg/rpc/store"
	"go.mau.fi/gomuks/tui/config"
	"go.mau.fi/gomuks/tui/debug"
	"go.mau.fi/gomuks/tui/lib/notification"
	"go.mau.fi/gomuks/tui/widget"
)

type MainView struct {
	flex *mauview.Flex

	roomList    *RoomList
	roomView    *mauview.Box
	currentRoom *RoomView
	activeSpace id.RoomID
	//cmdProcessor *CommandProcessor
	focused mauview.Focusable

	modal mauview.Component

	lastFocusTime time.Time

	matrix *client.GomuksClient
	config *config.Config
	parent *GomuksTUI
}

func (ui *GomuksTUI) NewMainView() mauview.Component {
	mainView := &MainView{
		flex:     mauview.NewFlex().SetDirection(mauview.FlexColumn),
		roomView: mauview.NewBox(nil).SetBorder(false),

		matrix: ui.gmx,
		config: ui.Config,
		parent: ui,
	}
	mainView.roomList = NewRoomList(mainView)
	//mainView.cmdProcessor = NewCommandProcessor(mainView)

	mainView.flex.
		AddFixedComponent(mainView.roomList, 25).
		AddFixedComponent(widget.NewBorder(), 1).
		AddProportionalComponent(mainView.roomView, 1)
	mainView.BumpFocus(nil)

	ui.MainView = mainView

	return mainView
}

func (view *MainView) ShowModal(modal mauview.Component) {
	view.modal = modal
	var ok bool
	view.focused, ok = modal.(mauview.Focusable)
	if !ok {
		view.focused = nil
	} else {
		view.focused.Focus()
	}
}

func (view *MainView) HideModal() {
	view.modal = nil
	view.focused = view.roomView
}

func (view *MainView) Draw(screen mauview.Screen) {
	if view.config.Preferences.HideRoomList {
		view.roomView.Draw(screen)
	} else {
		view.flex.Draw(screen)
	}

	if view.modal != nil {
		view.modal.Draw(screen)
	}
}

func (view *MainView) BumpFocus(roomView *RoomView) {
	if roomView != nil {
		view.lastFocusTime = time.Now()
		view.MarkRead(roomView)
	}
}

func (view *MainView) MarkRead(roomView *RoomView) {
	if roomView != nil && roomView == view.currentRoom && roomView.MessageView().GetScrollOffset() == 0 {
		req := roomView.Room.GetMarkAsReadParams()
		if req != nil {
			go func() {
				defer debug.Recover()
				err := view.matrix.MarkRead(context.TODO(), req)
				if err != nil {
					debug.Print("Failed to mark read for", roomView.Room.ID, err)
				}
			}()
		}
	}
}

func (view *MainView) InputChanged(roomView *RoomView, text string) {
	//if !roomView.config.Preferences.DisableTypingNotifs {
	//	view.matrix.SendTyping(roomView.Room.ID, len(text) > 0 && text[0] != '/')
	//}
}

func (view *MainView) ShowBare(roomView *RoomView) {
	if roomView == nil {
		return
	}
	_, height := view.parent.app.Screen().Size()
	view.parent.app.Suspend(func() {
		print("\033[2J\033[0;0H")
		// We don't know how much space there exactly is. Too few messages looks weird,
		// and too many messages shouldn't cause any problems, so we just show too many.
		height *= 2
		fmt.Println(roomView.MessageView().CapturePlaintext(height))
		fmt.Println("Press enter to return to normal mode.")
		reader := bufio.NewReader(os.Stdin)
		_, _, _ = reader.ReadRune()
		print("\033[2J\033[0;0H")
	})
}

func (view *MainView) OpenSyncingModal() *SyncingModal {
	component, modal := NewSyncingModal(view)
	view.ShowModal(component)
	return modal
}

func (view *MainView) OnKeyEvent(event mauview.KeyEvent) bool {
	view.BumpFocus(view.currentRoom)

	if view.modal != nil {
		return view.modal.OnKeyEvent(event)
	}

	kb := config.Keybind{
		Key: event.Key(),
		Ch:  event.Rune(),
		Mod: event.Modifiers(),
	}
	switch view.config.Keybindings.Main[kb] {
	case "next_room":
		view.SwitchRoom(view.roomList.Next())
	case "prev_room":
		view.SwitchRoom(view.roomList.Previous())
	case "search_rooms":
		view.ShowModal(NewFuzzySearchModal(view, 42, 12))
	case "search_spaces":
		view.ShowModal(NewSpaceSwitcherModal(view, 42, 12))
	case "next_space":
		view.SwitchToNextSpace()
	case "prev_space":
		view.SwitchToPrevSpace()
	case "scroll_up":
		msgView := view.currentRoom.MessageView()
		msgView.AddScrollOffset(msgView.TotalHeight())
	case "scroll_down":
		msgView := view.currentRoom.MessageView()
		msgView.AddScrollOffset(-msgView.TotalHeight())
	case "add_newline":
		return view.flex.OnKeyEvent(tcell.NewEventKey(tcell.KeyEnter, '\n', event.Modifiers()|tcell.ModShift))
	case "next_active_room":
		view.SwitchRoom(view.roomList.NextWithActivity())
	case "show_bare":
		view.ShowBare(view.currentRoom)
	case "force_quit":
		view.parent.Finish()
		return false
	case "quit":
		view.parent.Stop()
		return false
	default:
		if view.config.Preferences.HideRoomList {
			return view.roomView.OnKeyEvent(event)
		}
		return view.flex.OnKeyEvent(event)
	}
	return true
}

const WheelScrollOffsetDiff = 3

func (view *MainView) OnMouseEvent(event mauview.MouseEvent) bool {
	view.BumpFocus(view.currentRoom)
	if view.modal != nil {
		return view.modal.OnMouseEvent(event)
	}
	if view.config.Preferences.HideRoomList {
		return view.roomView.OnMouseEvent(event)
	}
	return view.flex.OnMouseEvent(event)
}

func (view *MainView) OnPasteEvent(event mauview.PasteEvent) bool {
	if view.modal != nil {
		return view.modal.OnPasteEvent(event)
	} else if view.config.Preferences.HideRoomList {
		return view.roomView.OnPasteEvent(event)
	}
	return view.flex.OnPasteEvent(event)
}

func (view *MainView) Focus() {
	if view.focused != nil {
		view.focused.Focus()
	}
}

func (view *MainView) Blur() {
	if view.focused != nil {
		view.focused.Blur()
	}
}

func (view *MainView) SwitchRoom(roomID id.RoomID) {
	roomData := view.matrix.GetRoom(roomID)
	if roomData == nil {
		debug.Print("Tried to switch to nonexistent room!", roomID)
		return
	}
	debug.Print("Selecting room", roomID)
	view.roomList.SetSelected(roomID)
	view.flex.SetFocused(view.roomView)
	if view.currentRoom != nil {
		view.currentRoom.Unload()
	}
	currentRoom := NewRoomView(view, roomData)
	view.currentRoom = currentRoom
	view.roomView.SetInnerComponent(currentRoom)
	view.roomView.Focus()
	view.MarkRead(currentRoom)
	if len(ptr.Val(roomData.TimelineCache.Current())) < 50 {
		go view.LoadHistory(roomID)
	}
	if !roomData.FullMembersLoaded.Load() {
		// TODO only load necessary members rather than all?
		go func() {
			defer debug.Recover()
			err := view.matrix.LoadRoomState(context.TODO(), roomID, true, false)
			if err != nil {
				debug.Print("Failed to load room state for", roomID, err)
			} else {
				currentRoom.UpdateUserList()
				view.parent.Render()
			}
		}()
	}
	view.parent.Render()
}

func (view *MainView) ShowStatus(msg string) {
	if view.currentRoom != nil {
		view.currentRoom.SetStatusNotification(msg, 3*time.Second)
	}
}

func (view *MainView) SwitchToSpace(spaceID id.RoomID) {
	if spaceID == "" {
		view.activeSpace = ""
		view.roomList.SetActiveSpace("")
		view.ShowStatus("Showing all rooms")
		view.parent.Render()
		return
	}
	rooms := view.matrix.GetRoomsInSpace(spaceID)
	if len(rooms) == 0 {
		debug.Print("Space", spaceID, "has no rooms")
		view.ShowStatus("Space has no rooms")
		return
	}
	var targetRoom id.RoomID
	for _, roomID := range rooms {
		if view.matrix.GetRoom(roomID) != nil {
			targetRoom = roomID
			break
		}
	}
	if targetRoom == "" {
		debug.Print("Space", spaceID, "has no joined rooms")
		view.ShowStatus("Space has no joined rooms")
		return
	}

	if view.activeSpace == spaceID && view.currentRoom != nil && view.currentRoom.Room.ID == targetRoom {
		return
	}

	view.activeSpace = spaceID
	view.roomList.SetActiveSpace(spaceID)
	view.SwitchRoom(targetRoom)
}

func (view *MainView) SwitchToNextSpace() {
	spaces := view.matrix.GetSpaceList()
	if len(spaces) == 0 {
		view.ShowStatus("No spaces available")
		return
	}
	if len(spaces) == 1 {
		if view.activeSpace == spaces[0].RoomID {
			return
		}
		view.SwitchToSpace(spaces[0].RoomID)
		return
	}
	if view.activeSpace != "" {
		for i, space := range spaces {
			if space.RoomID == view.activeSpace {
				nextSpace := spaces[(i+1)%len(spaces)]
				view.SwitchToSpace(nextSpace.RoomID)
				return
			}
		}
	}
	currentRoom := view.currentRoom
	if currentRoom != nil {
		for i, space := range spaces {
			if view.matrix.IsRoomInSpace(space.RoomID, currentRoom.Room.ID) {
				nextSpace := spaces[(i+1)%len(spaces)]
				view.SwitchToSpace(nextSpace.RoomID)
				return
			}
		}
	}
	view.SwitchToSpace(spaces[0].RoomID)
}

func (view *MainView) SwitchToPrevSpace() {
	spaces := view.matrix.GetSpaceList()
	if len(spaces) == 0 {
		view.ShowStatus("No spaces available")
		return
	}
	if len(spaces) == 1 {
		if view.activeSpace == spaces[0].RoomID {
			return
		}
		view.SwitchToSpace(spaces[0].RoomID)
		return
	}
	if view.activeSpace != "" {
		for i, space := range spaces {
			if space.RoomID == view.activeSpace {
				prevSpace := spaces[(i-1+len(spaces))%len(spaces)]
				view.SwitchToSpace(prevSpace.RoomID)
				return
			}
		}
	}
	currentRoom := view.currentRoom
	if currentRoom != nil {
		for i, space := range spaces {
			if view.matrix.IsRoomInSpace(space.RoomID, currentRoom.Room.ID) {
				prevSpace := spaces[(i-1+len(spaces))%len(spaces)]
				view.SwitchToSpace(prevSpace.RoomID)
				return
			}
		}
	}
	view.SwitchToSpace(spaces[len(spaces)-1].RoomID)
}

func (view *MainView) NotifyMessage(room *store.RoomStore, notif jsoncmd.SyncNotification) {
	if view.config.Preferences.DisableNotifications {
		return
	}
	currentRoom := view.currentRoom
	isCurrent := currentRoom != nil && currentRoom.Room.ID == room.ID
	recentlyFocused := time.Now().Add(-30 * time.Second).Before(view.lastFocusTime)
	if recentlyFocused && isCurrent {
		debug.Print("Not sending notification: room is focused")
		return
	}
	body := notif.Event.LocalContent.PreviewText
	if len(body) == 0 {
		debug.Print("Not sending notification with empty body")
		return
	}
	memberEvt := room.GetMember(notif.Event.Sender)
	notifTitle := notif.Event.Sender.Localpart()
	if memberEvt != nil && memberEvt.Displayname != "" {
		notifTitle = memberEvt.Displayname
	}
	if roomName := room.Meta.Current().Name; roomName != nil && *roomName != "" && notifTitle != *roomName {
		notifTitle = fmt.Sprintf("%s (%s)", notifTitle, *roomName)
	}
	err := notification.Send(notifTitle, body, notif.Highlight, notif.Sound)
	if err != nil {
		debug.Print("Failed to send notification:", err)
	} else {
		debug.Print("Sent notification:", notifTitle, body)
	}
}

func (view *MainView) LoadHistory(roomID id.RoomID) {
	defer debug.Recover()
	err := view.matrix.LoadMoreHistory(context.TODO(), roomID)
	if err != nil {
		debug.Print("Failed to fetch history for", roomID, err)
		return
	}
	view.parent.Render()
	if room := view.currentRoom; room != nil && room.Room.ID == roomID {
		view.MarkRead(room)
	}
}
