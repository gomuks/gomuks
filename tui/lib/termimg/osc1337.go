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
	_ "image/gif"
	"image/jpeg"
	"image/png"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)

const (
	// DefaultMaxPTYWidth is the default max pixel width before pre-downscaling for PTY.
	DefaultMaxPTYWidth = 1920
	// DefaultMaxPTYHeight is the default max pixel height before pre-downscaling for PTY.
	DefaultMaxPTYHeight = 1080
)

// FormatOSC1337 generates the iTerm2 OSC 1337 inline file transfer escape sequence.
// If isTmux is true, the sequence is wrapped in tmux DCS passthrough escape sequences (\x1bPtmux;\x1b...).
func FormatOSC1337(data []byte, cols, rows int, isTmux bool) []byte {
	b64 := base64.StdEncoding.EncodeToString(data)

	if isTmux {
		// Tmux DCS passthrough: \x1bPtmux;\x1b<escaped_seq>\x1b\
		// Any \x1b byte within <escaped_seq> must be doubled (\x1b\x1b).
		return []byte(fmt.Sprintf("\x1bPtmux;\x1b\x1b]1337;File=inline=1;width=%d;height=%d;preserveAspectRatio=1:%s\x07\x1b\\", cols, rows, b64))
	}

	// Standard iTerm2 OSC 1337 format
	return []byte(fmt.Sprintf("\x1b]1337;File=inline=1;width=%d;height=%d;preserveAspectRatio=1:%s\x07", cols, rows, b64))
}

// hasAlpha reports whether the image contains any non-opaque (transparent or translucent) pixels.
func hasAlpha(img image.Image) bool {
	if nrgba, ok := img.(*image.NRGBA); ok {
		for i := 3; i < len(nrgba.Pix); i += 4 {
			if nrgba.Pix[i] < 255 {
				return true
			}
		}
		return false
	}
	if rgba, ok := img.(*image.RGBA); ok {
		for i := 3; i < len(rgba.Pix); i += 4 {
			if rgba.Pix[i] < 255 {
				return true
			}
		}
		return false
	}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a < 0xffff {
				return true
			}
		}
	}
	return false
}

// PreDownscaleForPTY downscales giant images (>1920x1080 by default) before base64 encoding
// to prevent PTY buffer congestion and terminal latency over SSH / serial links.
// Image safety is validated first to reject decompression bombs (>16MP) before decoding.
// Alpha transparency is preserved by encoding transparent images or PNGs as PNG.
func PreDownscaleForPTY(imgData []byte, maxW, maxH int) ([]byte, error) {
	if maxW <= 0 {
		maxW = DefaultMaxPTYWidth
	}
	if maxH <= 0 {
		maxH = DefaultMaxPTYHeight
	}

	cfg, err := ValidateImageSafety(bytes.NewReader(imgData))
	if err != nil {
		return nil, fmt.Errorf("pre-downscale safety validation failed: %w", err)
	}

	// If image already fits within bounds, return original bytes without re-encoding
	if cfg.Width <= maxW && cfg.Height <= maxH {
		return imgData, nil
	}

	img, format, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image for PTY downscaling: %w", err)
	}

	resized := imaging.Fit(img, maxW, maxH, imaging.Lanczos)

	var buf bytes.Buffer
	if format == "png" || hasAlpha(resized) {
		err = png.Encode(&buf, resized)
	} else {
		err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 85})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to re-encode downscaled image: %w", err)
	}

	return buf.Bytes(), nil
}
