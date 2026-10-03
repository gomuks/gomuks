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
	"testing"

	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"
)

func TestCreatePlaceholderBuffer(t *testing.T) {
	t.Run("valid dimensions", func(t *testing.T) {
		cols := 30
		rows := 8
		buf := CreatePlaceholderBuffer(cols, rows)

		if len(buf) != rows {
			t.Fatalf("expected %d rows, got %d", rows, len(buf))
		}
		for y, line := range buf {
			if len(line) != cols {
				t.Fatalf("row %d: expected %d cols, got %d", y, cols, len(line))
			}
			for x, cell := range line {
				if cell.Char != ' ' {
					t.Errorf("row %d col %d: expected space character, got %q", y, x, cell.Char)
				}
				if cell.Style != tcell.StyleDefault {
					t.Errorf("row %d col %d: expected StyleDefault, got %v", y, x, cell.Style)
				}
			}
		}
	})

	t.Run("non-positive dimensions return nil", func(t *testing.T) {
		if buf := CreatePlaceholderBuffer(0, 10); buf != nil {
			t.Errorf("expected nil for 0 cols, got len=%d", len(buf))
		}
		if buf := CreatePlaceholderBuffer(10, 0); buf != nil {
			t.Errorf("expected nil for 0 rows, got len=%d", len(buf))
		}
		if buf := CreatePlaceholderBuffer(-5, -5); buf != nil {
			t.Errorf("expected nil for negative dimensions, got len=%d", len(buf))
		}
	})
}

// mockLockScreen tracks calls to LockRegion
type mockLockScreen struct {
	tcell.SimulationScreen
	lastLockX, lastLockY int
	lastLockW, lastLockH int
	lastLockVal          bool
	lockCallCount        int
}

func newMockLockScreen() *mockLockScreen {
	s := tcell.NewSimulationScreen("")
	_ = s.Init()
	return &mockLockScreen{SimulationScreen: s}
}

func (m *mockLockScreen) LockRegion(x, y, width, height int, lock bool) {
	m.lastLockX = x
	m.lastLockY = y
	m.lastLockW = width
	m.lastLockH = height
	m.lastLockVal = lock
	m.lockCallCount++
	m.SimulationScreen.LockRegion(x, y, width, height, lock)
}

func TestGetRootScreenAndOffset(t *testing.T) {
	mockScreen := newMockLockScreen()

	t.Run("direct root screen", func(t *testing.T) {
		root, x, y := GetRootScreenAndOffset(mockScreen)
		if root != mockScreen {
			t.Errorf("expected root screen to match mockScreen")
		}
		if x != 0 || y != 0 {
			t.Errorf("expected offset (0, 0), got (%d, %d)", x, y)
		}
	})

	t.Run("single proxy screen", func(t *testing.T) {
		proxy := mauview.NewProxyScreen(mockScreen, 12, 24, 80, 24)
		root, x, y := GetRootScreenAndOffset(proxy)
		if root != mockScreen {
			t.Errorf("expected root screen to match mockScreen")
		}
		if x != 12 || y != 24 {
			t.Errorf("expected offset (12, 24), got (%d, %d)", x, y)
		}
	})

	t.Run("nested proxy screen", func(t *testing.T) {
		p1 := mauview.NewProxyScreen(mockScreen, 10, 20, 100, 50)
		p2 := mauview.NewProxyScreen(p1, 5, 8, 60, 30)

		root, x, y := GetRootScreenAndOffset(p2)
		if root != mockScreen {
			t.Errorf("expected root screen to match mockScreen")
		}
		if x != 15 || y != 28 {
			t.Errorf("expected cumulative offset (15, 28), got (%d, %d)", x, y)
		}
	})
}

func TestImageMask_Lifecycle(t *testing.T) {
	mockScreen := newMockLockScreen()
	p1 := mauview.NewProxyScreen(mockScreen, 10, 20, 100, 50)
	p2 := mauview.NewProxyScreen(p1, 5, 8, 60, 30)

	mask := NewImageMask(p2, 2, 4, 32, 16)
	if mask.ScreenX != 17 { // 10 + 5 + 2
		t.Errorf("expected ScreenX 17, got %d", mask.ScreenX)
	}
	if mask.ScreenY != 32 { // 20 + 8 + 4
		t.Errorf("expected ScreenY 32, got %d", mask.ScreenY)
	}
	if mask.Cols != 32 || mask.Rows != 16 {
		t.Errorf("expected 32x16, got %dx%d", mask.Cols, mask.Rows)
	}
	if mask.Active {
		t.Errorf("expected newly created mask to be inactive")
	}

	// 1. Apply mask
	mask.Apply()
	if !mask.Active {
		t.Errorf("expected mask to be active after Apply()")
	}
	// We explicitly expect 0 LockRegion calls to guarantee tcell repaints during scrolling
	if mockScreen.lockCallCount != 0 {
		t.Errorf("expected 0 LockRegion calls to prevent scrolling freeze, got %d", mockScreen.lockCallCount)
	}

	// 2. Redundant Apply should be a no-op
	mask.Apply()
	if mockScreen.lockCallCount != 0 {
		t.Errorf("expected redundant Apply() to not call LockRegion, got count=%d", mockScreen.lockCallCount)
	}

	// 3. Clear mask
	mask.Clear()
	if mask.Active {
		t.Errorf("expected mask to be inactive after Clear()")
	}
	if mockScreen.lockCallCount != 0 {
		t.Errorf("expected 0 LockRegion calls, got %d", mockScreen.lockCallCount)
	}

	// 4. Redundant Clear should be a no-op
	mask.Clear()
	if mockScreen.lockCallCount != 0 {
		t.Errorf("expected redundant Clear() to not call LockRegion, got count=%d", mockScreen.lockCallCount)
	}

	// 5. Nil safety
	var nilMask *ImageMask
	nilMask.Apply()
	nilMask.Clear()
}
