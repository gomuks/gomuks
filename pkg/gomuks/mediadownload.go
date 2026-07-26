// gomuks - A Matrix client written in Go.
// Copyright (C) 2024 Tulir Asokan
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

package gomuks

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"image"
	"image/color"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gabriel-vasile/mimetype"
	"github.com/rs/zerolog"
	"go.mau.fi/util/jsontime"
	"go.mau.fi/util/ptr"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/attachment"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/orientation"
)

var ErrBadGateway = mautrix.RespError{
	ErrCode:    "FI.MAU.GOMUKS.BAD_GATEWAY",
	StatusCode: http.StatusBadGateway,
}

func (gmx *Gomuks) downloadMediaFromCache(ctx context.Context, w http.ResponseWriter, r *http.Request, entry *database.Media, force, useThumbnail bool) bool {
	if ctx.Err() != nil {
		return true
	}
	if !entry.UseCache() {
		if force {
			mautrix.MNotFound.WithMessage("Media not found in cache").Write(w)
			return true
		}
		return false
	}
	etag := entry.ETag(useThumbnail)
	if entry.Error != nil {
		w.Header().Set("Mau-Cached-Error", "true")
		entry.Error.Write(w)
		return true
	} else if etag != "" && r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return true
	} else if entry.MimeType != "" && r.URL.Query().Has("fallback") && !isAllowedAvatarMime(entry.MimeType) {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return true
	}
	log := zerolog.Ctx(ctx)
	hash := entry.Hash
	if useThumbnail {
		if entry.ThumbnailError != "" {
			log.Debug().Str(zerolog.ErrorFieldName, entry.ThumbnailError).Msg("Returning cached thumbnail error")
			w.WriteHeader(http.StatusInternalServerError)
			return true
		}
		if entry.ThumbnailHash == nil {
			err := gmx.generateAvatarThumbnail(entry, gmx.Config.Media.ThumbnailSize)
			if errors.Is(err, os.ErrNotExist) && !force {
				return false
			} else if err != nil {
				log.Err(err).Msg("Failed to generate avatar thumbnail")
				gmx.saveMediaCacheEntryWithThumbnail(ctx, entry, err)
				w.WriteHeader(http.StatusInternalServerError)
				return true
			} else {
				gmx.saveMediaCacheEntryWithThumbnail(ctx, entry, nil)
			}
		}
		hash = entry.ThumbnailHash
	}
	cacheFile, err := os.Open(gmx.cacheEntryToPath(hash[:]))
	if useThumbnail && errors.Is(err, os.ErrNotExist) {
		err = gmx.generateAvatarThumbnail(entry, gmx.Config.Media.ThumbnailSize)
		if errors.Is(err, os.ErrNotExist) && !force {
			return false
		} else if err != nil {
			log.Err(err).Msg("Failed to generate avatar thumbnail")
			gmx.saveMediaCacheEntryWithThumbnail(ctx, entry, err)
			w.WriteHeader(http.StatusInternalServerError)
			return true
		} else {
			gmx.saveMediaCacheEntryWithThumbnail(ctx, entry, nil)
			cacheFile, err = os.Open(gmx.cacheEntryToPath(hash[:]))
		}
	}
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !force {
			return false
		}
		log.Err(err).Msg("Failed to open cache file")
		mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to open cache file: %v", err)).Write(w)
		return true
	}
	defer func() {
		_ = cacheFile.Close()
	}()
	cacheEntryToHeaders(w, entry, useThumbnail)
	http.ServeContent(w, r, "", time.Time{}, cacheFile)
	return true
}

func (gmx *Gomuks) cacheEntryToPath(hash []byte) string {
	hashPath := hex.EncodeToString(hash[:])
	return filepath.Join(gmx.CacheDir, "media", hashPath[0:2], hashPath[2:4], hashPath[4:])
}

func cacheEntryToHeaders(w http.ResponseWriter, entry *database.Media, thumbnail bool) {
	if thumbnail {
		w.Header().Set("Content-Type", "image/webp")
		w.Header().Set("Content-Length", strconv.FormatInt(entry.ThumbnailSize, 10))
		w.Header().Set("Content-Disposition", "inline; filename=thumbnail.webp")
	} else {
		w.Header().Set("Content-Type", entry.MimeType)
		w.Header().Set("Content-Length", strconv.FormatInt(entry.Size, 10))
		w.Header().Set("Content-Disposition", mime.FormatMediaType(entry.ContentDisposition(), map[string]string{"filename": entry.FileName}))
	}
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; script-src 'none'; media-src 'self';")
	w.Header().Set("Cache-Control", "max-age=2592000, immutable")
	w.Header().Set("ETag", entry.ETag(thumbnail))
}

