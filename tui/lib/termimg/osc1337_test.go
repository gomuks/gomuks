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
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func TestFormatOSC1337_Standard(t *testing.T) {
	data := []byte("fake-image-bytes-for-testing")
	cols := 40
	rows := 12

	seq := FormatOSC1337(data, cols, rows, false)
	seqStr := string(seq)

	prefix := fmt.Sprintf("\x1b]1337;File=inline=1;width=%d;height=%d;preserveAspectRatio=1:", cols, rows)
	if !strings.HasPrefix(seqStr, prefix) {
		t.Errorf("expected sequence to start with %q, got %q", prefix, seqStr)
	}

	suffix := "\x07"
	if !strings.HasSuffix(seqStr, suffix) {
		t.Errorf("expected sequence to end with %q, got %q", suffix, seqStr)
	}

	b64Part := strings.TrimSuffix(strings.TrimPrefix(seqStr, prefix), suffix)
	decoded, err := base64.StdEncoding.DecodeString(b64Part)
	if err != nil {
		t.Fatalf("failed to decode base64 part: %v", err)
	}

	if !bytes.Equal(decoded, data) {
		t.Errorf("decoded base64 does not match original data")
	}
}

func TestFormatOSC1337_TmuxDCS(t *testing.T) {
	data := []byte("fake-image-bytes-for-testing")
	cols := 50
	rows := 15

	seq := FormatOSC1337(data, cols, rows, true)
	seqStr := string(seq)

	prefix := fmt.Sprintf("\x1bPtmux;\x1b\x1b]1337;File=inline=1;width=%d;height=%d;preserveAspectRatio=1:", cols, rows)
	if !strings.HasPrefix(seqStr, prefix) {
		t.Errorf("expected tmux DCS sequence to start with %q, got %q", prefix, seqStr)
	}

	suffix := "\x07\x1b\\"
	if !strings.HasSuffix(seqStr, suffix) {
		t.Errorf("expected tmux DCS sequence to end with %q, got %q", suffix, seqStr)
	}

	b64Part := strings.TrimSuffix(strings.TrimPrefix(seqStr, prefix), suffix)
	decoded, err := base64.StdEncoding.DecodeString(b64Part)
	if err != nil {
		t.Fatalf("failed to decode base64 payload: %v", err)
	}

	if !bytes.Equal(decoded, data) {
		t.Errorf("decoded payload does not match original data")
	}
}

func TestPreDownscaleForPTY(t *testing.T) {
	t.Run("image within bounds is not modified", func(t *testing.T) {
		data := createTestPNG(800, 600)
		out, err := PreDownscaleForPTY(data, 1920, 1080)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(out, data) {
			t.Errorf("expected output to be identical to original bytes for small image")
		}
	})

	t.Run("image exceeding bounds is downscaled", func(t *testing.T) {
		// Create 2400x1600 image
		img := image.NewRGBA(image.Rect(0, 0, 2400, 1600))
		img.Set(0, 0, color.RGBA{R: 255, A: 255})
		var buf bytes.Buffer
		_ = png.Encode(&buf, img)
		data := buf.Bytes()

		out, err := PreDownscaleForPTY(data, 1920, 1080)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("failed to decode downscaled image config: %v", err)
		}

		if cfg.Width > 1920 || cfg.Height > 1080 {
			t.Errorf("expected downscaled image <= 1920x1080, got %dx%d", cfg.Width, cfg.Height)
		}
	})

	t.Run("invalid image data returns error", func(t *testing.T) {
		_, err := PreDownscaleForPTY([]byte("corrupt-bytes"), 1920, 1080)
		if err == nil {
			t.Fatalf("expected error on corrupt input, got nil")
		}
	})

	t.Run("decompression bombs exceeding 16MP are rejected before decoding", func(t *testing.T) {
		// 400MP payload (20000x20000)
		bomb := createCraftedPNGHeader(20000, 20000)
		_, err := PreDownscaleForPTY(bomb, 1920, 1080)
		if err == nil {
			t.Fatalf("expected 400MP decompression bomb to be rejected, got nil error")
		}
		if !errors.Is(err, ErrImageTooLarge) {
			t.Errorf("expected error to wrap ErrImageTooLarge, got: %v", err)
		}

		// Just over 16MP (4001x4000 = 16,004,000 pixels)
		bomb2 := createCraftedPNGHeader(4001, 4000)
		_, err = PreDownscaleForPTY(bomb2, 1920, 1080)
		if err == nil {
			t.Fatalf("expected 16.004MP image to be rejected, got nil error")
		}
		if !errors.Is(err, ErrImageTooLarge) {
			t.Errorf("expected error to wrap ErrImageTooLarge, got: %v", err)
		}
	})

	t.Run("transparent PNG preserves alpha transparency when downscaled", func(t *testing.T) {
		// Create 2400x1600 RGBA image with transparent and colored regions
		img := image.NewRGBA(image.Rect(0, 0, 2400, 1600))
		// Leave (0, 0) as transparent (0, 0, 0, 0)
		// Set (10, 10) to translucent red
		img.Set(10, 10, color.NRGBA{R: 255, G: 0, B: 0, A: 128})
		// Set (20, 20) to opaque green
		img.Set(20, 20, color.NRGBA{R: 0, G: 255, B: 0, A: 255})

		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			t.Fatalf("failed to encode test png: %v", err)
		}
		data := buf.Bytes()

		out, err := PreDownscaleForPTY(data, 1200, 800)
		if err != nil {
			t.Fatalf("unexpected error downscaling transparent PNG: %v", err)
		}

		// Verify output is encoded as PNG, not JPEG
		decoded, format, err := image.Decode(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("failed to decode downscaled image: %v", err)
		}
		if format != "png" {
			t.Errorf("expected downscaled format to be %q, got %q", "png", format)
		}

		bounds := decoded.Bounds()
		if bounds.Dx() > 1200 || bounds.Dy() > 800 {
			t.Errorf("expected dimensions <= 1200x800, got %dx%d", bounds.Dx(), bounds.Dy())
		}

		// Verify alpha channel is preserved (image still has non-opaque pixels)
		if !hasAlpha(decoded) {
			t.Errorf("expected downscaled image to preserve alpha transparency, but hasAlpha was false")
		}

		// Verify top-left corner remains fully transparent
		_, _, _, a := decoded.At(0, 0).RGBA()
		if a != 0 {
			t.Errorf("expected pixel at (0,0) to have alpha 0, got %d", a)
		}
	})

	t.Run("opaque JPEG is downscaled and re-encoded as JPEG", func(t *testing.T) {
		img := image.NewRGBA(image.Rect(0, 0, 2400, 1600))
		for y := 0; y < 1600; y += 100 {
			for x := 0; x < 2400; x += 100 {
				img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
			}
		}

		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
			t.Fatalf("failed to encode test jpeg: %v", err)
		}
		data := buf.Bytes()

		out, err := PreDownscaleForPTY(data, 1200, 800)
		if err != nil {
			t.Fatalf("unexpected error downscaling JPEG: %v", err)
		}

		_, format, err := image.Decode(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("failed to decode downscaled image: %v", err)
		}
		if format != "jpeg" {
			t.Errorf("expected downscaled format to be %q, got %q", "jpeg", format)
		}
	})
}
