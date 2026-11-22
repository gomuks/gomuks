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
	"fmt"
	"sort"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"go.mau.fi/mauview"

	"go.mau.fi/gomuks/tui/config"
)

type FuzzyPickerModal struct {
	mauview.Component

	container *mauview.Box

	search  *mauview.InputArea
	results *mauview.TextView

	matches  fuzzy.Ranks
	selected int

	titles   []string
	onSelect func(index int)

	parent *MainView
}

func NewFuzzyPickerModal(mainView *MainView, title string, titles []string, onSelect func(index int), width int, height int) *FuzzyPickerModal {
	fp := &FuzzyPickerModal{
		parent:   mainView,
		titles:   titles,
		onSelect: onSelect,
	}

	fp.results = mauview.NewTextView().SetRegions(true)
	fp.search = mauview.NewInputArea().
		SetChangedFunc(fp.changeHandler).
		SetTextColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorDarkCyan)
	fp.search.Focus()

	flex := mauview.NewFlex().
		SetDirection(mauview.FlexRow).
		AddFixedComponent(fp.search, 1).
		AddProportionalComponent(fp.results, 1)

	fp.container = mauview.NewBox(flex).
		SetBorder(true).
		SetTitle(title).
		SetBlurCaptureFunc(func() bool {
			fp.parent.HideModal()
			return true
		})

	fp.Component = mauview.Center(fp.container, width, height).SetAlwaysFocusChild(true)
	fp.changeHandler("")

	return fp
}

func (fp *FuzzyPickerModal) Focus() {
	fp.container.Focus()
}

func (fp *FuzzyPickerModal) Blur() {
	fp.container.Blur()
}

func (fp *FuzzyPickerModal) changeHandler(str string) {
	if len(str) == 0 {
		fp.matches = make(fuzzy.Ranks, len(fp.titles))
		for i, title := range fp.titles {
			fp.matches[i] = fuzzy.Rank{
				Source:        "",
				Target:        title,
				OriginalIndex: i,
				Distance:      0,
			}
		}
		fp.results.Clear()
		for _, match := range fp.matches {
			_, _ = fmt.Fprintf(fp.results, `["%d"]%s[""]%s`, match.OriginalIndex, match.Target, "\n")
		}
		if len(fp.matches) > 0 {
			fp.results.Highlight(strconv.Itoa(fp.matches[0].OriginalIndex))
			fp.selected = 0
			fp.results.ScrollToBeginning()
		} else {
			fp.results.Highlight()
		}
		return
	}

	fp.matches = fuzzy.RankFindFold(str, fp.titles)
	if len(fp.matches) > 0 {
		sort.Sort(fp.matches)
		fp.results.Clear()
		for _, match := range fp.matches {
			_, _ = fmt.Fprintf(fp.results, `["%d"]%s[""]%s`, match.OriginalIndex, match.Target, "\n")
		}
		fp.results.Highlight(strconv.Itoa(fp.matches[0].OriginalIndex))
		fp.selected = 0
		fp.results.ScrollToBeginning()
	} else {
		fp.results.Clear()
		fp.results.Highlight()
	}
}


func (fp *FuzzyPickerModal) OnKeyEvent(event mauview.KeyEvent) bool {
	highlights := fp.results.GetHighlights()
	kb := config.Keybind{
		Key: event.Key(),
		Ch:  event.Rune(),
		Mod: event.Modifiers(),
	}
	switch fp.parent.config.Keybindings.Modal[kb] {
	case "cancel":
		fp.parent.HideModal()
		return true
	case "select_next":
		if len(highlights) > 0 {
			fp.selected = (fp.selected + 1) % len(fp.matches)
			fp.results.Highlight(strconv.Itoa(fp.matches[fp.selected].OriginalIndex))
			fp.results.ScrollToHighlight()
		}
		return true
	case "select_prev":
		if len(highlights) > 0 {
			fp.selected = (fp.selected - 1) % len(fp.matches)
			if fp.selected < 0 {
				fp.selected += len(fp.matches)
			}
			fp.results.Highlight(strconv.Itoa(fp.matches[fp.selected].OriginalIndex))
			fp.results.ScrollToHighlight()
		}
		return true
	case "confirm":
		if len(highlights) > 0 && fp.onSelect != nil {
			fp.onSelect(fp.matches[fp.selected].OriginalIndex)
		}
		fp.parent.HideModal()
		fp.results.Clear()
		fp.search.SetText("")
		return true
	}
	return fp.search.OnKeyEvent(event)
}
