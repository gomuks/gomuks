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
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	"sync/atomic"

	"github.com/gdamore/tcell/v2"

	"go.mau.fi/gomuks/tui/messages/tstring"
)

var nextKittyID uint32

// NextKittyImageID returns a globally unique 24-bit image ID.
func NextKittyImageID() uint32 {
	id := atomic.AddUint32(&nextKittyID, 1) & 0x00FFFFFF
	if id == 0 {
		id = atomic.AddUint32(&nextKittyID, 1) & 0x00FFFFFF
	}
	return id
}

// Kitty diacritics table (297 characters) per official Kitty graphics protocol specification:
// https://sw.kovidgoyal.net/kitty/graphics-protocol/#unicode-placeholders
var kittyDiacritics = [297]rune{
	0x0305, 0x030D, 0x030E, 0x0310, 0x0312, 0x033D, 0x033E, 0x033F, 0x0346, 0x034A,
	0x034B, 0x034C, 0x0350, 0x0351, 0x0352, 0x0357, 0x035B, 0x0363, 0x0364, 0x0365,
	0x0366, 0x0367, 0x0368, 0x0369, 0x036A, 0x036B, 0x036C, 0x036D, 0x036E, 0x036F,
	0x0483, 0x0484, 0x0485, 0x0486, 0x0487, 0x0592, 0x0593, 0x0594, 0x0595, 0x0597,
	0x0598, 0x0599, 0x059C, 0x059D, 0x059E, 0x059F, 0x05A0, 0x05A1, 0x05A8, 0x05A9,
	0x05AB, 0x05AC, 0x05AF, 0x05C4, 0x0610, 0x0611, 0x0612, 0x0613, 0x0614, 0x0615,
	0x0616, 0x0617, 0x0657, 0x0658, 0x0659, 0x065A, 0x065B, 0x065D, 0x065E, 0x06D6,
	0x06D7, 0x06D8, 0x06D9, 0x06DA, 0x06DB, 0x06DC, 0x06DF, 0x06E0, 0x06E1, 0x06E2,
	0x06E4, 0x06E7, 0x06E8, 0x06EB, 0x06EC, 0x0730, 0x0732, 0x0733, 0x0735, 0x0736,
	0x073A, 0x073D, 0x073F, 0x0740, 0x0741, 0x0743, 0x0745, 0x0747, 0x0749, 0x074A,
	0x07EB, 0x07EC, 0x07ED, 0x07EE, 0x07EF, 0x07F0, 0x07F1, 0x07F3, 0x0816, 0x0817,
	0x0818, 0x0819, 0x081B, 0x081C, 0x081D, 0x081E, 0x081F, 0x0820, 0x0821, 0x0822,
	0x0823, 0x0825, 0x0826, 0x0827, 0x0829, 0x082A, 0x082B, 0x082C, 0x082D, 0x0951,
	0x0953, 0x0954, 0x0F82, 0x0F83, 0x0F86, 0x0F87, 0x135D, 0x135E, 0x135F, 0x17DD,
	0x193A, 0x1A17, 0x1A75, 0x1A76, 0x1A77, 0x1A78, 0x1A79, 0x1A7A, 0x1A7B, 0x1A7C,
	0x1B6B, 0x1B6D, 0x1B6E, 0x1B6F, 0x1B70, 0x1B71, 0x1B72, 0x1B73, 0x1CD0, 0x1CD1,
	0x1CD2, 0x1CDA, 0x1CDB, 0x1CE0, 0x1DC0, 0x1DC1, 0x1DC3, 0x1DC4, 0x1DC5, 0x1DC6,
	0x1DC7, 0x1DC8, 0x1DC9, 0x1DCB, 0x1DCC, 0x1DD1, 0x1DD2, 0x1DD3, 0x1DD4, 0x1DD5,
	0x1DD6, 0x1DD7, 0x1DD8, 0x1DD9, 0x1DDA, 0x1DDB, 0x1DDC, 0x1DDD, 0x1DDE, 0x1DDF,
	0x1DE0, 0x1DE1, 0x1DE2, 0x1DE3, 0x1DE4, 0x1DE5, 0x1DE6, 0x1DFE, 0x20D0, 0x20D1,
	0x20D4, 0x20D5, 0x20D6, 0x20D7, 0x20DB, 0x20DC, 0x20E1, 0x20E7, 0x20E9, 0x20F0,
	0x2CEF, 0x2CF0, 0x2CF1, 0x2DE0, 0x2DE1, 0x2DE2, 0x2DE3, 0x2DE4, 0x2DE5, 0x2DE6,
	0x2DE7, 0x2DE8, 0x2DE9, 0x2DEA, 0x2DEB, 0x2DEC, 0x2DED, 0x2DEE, 0x2DEF, 0x2DF0,
	0x2DF1, 0x2DF2, 0x2DF3, 0x2DF4, 0x2DF5, 0x2DF6, 0x2DF7, 0x2DF8, 0x2DF9, 0x2DFA,
	0x2DFB, 0x2DFC, 0x2DFD, 0x2DFE, 0x2DFF, 0xA66F, 0xA67C, 0xA67D, 0xA6F0, 0xA6F1,
	0xA8E0, 0xA8E1, 0xA8E2, 0xA8E3, 0xA8E4, 0xA8E5, 0xA8E6, 0xA8E7, 0xA8E8, 0xA8E9,
	0xA8EA, 0xA8EB, 0xA8EC, 0xA8ED, 0xA8EE, 0xA8EF, 0xA8F0, 0xA8F1, 0xAAB0, 0xAAB2,
	0xAAB3, 0xAAB7, 0xAAB8, 0xAABE, 0xAABF, 0xAAC1, 0xFE20, 0xFE21, 0xFE22, 0xFE23,
	0xFE24, 0xFE25, 0xFE26, 0x10A0F, 0x10A38, 0x1D185, 0x1D186, 0x1D187, 0x1D188,
	0x1D189, 0x1D1AA, 0x1D1AB, 0x1D1AC, 0x1D1AD, 0x1D242, 0x1D243, 0x1D244,
}

