// gomuks - A Matrix client written in Go.
// Copyright (C) 2025 Tulir Asokan
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

//go:build cgo

package gomuks

import (
	"image"
	"io"

	cwebp "go.mau.fi/webp"
)

func init() {
	encodeAvatarThumbnail = func(writer io.Writer, img image.Image) error {
		return cwebp.Encode(writer, img, &cwebp.Options{Quality: 80})
	}
	encodeWebp = func(writer io.Writer, img image.Image, quality float32, lossless bool) error {
		return cwebp.Encode(writer, img, &cwebp.Options{
			Lossless: lossless,
			Quality:  quality,
		})
	}
}
