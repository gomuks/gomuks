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
	"math"
)

const (
	// DefaultMaxWidth is the default column boundary for image previews (iamb parity).
	DefaultMaxWidth = 66
	// DefaultMaxHeight is the default row boundary for image previews (iamb parity).
	DefaultMaxHeight = 16
	// MinHeight is the minimum vertical row allocation for panoramic images.
	MinHeight = 3
)

// BoundingBox represents character cell dimensions (columns and rows).
type BoundingBox struct {
	Cols int
	Rows int
}

// CalculateClampedDimensions computes proportional 2D character cell dimensions for an image.
// Terminal font cells typically have ~1:2 width-to-height aspect ratio.
// For half-block cells (2 vertical sub-pixels per cell), effective cell aspect ratio is (W / H) * 2.0.
// Dimensions are clamped within [min(availableCols, maxCols), maxRows], maintaining at least MinHeight rows
// (unless nativeRows < MinHeight).
func CalculateClampedDimensions(imgW, imgH, availableCols, maxCols, maxRows int) BoundingBox {
	if imgW <= 0 || imgH <= 0 {
		return BoundingBox{Cols: 0, Rows: 0}
	}

	if maxCols <= 0 {
		maxCols = DefaultMaxWidth
	}
	if maxRows <= 0 {
		maxRows = DefaultMaxHeight
	}
	if availableCols <= 0 {
		availableCols = maxCols
	}

	limitCols := availableCols
	if maxCols < limitCols {
		limitCols = maxCols
	}
	if limitCols < 1 {
		limitCols = 1
	}

	cellAR := (float64(imgW) / float64(imgH)) * 2.0

	// Character cell dimensions at 1:1 scale (1 column per pixel, 2 pixels per row)
	nativeCols := imgW
	nativeRows := int(math.Round(float64(imgH) / 2.0))
	if nativeRows < 1 {
		nativeRows = 1
	}

	var cols, rows int
	if nativeCols <= limitCols && nativeRows <= maxRows {
		cols = nativeCols
		rows = nativeRows
	} else {
		// Fit within bounding box limitCols x maxRows
		cols = limitCols
		rows = int(math.Round(float64(limitCols) / cellAR))
		if rows > maxRows {
			rows = maxRows
			cols = int(math.Round(float64(maxRows) * cellAR))
		}
		if cols > limitCols {
			cols = limitCols
		}
	}

	// Enforce MinHeight (unless the source image itself has fewer native rows than MinHeight)
	minH := MinHeight
	if nativeRows < minH {
		minH = nativeRows
	}
	if minH < 1 {
		minH = 1
	}

	if rows < minH {
		rows = minH
	}
	if rows > maxRows {
		rows = maxRows
	}
	if cols > limitCols {
		cols = limitCols
	}
	if cols < 1 {
		cols = 1
	}

	return BoundingBox{
		Cols: cols,
		Rows: rows,
	}
}
