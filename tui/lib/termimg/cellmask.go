// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2026 Tulir Asokan, buihongduc132
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

package termimg

import (
	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"

	"go.mau.fi/gomuks/tui/messages/tstring"
)

// CreatePlaceholderBuffer allocates a 2D grid of rows lines, each containing cols empty cells.
// This reserves character cell space in the TUI timeline scroll buffer for hardware image previews,
// ensuring the layout engine correctly measures the message's vertical height and row positions.
func CreatePlaceholderBuffer(cols, rows int) []tstring.TString {
	if cols <= 0 || rows <= 0 {
		return nil
	}

	buf := make([]tstring.TString, rows)
	for y := 0; y < rows; y++ {
		line := make(tstring.TString, cols)
		for x := 0; x < cols; x++ {
			line[x] = tstring.NewStyleCell(' ', tcell.StyleDefault)
		}
		buf[y] = line
	}
	return buf
}

// GetRootScreenAndOffset traverses nested mauview.ProxyScreen wrappers to extract
// the underlying tcell.Screen root and calculate cumulative absolute screen coordinates (offsetX, offsetY).
func GetRootScreenAndOffset(s mauview.Screen) (tcell.Screen, int, int) {
	offsetX, offsetY := 0, 0
	current := s

	for current != nil {
		if proxy, ok := current.(*mauview.ProxyScreen); ok {
			offsetX += proxy.OffsetX
			offsetY += proxy.OffsetY
			current = proxy.Parent
			continue
		}
		break
	}

	if ts, ok := current.(tcell.Screen); ok {
		return ts, offsetX, offsetY
	}

	return nil, offsetX, offsetY
}

// ImageMask encapsulates screen coordinates and state for locking a region of cells
// via tcell.Screen.LockRegion to prevent tcell redrawing over OSC 1337 image graphics.
type ImageMask struct {
	RootScreen tcell.Screen
	ScreenX    int
	ScreenY    int
	Cols       int
	Rows       int
	Active     bool
}

// NewImageMask creates an ImageMask by resolving absolute coordinates from the given mauview.Screen.
func NewImageMask(screen mauview.Screen, x, y, cols, rows int) *ImageMask {
	root, offX, offY := GetRootScreenAndOffset(screen)
	return &ImageMask{
		RootScreen: root,
		ScreenX:    offX + x,
		ScreenY:    offY + y,
		Cols:       cols,
		Rows:       rows,
	}
}

// Apply marks the image mask as active. Note: We strictly avoid calling tcell.Screen.LockRegion
// because locking cells prevents tcell from repainting during timeline scrolling, which creates
// frozen / cached screen regions.
func (m *ImageMask) Apply() {
	if m != nil && !m.Active && m.Cols > 0 && m.Rows > 0 {
		m.Active = true
	}
}

// Clear marks the image mask as inactive.
func (m *ImageMask) Clear() {
	if m != nil && m.Active {
		m.Active = false
	}
}
