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
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func createTestPNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func createTestJPEG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}

// createCraftedPNGHeader creates a valid PNG header with arbitrary width and height.
func createCraftedPNGHeader(width, height uint32) []byte {
	var buf bytes.Buffer
	// PNG Signature (8 bytes)
	buf.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})

	// IHDR chunk: length (13 bytes), chunk type "IHDR", chunk data, CRC
	ihdrData := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdrData[0:4], width)
	binary.BigEndian.PutUint32(ihdrData[4:8], height)
	ihdrData[8] = 8  // Bit depth
	ihdrData[9] = 2  // Color type (Truecolor)
	ihdrData[10] = 0 // Compression method (deflate)
	ihdrData[11] = 0 // Filter method
	ihdrData[12] = 0 // Interlace method

	// Chunk length
	var lenBytes [4]byte
	binary.BigEndian.PutUint32(lenBytes[:], 13)
	buf.Write(lenBytes[:])

	// Chunk type + data
	chunkType := []byte("IHDR")
	buf.Write(chunkType)
	buf.Write(ihdrData)

	// Calculate CRC32 for chunk type + data
	crc := crc32.NewIEEE()
	crc.Write(chunkType)
	crc.Write(ihdrData)
	var crcBytes [4]byte
	binary.BigEndian.PutUint32(crcBytes[:], crc.Sum32())
	buf.Write(crcBytes[:])

	return buf.Bytes()
}

func TestValidateImageSafety_ValidImages(t *testing.T) {
	t.Run("valid PNG", func(t *testing.T) {
		data := createTestPNG(320, 240)
		cfg, err := ValidateImageSafety(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("expected valid image, got error: %v", err)
		}
		if cfg.Width != 320 || cfg.Height != 240 {
			t.Errorf("expected 320x240, got %dx%d", cfg.Width, cfg.Height)
		}
	})

	t.Run("valid JPEG", func(t *testing.T) {
		data := createTestJPEG(1920, 1080)
		cfg, err := ValidateImageSafety(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("expected valid image, got error: %v", err)
		}
		if cfg.Width != 1920 || cfg.Height != 1080 {
			t.Errorf("expected 1920x1080, got %dx%d", cfg.Width, cfg.Height)
		}
	})

	t.Run("boundary image at exactly 16 Megapixels", func(t *testing.T) {
		// 4000 x 4000 = 16,000,000 pixels (MaxPixels)
		data := createCraftedPNGHeader(4000, 4000)
		cfg, err := ValidateImageSafety(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("expected boundary image to pass, got error: %v", err)
		}
		if cfg.Width != 4000 || cfg.Height != 4000 {
			t.Errorf("expected 4000x4000, got %dx%d", cfg.Width, cfg.Height)
		}
	})
}

func TestValidateImageSafety_DecompressionBombs(t *testing.T) {
	t.Run("oversized 400MP image (20000x20000)", func(t *testing.T) {
		// 20000 x 20000 = 400,000,000 pixels (> 16MP)
		data := createCraftedPNGHeader(20000, 20000)
		_, err := ValidateImageSafety(bytes.NewReader(data))
		if err == nil {
			t.Fatalf("expected decompression bomb to be rejected, got nil error")
		}
		if !errors.Is(err, ErrImageTooLarge) {
			t.Errorf("expected ErrImageTooLarge, got %v", err)
		}
	})

	t.Run("just over 16MP (4001x4000 = 16004000 pixels)", func(t *testing.T) {
		data := createCraftedPNGHeader(4001, 4000)
		_, err := ValidateImageSafety(bytes.NewReader(data))
		if err == nil {
			t.Fatalf("expected oversized image to be rejected, got nil error")
		}
		if !errors.Is(err, ErrImageTooLarge) {
			t.Errorf("expected ErrImageTooLarge, got %v", err)
		}
	})

	t.Run("integer overflow dimension (width > MaxPixels)", func(t *testing.T) {
		data := createCraftedPNGHeader(0x7fffffff, 2)
		_, err := ValidateImageSafety(bytes.NewReader(data))
		if err == nil {
			t.Fatalf("expected integer overflow image to be rejected, got nil error")
		}
		if !errors.Is(err, ErrImageTooLarge) {
			t.Errorf("expected ErrImageTooLarge, got %v", err)
		}
	})
}

func TestValidateImageSafety_CorruptAndInvalid(t *testing.T) {
	t.Run("empty payload", func(t *testing.T) {
		_, err := ValidateImageSafety(bytes.NewReader(nil))
		if err == nil {
			t.Fatalf("expected empty reader to fail, got nil error")
		}
		if !errors.Is(err, ErrInvalidImage) {
			t.Errorf("expected ErrInvalidImage, got %v", err)
		}
	})

	t.Run("garbage payload", func(t *testing.T) {
		_, err := ValidateImageSafety(bytes.NewReader([]byte("not a real image payload")))
		if err == nil {
			t.Fatalf("expected garbage reader to fail, got nil error")
		}
		if !errors.Is(err, ErrInvalidImage) {
			t.Errorf("expected ErrInvalidImage, got %v", err)
		}
	})

	t.Run("truncated header", func(t *testing.T) {
		header := createCraftedPNGHeader(100, 100)
		_, err := ValidateImageSafety(bytes.NewReader(header[:10]))
		if err == nil {
			t.Fatalf("expected truncated header to fail, got nil error")
		}
		if !errors.Is(err, ErrInvalidImage) {
			t.Errorf("expected ErrInvalidImage, got %v", err)
		}
	})
}
