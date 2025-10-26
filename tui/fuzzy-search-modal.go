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
	"sort"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"go.mau.fi/mauview"

	"go.mau.fi/gomuks/pkg/rpc/store"
	"go.mau.fi/gomuks/tui/config"
	"go.mau.fi/gomuks/tui/debug"
)

type FuzzySearchModal struct {
	mauview.Component

	container *mauview.Box

	search  *mauview.InputArea
	results *mauview.TextView

	matches  fuzzy.Ranks
	selected int

	roomList   []*store.RoomListEntry
	roomTitles []string

	parent *MainView
}

func NewFuzzySearchModal(mainView *MainView, width int, height int) *FuzzySearchModal {
	fs := &FuzzySearchModal{
		parent:   mainView,
		roomList: mainView.matrix.ReversedRoomList.Current(),
	}
	fs.roomTitles = make([]string, len(fs.roomList))
	for i, room := range fs.roomList {
		fs.roomTitles[i] = room.Name
	}

	fs.results = mauview.NewTextView().SetRegions(true)
	fs.search = mauview.NewInputArea().
		SetChangedFunc(fs.changeHandler).
		SetTextColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorDarkCyan)
	fs.search.Focus()

	flex := mauview.NewFlex().
		SetDirection(mauview.FlexRow).
		AddFixedComponent(fs.search, 1).
		AddProportionalComponent(fs.results, 1)

	fs.container = mauview.NewBox(flex).
		SetBorder(true).
		SetTitle("Quick Room Switcher").
		SetBlurCaptureFunc(func() bool {
			fs.parent.HideModal()
			return true
		})

	fs.Component = mauview.Center(fs.container, width, height).SetAlwaysFocusChild(true)

	return fs
}

func (fs *FuzzySearchModal) Focus() {
	fs.container.Focus()
}

func (fs *FuzzySearchModal) Blur() {
	fs.container.Blur()
}

func (fs *FuzzySearchModal) changeHandler(str string) {
	// Get matches and display in result box
	fs.matches = fuzzy.RankFindFold(str, fs.roomTitles)
	if len(str) > 0 && len(fs.matches) > 0 {
		sort.Sort(fs.matches)
		fs.results.Clear()
		for _, match := range fs.matches {
			_, _ = fmt.Fprintf(fs.results, `["%d"]%s[""]%s`, match.OriginalIndex, match.Target, "\n")
		}
		//fs.parent.parent.Render()
		fs.results.Highlight(strconv.Itoa(fs.matches[0].OriginalIndex))
		fs.selected = 0
		fs.results.ScrollToBeginning()
	} else {
		fs.results.Clear()
		fs.results.Highlight()
	}
}

func (fs *FuzzySearchModal) OnKeyEvent(event mauview.KeyEvent) bool {
	highlights := fs.results.GetHighlights()
	kb := config.Keybind{
		Key: event.Key(),
		Ch:  event.Rune(),
		Mod: event.Modifiers(),
	}
	switch fs.parent.config.Keybindings.Modal[kb] {
	case "cancel":
		// Close room finder
		fs.parent.HideModal()
		return true
	case "select_next":
		// Cycle highlighted area to next match
		if len(highlights) > 0 {
			fs.selected = (fs.selected + 1) % len(fs.matches)
			fs.results.Highlight(strconv.Itoa(fs.matches[fs.selected].OriginalIndex))
			fs.results.ScrollToHighlight()
		}
		return true
	case "select_prev":
		if len(highlights) > 0 {
			fs.selected = (fs.selected - 1) % len(fs.matches)
			if fs.selected < 0 {
				fs.selected += len(fs.matches)
			}
			fs.results.Highlight(strconv.Itoa(fs.matches[fs.selected].OriginalIndex))
			fs.results.ScrollToHighlight()
		}
		return true
	case "confirm":
		// Switch room to currently selected room
		if len(highlights) > 0 {
			debug.Print("Fuzzy Selected Room:", fs.roomList[fs.matches[fs.selected].OriginalIndex].Name)
			fs.parent.SwitchRoom(fs.roomList[fs.matches[fs.selected].OriginalIndex].RoomID)
		}
		fs.parent.HideModal()
		fs.results.Clear()
		fs.search.SetText("")
		return true
	}
	return fs.search.OnKeyEvent(event)
}