func (gmx *Gomuks) saveMediaCacheEntryWithThumbnail(ctx context.Context, entry *database.Media, err error) {
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		entry.ThumbnailError = err.Error()
	}
	err = gmx.Client.DB.Media.Put(ctx, entry)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("Failed to save cache entry after generating thumbnail")
	}
}

func BytesPerPixel(cm color.Model) int {
	switch cm {
	case color.GrayModel:
		return 1
	case color.Gray16Model:
		return 2
	case color.YCbCrModel:
		return 3
	case color.RGBAModel, color.NRGBAModel, color.CMYKModel, color.NYCbCrAModel:
		return 4
	case color.RGBA64Model, color.NRGBA64Model:
		return 8
	default:
		return 16
	}
}

func decodeImageWithOrientationFix(file *os.File, maxDecodeMemory int) (image.Image, error) {
	cfg, decodedFrom, err := image.DecodeConfig(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image config: %w", err)
	} else if cfg.Width > 20000 || cfg.Height > 20000 || cfg.Width*cfg.Height*BytesPerPixel(cfg.ColorModel) > maxDecodeMemory {
		return nil, fmt.Errorf("image dimensions too large: %dx%d (color model: %T)", cfg.Width, cfg.Height, cfg.ColorModel)
	}
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to start of temp file: %w", err)
	}
	decoded, _, err := image.Decode(file)
	if err != nil {
		if decodedFrom == "webp" {
			_, err = file.Seek(0, io.SeekStart)
			if err != nil {
				return nil, fmt.Errorf("failed to seek to start of temp file: %w", err)
			}
			decoded, err = decodeAnimatedWebp(file)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to decode image: %w", err)
		}
	}
	var o orientation.Orientation
	if decodedFrom == "jpeg" {
		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("failed to seek to start of temp file: %w", err)
		}
		o = orientation.Read(file)
	} else if decodedFrom == "heic" {
		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("failed to seek to start of temp file: %w", err)
		}
		exif, err := parseHEICEXIF(file)
		if err != nil {
			return nil, fmt.Errorf("failed to parse HEIC EXIF: %w", err)
		}
		o = orientation.ReadEXIF(bytes.NewReader(exif))
	}
	if o != orientation.Unspecified {
		decoded = o.Fix(decoded)
	}
	return decoded, nil
}

var encodeAvatarThumbnail = func(writer io.Writer, img image.Image) error {
	return fmt.Errorf("thumbnail encoding not implemented")
}

var encodeWebp = func(writer io.Writer, img image.Image, quality float32, lossless bool) error {
	return fmt.Errorf("webp encoding not implemented")
}

var decodeAnimatedWebp = func(data io.Reader) (image.Image, error) {
	return nil, fmt.Errorf("animated webp decoding not implemented")
}

var parseHEICEXIF = func(data io.ReaderAt) ([]byte, error) {
	return nil, fmt.Errorf("HEIC EXIF parsing not implemented")
}

const maxAvatarDecodeMemory = 128 * 1024 * 1024

func (gmx *Gomuks) generateAvatarThumbnail(entry *database.Media, size int) error {
	cacheFile, err := os.Open(gmx.cacheEntryToPath(entry.Hash[:]))
	if err != nil {
		return fmt.Errorf("failed to open full file: %w", err)
	}
	img, err := decodeImageWithOrientationFix(cacheFile, maxAvatarDecodeMemory)
	if err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(gmx.TempDir, "thumbnail-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()
	thumbnailImage := imaging.Thumbnail(img, size, size, imaging.Lanczos)
	fileHasher := sha256.New()
	wrappedWriter := io.MultiWriter(fileHasher, tempFile)
	err = encodeAvatarThumbnail(wrappedWriter, thumbnailImage)
	if err != nil {
		return fmt.Errorf("failed to encode thumbnail: %w", err)
	}
	fileInfo, err := tempFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat temporary file: %w", err)
	}
	entry.ThumbnailHash = (*[32]byte)(fileHasher.Sum(nil))
	entry.ThumbnailError = ""
	entry.ThumbnailSize = fileInfo.Size()
	cachePath := gmx.cacheEntryToPath(entry.ThumbnailHash[:])
	err = os.MkdirAll(filepath.Dir(cachePath), 0700)
	if err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	tempFile.Close()
	err = os.Rename(tempFile.Name(), cachePath)
	if err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}
	return nil
}

type noErrorWriter struct {
	io.Writer
}

func (new *noErrorWriter) Write(p []byte) (n int, err error) {
	n, _ = new.Writer.Write(p)
	return
}

// note: this should stay in sync with makeAvatarFallback in web/src/api/media.ts
const fallbackAvatarTemplate = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000 1000">
  <rect x="0" y="0" width="1000" height="1000" fill="%s"/>
  <text x="500" y="750" text-anchor="middle" fill="#fff" font-weight="bold" font-size="666"
    font-family="-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif"
  >%s</text>
