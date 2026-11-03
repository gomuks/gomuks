// gomuks - A terminal Matrix client written in Go.
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

package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"go.mau.fi/gomuks/tui/config"
)

func TestFuzzyPickerModal_SearchAndNavigation(t *testing.T) {
	cfg := config.NewConfig()
	cfg.LoadKeybindings()

	mainView := &MainView{
		config: cfg,
	}

	titles := []string{"General", "Random", "Development", "Announcements"}
	var selectedIndex int = -1

	picker := NewFuzzyPickerModal(mainView, "Test Picker", titles, func(index int) {
		selectedIndex = index
	}, 40, 10)

	// Search for "Dev"
	picker.changeHandler("Dev")
	if len(picker.matches) != 1 {
		t.Fatalf("expected 1 match for 'Dev', got %d", len(picker.matches))
	}
	if picker.matches[0].OriginalIndex != 2 {
		t.Errorf("expected original index 2 ('Development'), got %d", picker.matches[0].OriginalIndex)
	}

	// Confirm selection
	enterEvt := tcell.NewEventKey(tcell.KeyEnter, 13, 0)
	handled := picker.OnKeyEvent(enterEvt)
	if !handled {
		t.Error("expected Enter key event to be handled")
	}
	if selectedIndex != 2 {
		t.Errorf("expected callback with index 2, got %d", selectedIndex)
	}
}

func TestFuzzyPickerModal_CycleMatches(t *testing.T) {
	cfg := config.NewConfig()
	cfg.LoadKeybindings()

	mainView := &MainView{
		config: cfg,
	}

	titles := []string{"Alpha", "Alphabet", "Alpine"}
	var selectedIndex int = -1

	picker := NewFuzzyPickerModal(mainView, "Test Picker", titles, func(index int) {
		selectedIndex = index
	}, 40, 10)

	picker.changeHandler("Al")
	if len(picker.matches) != 3 {
		t.Fatalf("expected 3 matches for 'Al', got %d", len(picker.matches))
	}

	// Next
	downEvt := tcell.NewEventKey(tcell.KeyDown, 0, 0)
	picker.OnKeyEvent(downEvt)
	if picker.selected != 1 {
		t.Errorf("expected selected index 1, got %d", picker.selected)
	}

	// Next again
	picker.OnKeyEvent(downEvt)
	if picker.selected != 2 {
		t.Errorf("expected selected index 2, got %d", picker.selected)
	}

	// Prev
	upEvt := tcell.NewEventKey(tcell.KeyUp, 0, 0)
	picker.OnKeyEvent(upEvt)
	if picker.selected != 1 {
		t.Errorf("expected selected index 1, got %d", picker.selected)
	}

	// Confirm
	expectedIndex := picker.matches[picker.selected].OriginalIndex
	enterEvt := tcell.NewEventKey(tcell.KeyEnter, 13, 0)
	picker.OnKeyEvent(enterEvt)
	if selectedIndex != expectedIndex {
		t.Errorf("expected selected item %d, got %d", expectedIndex, selectedIndex)
	}
}

func TestFuzzyPickerModal_InitialPopulate(t *testing.T) {
	cfg := config.NewConfig()
	cfg.LoadKeybindings()

	mainView := &MainView{
		config: cfg,
	}

	titles := []string{"All Rooms", "Work Space", "Community", "Gaming"}
	var selectedIndex int = -1

	picker := NewFuzzyPickerModal(mainView, "Space Switcher", titles, func(index int) {
		selectedIndex = index
	}, 40, 10)

	// Initially with query "", all 4 items should be listed
	if len(picker.matches) != 4 {
		t.Fatalf("expected 4 matches initially on empty query, got %d", len(picker.matches))
	}
	if picker.selected != 0 {
		t.Errorf("expected initially selected index 0, got %d", picker.selected)
	}
	if picker.matches[0].Target != "All Rooms" {
		t.Errorf("expected first item to be 'All Rooms', got %q", picker.matches[0].Target)
	}

	// Immediate Enter on open selects the first item ("All Rooms")
	enterEvt := tcell.NewEventKey(tcell.KeyEnter, 13, 0)
	picker.OnKeyEvent(enterEvt)
	if selectedIndex != 0 {
		t.Errorf("expected selectedIndex 0 on initial Enter, got %d", selectedIndex)
	}
}

func TestFuzzyPickerModal_FilterAndRestoreAll(t *testing.T) {
	cfg := config.NewConfig()
	cfg.LoadKeybindings()

	mainView := &MainView{
		config: cfg,
	}

	titles := []string{"All Rooms", "Work Space", "Community", "Gaming"}

	picker := NewFuzzyPickerModal(mainView, "Space Switcher", titles, nil, 40, 10)

	// Filter down to 1 item
	picker.changeHandler("Gaming")
	if len(picker.matches) != 1 {
		t.Fatalf("expected 1 match for 'Gaming', got %d", len(picker.matches))
	}
	if picker.matches[0].Target != "Gaming" {
		t.Errorf("expected match to be 'Gaming', got %q", picker.matches[0].Target)
	}

	// Backspace / clear query restores all 4 items
	picker.changeHandler("")
	if len(picker.matches) != 4 {
		t.Fatalf("expected 4 matches after clearing query, got %d", len(picker.matches))
	}
	if picker.matches[0].Target != "All Rooms" {
		t.Errorf("expected first item to be 'All Rooms', got %q", picker.matches[0].Target)
	}
}


