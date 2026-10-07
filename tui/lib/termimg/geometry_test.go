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
)

func TestCalculateClampedDimensions(t *testing.T) {
	cases := []struct {
		name          string
		imgW, imgH    int
		availableCols int
		maxCols       int
		maxRows       int
		expectedCols  int
		expectedRows  int
	}{
		{
			name:          "square image 1000x1000 clamped to 32x16",
			imgW:          1000,
			imgH:          1000,
			availableCols: 80,
			maxCols:       66,
			maxRows:       16,
			expectedCols:  32,
			expectedRows:  16,
		},
		{
			name:          "16:9 landscape 1920x1080 clamped to 57x16",
			imgW:          1920,
			imgH:          1080,
			availableCols: 80,
			maxCols:       66,
			maxRows:       16,
			expectedCols:  57,
			expectedRows:  16,
		},
		{
			name:          "extreme panorama 2000x100 clamped with minHeight 3",
			imgW:          2000,
			imgH:          100,
			availableCols: 80,
			maxCols:       66,
			maxRows:       16,
			expectedCols:  66,
			expectedRows:  3,
		},
		{
			name:          "tall portrait 1080x2400 clamped to 14x16",
			imgW:          1080,
			imgH:          2400,
			availableCols: 80,
			maxCols:       66,
			maxRows:       16,
			expectedCols:  14,
			expectedRows:  16,
		},
		{
			name:          "small thumbnail 40x20 preserved without upscaling",
			imgW:          40,
			imgH:          20,
			availableCols: 80,
			maxCols:       66,
			maxRows:       16,
			expectedCols:  40,
			expectedRows:  10,
		},
		{
			name:          "tiny image 2x1 not forced to minHeight 3",
			imgW:          2,
			imgH:          1,
			availableCols: 80,
			maxCols:       66,
			maxRows:       16,
			expectedCols:  2,
			expectedRows:  1,
		},
		{
			name:          "tiny square 2x2 produces 1 row preserving 1:1 aspect ratio",
			imgW:          2,
			imgH:          2,
			availableCols: 80,
			maxCols:       66,
			maxRows:       16,
			expectedCols:  2,
			expectedRows:  1,
		},
		{
			name:          "tiny image 1x1 produces 1 row",
			imgW:          1,
			imgH:          1,
			availableCols: 80,
			maxCols:       66,
			maxRows:       16,
			expectedCols:  1,
			expectedRows:  1,
		},
		{
			name:          "tiny rectangular 4x2 produces 1 row",
			imgW:          4,
			imgH:          2,
			availableCols: 80,
			maxCols:       66,
			maxRows:       16,
			expectedCols:  4,
			expectedRows:  1,
		},
		{
			name:          "small square 4x4 produces 2 rows",
			imgW:          4,
			imgH:          4,
			availableCols: 80,
			maxCols:       66,
			maxRows:       16,
			expectedCols:  4,
			expectedRows:  2,
		},
		{
			name:          "narrow screen availableCols 30 clamps cols to 30",
			imgW:          1920,
			imgH:          1080,
			availableCols: 30,
			maxCols:       66,
			maxRows:       16,
			expectedCols:  30,
			expectedRows:  8,
		},
		{
			name:          "zero image width returns zero bounding box",
			imgW:          0,
			imgH:          100,
			availableCols: 80,
			maxCols:       66,
			maxRows:       16,
			expectedCols:  0,
			expectedRows:  0,
		},
		{
			name:          "negative image height returns zero bounding box",
			imgW:          100,
			imgH:          -5,
			availableCols: 80,
			maxCols:       66,
			maxRows:       16,
			expectedCols:  0,
			expectedRows:  0,
		},
		{
			name:          "defaults applied when maxCols and maxRows <= 0",
			imgW:          1000,
			imgH:          1000,
			availableCols: 100,
			maxCols:       0,
			maxRows:       0,
			expectedCols:  32,
			expectedRows:  16,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			box := CalculateClampedDimensions(tc.imgW, tc.imgH, tc.availableCols, tc.maxCols, tc.maxRows)
			if box.Cols != tc.expectedCols || box.Rows != tc.expectedRows {
				t.Errorf("CalculateClampedDimensions(%d, %d, %d, %d, %d) = {Cols: %d, Rows: %d}, expected {Cols: %d, Rows: %d}",
					tc.imgW, tc.imgH, tc.availableCols, tc.maxCols, tc.maxRows,
					box.Cols, box.Rows, tc.expectedCols, tc.expectedRows)
			}
		})
	}
}