// KittyDiacritic returns the combining rune corresponding to integer index [0..296].
func KittyDiacritic(n int) rune {
	if n < 0 || n >= len(kittyDiacritics) {
		return kittyDiacritics[0]
	}
	return kittyDiacritics[n]
}

const KittyPlaceholderRune rune = 0x10EEEE

// FormatKittyTransmit builds the chunked Kitty Graphics Protocol virtual placement sequence
// for the given RGBA image, matching ratatui-image's proven implementation.
func FormatKittyTransmit(img image.Image, id uint32, isTmux bool) []byte {
	if img == nil {
		return nil
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return nil
	}

	// Convert image to raw RGBA bytes (f=32)
	rgba, ok := img.(*image.RGBA)
	if !ok {
		rgba = image.NewRGBA(bounds)
		draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	}
	rawBytes := rgba.Pix

	// Chunk raw bytes: max 4096 base64 chars per chunk -> (4096 / 4) * 3 = 3072 raw bytes
	const chunkSize = 3072
	totalChunks := (len(rawBytes) + chunkSize - 1) / chunkSize
	if totalChunks == 0 {
		totalChunks = 1
	}

	var buf bytes.Buffer
	for i := 0; i < totalChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(rawBytes) {
			end = len(rawBytes)
		}
		chunk := rawBytes[start:end]
		encoded := base64.StdEncoding.EncodeToString(chunk)

		more := 1
		if i == totalChunks-1 {
			more = 0
		}

		var payload bytes.Buffer
		if isTmux {
			payload.WriteString("\x1bPtmux;\x1b\x1b_Gq=2,")
		} else {
			payload.WriteString("\x1b_Gq=2,")
		}

		if i == 0 {
			payload.WriteString(fmt.Sprintf("i=%d,a=T,U=1,f=32,t=d,s=%d,v=%d,", id, w, h))
		}
		payload.WriteString(fmt.Sprintf("m=%d;%s", more, encoded))

		if isTmux {
			payload.WriteString("\x1b\\\x1b\\")
		} else {
			payload.WriteString("\x1b\\")
		}

		buf.Write(payload.Bytes())
	}

	return buf.Bytes()
}

// CreateKittyPlaceholderBuffer generates a 2D grid of rows lines x cols cells
// filled with Kitty Unicode Placeholder characters (0x10EEEE) and diacritics.
// The foreground color of every cell encodes the 24-bit image ID.
func CreateKittyPlaceholderBuffer(cols, rows int, id uint32) []tstring.TString {
	if cols <= 0 || rows <= 0 {
		return nil
	}

	r := int32((id >> 16) & 0xFF)
	g := int32((id >> 8) & 0xFF)
	b := int32(id & 0xFF)
	idExtra := int((id >> 24) & 0xFF)

	cellStyle := tcell.StyleDefault.Foreground(tcell.NewRGBColor(r, g, b))

	buf := make([]tstring.TString, rows)
	for y := 0; y < rows; y++ {
		line := make(tstring.TString, cols)
		rowDiacritic := KittyDiacritic(y)
		extraDiacritic := KittyDiacritic(idExtra)

		for x := 0; x < cols; x++ {
			comb := []rune{rowDiacritic, KittyDiacritic(x), extraDiacritic}
			line[x] = tstring.NewCombCell(KittyPlaceholderRune, comb, cellStyle)
		}
		buf[y] = line
	}
	return buf
}
