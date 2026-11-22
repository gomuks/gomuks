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
	"errors"
	"fmt"
	"image"
	"io"

	// Register image decoders for header decoding
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)

const (
	// MaxMegaPixels defines the maximum allowed image dimension in megapixels.
	MaxMegaPixels = 16
	// MaxPixels defines the maximum allowed total pixel count (16,000,000 pixels).
	MaxPixels = MaxMegaPixels * 1000 * 1000
)

var (
	// ErrImageTooLarge is returned when image dimensions exceed MaxPixels.
	ErrImageTooLarge = errors.New("image exceeds maximum allowed size of 16 megapixels")
	// ErrInvalidImage is returned when image header cannot be parsed or has non-positive dimensions.
	ErrInvalidImage = errors.New("invalid image")
)

// ValidateImageSafety reads image configuration without decoding pixel buffers.
// It verifies that image dimensions do not exceed MaxPixels (16 Megapixels),
// preventing decompression bomb Denial-of-Service attacks.
func ValidateImageSafety(r io.Reader) (image.Config, error) {
	cfg, _, err := image.DecodeConfig(r)
	if err != nil {
		return image.Config{}, fmt.Errorf("%w: failed to decode image config: %v", ErrInvalidImage, err)
	}

	if cfg.Width <= 0 || cfg.Height <= 0 {
		return cfg, fmt.Errorf("%w: non-positive dimensions %dx%d", ErrInvalidImage, cfg.Width, cfg.Height)
	}

	// Guard against integer overflow and total pixel count > MaxPixels
	if cfg.Width > MaxPixels || cfg.Height > MaxPixels || cfg.Width > MaxPixels/cfg.Height {
		return cfg, fmt.Errorf("%w: %dx%d (%d pixels, max %d)", ErrImageTooLarge, cfg.Width, cfg.Height, cfg.Width*cfg.Height, MaxPixels)
	}

	return cfg, nil
}
