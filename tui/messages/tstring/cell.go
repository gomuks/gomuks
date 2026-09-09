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

package tstring

import (
	"github.com/mattn/go-runewidth"

	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"
)

type Cell struct {
	Char  rune
	Comb  []rune
	Style tcell.Style
}

func NewStyleCell(char rune, style tcell.Style) Cell {
	return Cell{Char: char, Style: style}
}

func NewCombCell(char rune, comb []rune, style tcell.Style) Cell {
	return Cell{Char: char, Comb: comb, Style: style}
}

func NewColorCell(char rune, color tcell.Color) Cell {
	return Cell{Char: char, Style: tcell.StyleDefault.Foreground(color)}
}

func NewCell(char rune) Cell {
	return Cell{Char: char, Style: tcell.StyleDefault}
}

func (cell Cell) RuneWidth() int {
	if cell.Char == 0x10EEEE {
		return 1
	}
	return runewidth.RuneWidth(cell.Char)
}

func (cell Cell) Draw(screen mauview.Screen, x, y int) (chWidth int) {
	chWidth = cell.RuneWidth()
	for runeWidthOffset := 0; runeWidthOffset < chWidth; runeWidthOffset++ {
		screen.SetContent(x+runeWidthOffset, y, cell.Char, cell.Comb, cell.Style)
	}
	return
}