</svg>`

type avatarResponseWriter struct {
	http.ResponseWriter
	bgColor   string
	character string
	errored   bool
}

func isAllowedAvatarMime(mime string) bool {
	switch mime {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

func MakeFallbackAvatar(bgColor string, character string) []byte {
	return []byte(fmt.Sprintf(fallbackAvatarTemplate, bgColor, html.EscapeString(character)))
}

func (w *avatarResponseWriter) WriteHeader(statusCode int) {
	if statusCode != http.StatusOK && statusCode != http.StatusNotModified {
		data := MakeFallbackAvatar(w.bgColor, w.character)
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.Header().Del("Content-Disposition")
		w.ResponseWriter.WriteHeader(http.StatusOK)
		_, _ = w.ResponseWriter.Write(data)
		w.errored = true
		return
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *avatarResponseWriter) Write(p []byte) (n int, err error) {
	if w.errored {
		return len(p), nil
	}
	return w.ResponseWriter.Write(p)
}

func (gmx *Gomuks) DownloadMedia(w http.ResponseWriter, r *http.Request) {
	mxc := id.ContentURI{
		Homeserver: r.PathValue("server"),
		FileID:     r.PathValue("media_id"),
	}
	if !mxc.IsValid() {
		mautrix.MInvalidParam.WithMessage("Invalid mxc URI").Write(w)
		return
	}
	query := r.URL.Query()
	fallback := query.Get("fallback")
	if fallback != "" {
		fallbackParts := strings.Split(fallback, ":")
		if len(fallbackParts) == 2 {
			w = &avatarResponseWriter{
				ResponseWriter: w,
				bgColor:        fallbackParts[0],
				character:      fallbackParts[1],
			}
		}
	}

	encrypted, _ := strconv.ParseBool(query.Get("encrypted"))
	useThumbnail := query.Get("thumbnail") == "avatar"

	logVal := zerolog.Ctx(r.Context()).With().
		Stringer("mxc_uri", mxc).
		Bool("encrypted", encrypted).
		Logger()
	log := &logVal
	ctx := log.WithContext(r.Context())
	cacheEntry, err := gmx.Client.DB.Media.Get(ctx, mxc)
	if err != nil {
		log.Err(err).Msg("Failed to get cached media entry")
		mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to get cached media entry: %v", err)).Write(w)
		return
	} else if (cacheEntry == nil || cacheEntry.EncFile == nil) && encrypted {
		mautrix.MNotFound.WithMessage("Media encryption keys not found in cache").Write(w)
		return
	} else if cacheEntry != nil && cacheEntry.EncFile != nil && !encrypted {
		mautrix.MNotFound.WithMessage("Tried to download encrypted media without encrypted flag").Write(w)
		return
	}

	if gmx.downloadMediaFromCache(ctx, w, r, cacheEntry, false, useThumbnail) {
		return
	}

	tempFile, err := os.CreateTemp(gmx.TempDir, "download-*")
	if err != nil {
		log.Err(err).Msg("Failed to create temporary file")
		mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to create temp file: %v", err)).Write(w)
		return
	}
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()

	addErrorToCacheEntry := func(err error) {
		if ctx.Err() != nil {
			return
		}
		if cacheEntry == nil {
			cacheEntry = &database.Media{
				MXC: mxc,
			}
		}
		if cacheEntry.Error == nil {
			cacheEntry.Error = &database.MediaError{
				ReceivedAt: jsontime.UnixMilliNow(),
				Attempts:   1,
			}
		} else {
			cacheEntry.Error.Attempts++
			cacheEntry.Error.ReceivedAt = jsontime.UnixMilliNow()
		}
		var httpErr mautrix.HTTPError
		if errors.As(err, &httpErr) {
			if httpErr.WrappedError != nil {
				cacheEntry.Error.Matrix = ptr.Ptr(ErrBadGateway.WithMessage(httpErr.WrappedError.Error()))
				cacheEntry.Error.StatusCode = http.StatusBadGateway
			} else if httpErr.RespError != nil {
				cacheEntry.Error.Matrix = httpErr.RespError
				cacheEntry.Error.StatusCode = httpErr.Response.StatusCode
			} else {
				cacheEntry.Error.Matrix = ptr.Ptr(mautrix.MUnknown.WithMessage("Server returned non-JSON error with status %d", httpErr.Response.StatusCode))
				cacheEntry.Error.StatusCode = httpErr.Response.StatusCode
			}
		} else if errors.Is(err, attachment.ErrHashMismatch) ||
			errors.Is(err, attachment.ErrUnsupportedVersion) ||
			errors.Is(err, attachment.ErrUnsupportedAlgorithm) ||
			errors.Is(err, attachment.ErrInvalidKey) ||
			errors.Is(err, attachment.ErrInvalidInitVector) ||
			errors.Is(err, attachment.ErrInvalidHash) {
			cacheEntry.Error.Matrix = ptr.Ptr(mautrix.MUnknown.WithMessage(err.Error()))
			cacheEntry.Error.StatusCode = http.StatusInternalServerError
		} else {
			cacheEntry.Error.Matrix = ptr.Ptr(ErrBadGateway.WithMessage(err.Error()))
			cacheEntry.Error.StatusCode = http.StatusBadGateway
		}
		err = gmx.Client.DB.Media.Put(ctx, cacheEntry)
		if err != nil {
			log.Err(err).Msg("Failed to save errored cache entry")
		}
		if w != nil {
			cacheEntry.Error.Write(w)
		}
	}

	resp, err := gmx.Client.Client.Download(mautrix.WithMaxRetries(ctx, 0), mxc)
	if err != nil {
		if ctx.Err() != nil {
			w.WriteHeader(499)
			return
		}
		log.Err(err).Msg("Failed to download media")
		addErrorToCacheEntry(err)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if cacheEntry == nil {
		cacheEntry = &database.Media{
			MXC:      mxc,
			MimeType: resp.Header.Get("Content-Type"),
			Size:     resp.ContentLength,
		}
	}

	reader := resp.Body
	if cacheEntry.EncFile != nil {
		err = cacheEntry.EncFile.PrepareForDecryption()
		if err != nil {
			log.Err(err).Msg("Failed to prepare media for decryption")
			addErrorToCacheEntry(err)
			return
		}
		reader = cacheEntry.EncFile.DecryptStream(reader)
	}
	if cacheEntry.FileName == "" {
		_, params, _ := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
		cacheEntry.FileName = params["filename"]
	}
	if cacheEntry.MimeType == "" {
		cacheEntry.MimeType = resp.Header.Get("Content-Type")
	}
	cacheEntry.Size = resp.ContentLength
	fileHasher := sha256.New()
	wrappedReader := io.TeeReader(reader, fileHasher)
	if cacheEntry.Size > 0 && cacheEntry.EncFile == nil && !useThumbnail && r.Header.Get("Range") == "" {
		cacheEntryToHeaders(w, cacheEntry, useThumbnail)
		// These homeserver -> frontend streamed responses don't support ranges themselves,
		// but the browser can still request ranges, it just won't be streamed from the homeserver then.
		w.Header().Set("Accept-Ranges", "bytes")
		w.WriteHeader(http.StatusOK)
		wrappedReader = io.TeeReader(wrappedReader, &noErrorWriter{w})
		w = nil
	}
	if ctx.Err() != nil {
		return
	}
	cacheEntry.Size, err = io.Copy(tempFile, wrappedReader)
	if err != nil {
		log.Err(err).Msg("Failed to copy media to temporary file")
		addErrorToCacheEntry(err)
		return
	}
	if ctx.Err() != nil {
		return
	}
	err = reader.Close()
	if err != nil {
		log.Err(err).Msg("Failed to close media reader")
		addErrorToCacheEntry(err)
		return
	}
	// This is a hack for Beeper as some buckets (wasabi?) apparently don't respect the content-type header in uploads
	if (cacheEntry.MimeType == "application/octet-stream" || cacheEntry.MimeType == "binary/octet-stream") && fallback != "" {
		if _, err = tempFile.Seek(0, io.SeekStart); err != nil {
			log.Err(err).Msg("Failed to seek to start of temp file to find mime type")
		} else if overrideMime, err := mimetype.DetectReader(tempFile); err != nil {
			log.Err(err).Msg("Failed to detect mime type of avatar media with octet-stream type")
		} else {
			log.Debug().
				Stringer("new_mime_type", overrideMime).
				Msg("Overriding mime mime type of avatar media")
			cacheEntry.MimeType = overrideMime.String()
		}
	}
	_ = tempFile.Close()
	cacheEntry.Hash = (*[32]byte)(fileHasher.Sum(nil))
	cacheEntry.Error = nil
	err = gmx.Client.DB.Media.Put(ctx, cacheEntry)
	if err != nil {
		log.Err(err).Msg("Failed to save cache entry")
		mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to save cache entry: %v", err)).Write(w)
		return
	}
	cachePath := gmx.cacheEntryToPath(cacheEntry.Hash[:])
	err = os.MkdirAll(filepath.Dir(cachePath), 0700)
	if err != nil {
		log.Err(err).Msg("Failed to create cache directory")
		mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to create cache directory: %v", err)).Write(w)
		return
	}
	err = os.Rename(tempFile.Name(), cachePath)
	if err != nil {
		log.Err(err).Msg("Failed to rename temporary file")
		mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to rename temp file: %v", err)).Write(w)
		return
	}
	if w != nil {
		gmx.downloadMediaFromCache(ctx, w, r, cacheEntry, true, useThumbnail)
	}
}
