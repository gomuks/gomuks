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
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestNextKittyImageID(t *testing.T) {
	id1 := NextKittyImageID()
	id2 := NextKittyImageID()
	if id1 == 0 || id2 == 0 {
		t.Fatalf("expected non-zero IDs, got %d, %d", id1, id2)
	}
	if id1 == id2 {
		t.Fatalf("expected unique IDs, got %d == %d", id1, id2)
	}
	if id1 > 0x00FFFFFF || id2 > 0x00FFFFFF {
		t.Fatalf("IDs must be 24-bit, got %d, %d", id1, id2)
	}
}

func TestKittyDiacritic(t *testing.T) {
	d0 := KittyDiacritic(0)
	d1 := KittyDiacritic(1)
	if d0 != 0x0305 {
		t.Errorf("expected diacritic(0) to be 0x0305, got 0x%04X", d0)
	}
	if d1 != 0x030D {
		t.Errorf("expected diacritic(1) to be 0x030D, got 0x%04X", d1)
	}
	dOut := KittyDiacritic(9999)
	if dOut != 0x0305 {
		t.Errorf("expected out-of-bounds to fallback to 0x0305, got 0x%04X", dOut)
	}
}

func TestFormatKittyTransmit(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	seq := FormatKittyTransmit(img, 42, false)
	seqStr := string(seq)

	if !strings.HasPrefix(seqStr, "\x1b_Gq=2,") {
		t.Errorf("expected Kitty prefix, got: %q", seqStr[:20])
	}
	if !strings.Contains(seqStr, "i=42,a=T,U=1,f=32,t=d,s=10,v=10,") {
		t.Errorf("expected placement parameters in transmit sequence, got: %q", seqStr)
	}
	if !strings.HasSuffix(seqStr, "\x1b\\") {
		t.Errorf("expected Kitty string terminator suffix, got: %q", seqStr[len(seqStr)-4:])
	}
}

func TestFormatKittyTransmit_Tmux(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	seq := FormatKittyTransmit(img, 100, true)
	seqStr := string(seq)

	if !strings.HasPrefix(seqStr, "\x1bPtmux;\x1b\x1b_Gq=2,") {
		t.Errorf("expected tmux DCS prefix, got: %q", seqStr[:30])
	}
	if !strings.HasSuffix(seqStr, "\x1b\\\x1b\\") {
		t.Errorf("expected tmux DCS suffix, got: %q", seqStr[len(seqStr)-6:])
	}
}

func TestCreateKittyPlaceholderBuffer(t *testing.T) {
	cols, rows := 20, 5
	id := uint32(0x00112233)
	buf := CreateKittyPlaceholderBuffer(cols, rows, id)

	if len(buf) != rows {
		t.Fatalf("expected %d rows, got %d", rows, len(buf))
	}
	for y, line := range buf {
		if len(line) != cols {
			t.Fatalf("row %d: expected %d cols, got %d", y, cols, len(line))
		}
		for x, cell := range line {
			if cell.Char != KittyPlaceholderRune {
				t.Fatalf("row %d col %d: expected rune 0x10EEEE, got 0x%X", y, x, cell.Char)
			}
			if len(cell.Comb) != 3 {
				t.Fatalf("row %d col %d: expected 3 combining runes, got %d", y, x, len(cell.Comb))
			}
			if cell.Comb[0] != KittyDiacritic(y) {
				t.Errorf("row %d: expected row diacritic 0x%X, got 0x%X", y, KittyDiacritic(y), cell.Comb[0])
			}
			if cell.Comb[1] != KittyDiacritic(x) {
				t.Errorf("col %d: expected col diacritic 0x%X, got 0x%X", x, KittyDiacritic(x), cell.Comb[1])
			}
		}
	}
}
