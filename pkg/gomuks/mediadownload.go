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
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
	"go.mau.fi/gomuks/pkg/orientation"
)

func (gmx *Gomuks) DownloadMediaHTTP(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	encrypted, _ := strconv.ParseBool(query.Get("encrypted"))
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
	params := jsoncmd.DownloadMediaParams{
		MXC: id.ContentURI{
			Homeserver: r.PathValue("server"),
			FileID:     r.PathValue("media_id"),
		},
		Encrypted:       encrypted,
		IsAvatar:        fallback != "",
		ThumbnailAvatar: query.Get("thumbnail") == "avatar",
	}
	if query.Get("crypto_version") == "v2" {
		params.Keys = &attachment.EncryptedFile{
			Key: attachment.JSONWebKey{
				Key:         query.Get("crypto_key"),
				Algorithm:   "A256CTR",
				Extractable: true,
				KeyType:     "oct",
				KeyOps:      []string{"encrypt", "decrypt"},
			},
			InitVector: query.Get("crypto_iv"),
			Hashes: attachment.EncryptedFileHashes{
				SHA256: query.Get("crypto_hash"),
			},
			Version: "v2",
		}
	}

	entry, err := gmx.GetMediaCacheEntry(r.Context(), params)
	if err != nil {
		writeDownloadError(w, r, err)
		return
	}
	if gmx.downloadMediaFromCache(w, r, entry, params, false) {
		return
	}
	getStreamWriter := func(cacheEntry *database.Media) io.Writer {
		if r.Header.Get("Range") != "" || params.ThumbnailAvatar {
			return nil
		}
		cacheEntry.ToHeaders(w.Header(), false)
		// These homeserver -> frontend streamed responses don't support ranges themselves,
		// but the browser can still request ranges, it just won't be streamed from the homeserver then.
		w.Header().Set("Accept-Ranges", "bytes")
		w.WriteHeader(http.StatusOK)
		sw := w
		w = nil
		return sw
	}
	entry, err = gmx.DownloadMedia(r.Context(), entry, getStreamWriter, params)
	if err != nil {
		writeDownloadError(w, r, err)
	} else {
		gmx.downloadMediaFromCache(w, r, entry, params, true)
	}
}

func (gmx *Gomuks) downloadMediaFromCache(w http.ResponseWriter, r *http.Request, entry *database.Media, params jsoncmd.DownloadMediaParams, force bool) bool {
	if w == nil {
		return true
	}
	cacheFile, err := gmx.OpenCacheFile(r.Context(), entry, params, force, r.Header.Get("If-None-Match"))
	if errors.Is(err, ErrNotModified) {
		w.WriteHeader(http.StatusNotModified)
	} else if errors.Is(err, ErrUnsupportedMediaType) {
		w.WriteHeader(http.StatusUnsupportedMediaType)
	} else if err != nil {
		writeDownloadError(w, r, err)
	} else if cacheFile == nil {
		if force {
			panic(fmt.Errorf("OpenCacheFile returned nil with force flag"))
		}
		return false
	} else {
		defer func() {
			_ = cacheFile.Close()
		}()
		entry.ToHeaders(w.Header(), params.ThumbnailAvatar)
		http.ServeContent(w, r, "", time.Time{}, cacheFile)
	}
	return true
}

func writeDownloadError(w http.ResponseWriter, r *http.Request, err error) {
	if w == nil {
		return
	} else if r.Context().Err() != nil {
		w.WriteHeader(499)
	}
	var respErr mautrix.RespError
	if errors.As(err, &respErr) {
		respErr.Write(w)
	} else {
		mautrix.MUnknown.WithMessage(err.Error()).Write(w)
	}
}

func (gmx *Gomuks) GetMediaCacheEntry(ctx context.Context, params jsoncmd.DownloadMediaParams) (*database.Media, error) {
	if !params.MXC.IsValid() {
		return nil, mautrix.MInvalidParam.WithMessage("Invalid mxc URI")
	}

	encrypted := params.Encrypted

	logVal := zerolog.Ctx(ctx).With().
		Str("action", "download media").
		Stringer("mxc_uri", params.MXC).
		Bool("encrypted", encrypted).
		Logger()
	log := &logVal
	ctx = log.WithContext(ctx)
	cacheEntry, err := gmx.Client.DB.Media.Get(ctx, params.MXC)
	if err != nil {
		log.Err(err).Msg("Failed to get cached media entry")
		return nil, mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to get cached media entry: %v", err))
	} else if (cacheEntry == nil || cacheEntry.EncFile == nil) && encrypted {
		if params.Keys != nil {
			return &database.Media{
				MXC:     params.MXC,
				EncFile: params.Keys,
			}, nil
		}
		return nil, mautrix.MNotFound.WithMessage("Media encryption keys not found in cache")
	} else if cacheEntry != nil && cacheEntry.EncFile != nil && !encrypted {
		return nil, mautrix.MNotFound.WithMessage("Tried to download encrypted media without encrypted flag")
	}
	return cacheEntry, nil
}

func (gmx *Gomuks) DownloadMedia(
	ctx context.Context,
	cacheEntry *database.Media,
	getStreamWriter func(*database.Media) io.Writer,
	params jsoncmd.DownloadMediaParams,
) (*database.Media, error) {
	if cacheEntry == nil {
		cacheEntry = &database.Media{
			MXC: params.MXC,
		}
	}

	log := zerolog.Ctx(ctx)
	tempFile, err := os.CreateTemp(gmx.TempDir, "download-*")
	if err != nil {
		log.Err(err).Msg("Failed to create temporary file")
		return nil, mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to create temp file: %v", err))
	}
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()

	addErrorToCacheEntry := func(err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
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
		return cacheEntry.Error.AsRespError()
	}

	resp, err := gmx.Client.Client.Download(mautrix.WithMaxRetries(ctx, 0), params.MXC)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		log.Err(err).Msg("Failed to download media")
		return nil, addErrorToCacheEntry(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if cacheEntry.FileName == "" {
		_, respDisposition, _ := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
		cacheEntry.FileName = respDisposition["filename"]
	}
	if cacheEntry.MimeType == "" {
		cacheEntry.MimeType = resp.Header.Get("Content-Type")
	}
	cacheEntry.Size = resp.ContentLength

	reader := resp.Body
	if cacheEntry.EncFile != nil {
		err = cacheEntry.EncFile.PrepareForDecryption()
		if err != nil {
			log.Err(err).Msg("Failed to prepare media for decryption")
			return nil, addErrorToCacheEntry(err)
		}
		reader = cacheEntry.EncFile.DecryptStream(reader)
	}
	fileHasher := sha256.New()
	wrappedReader := io.TeeReader(reader, fileHasher)
	var sw io.Writer
	if cacheEntry.EncFile == nil && !params.ThumbnailAvatar && getStreamWriter != nil {
		sw = getStreamWriter(cacheEntry)
		if sw != nil {
			wrappedReader = io.TeeReader(wrappedReader, &noErrorWriter{sw})
		}
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	cacheEntry.Size, err = io.Copy(tempFile, wrappedReader)
	if err != nil {
		log.Err(err).Msg("Failed to copy media to temporary file")
		return nil, addErrorToCacheEntry(err)
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if sw != nil {
		swCloser, ok := sw.(io.Closer)
		if ok {
			_ = swCloser.Close()
		}
	}
	err = reader.Close()
	if err != nil {
		log.Err(err).Msg("Failed to close media reader")
		return nil, addErrorToCacheEntry(err)
	}
	// This is a hack for Beeper as some buckets (wasabi?) apparently don't respect the content-type header in uploads
	if (cacheEntry.MimeType == "application/octet-stream" || cacheEntry.MimeType == "binary/octet-stream") && params.IsAvatar {
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
		return nil, mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to save cache entry: %v", err))
	}
	cachePath := gmx.CacheEntryToPath(cacheEntry.Hash)
	err = os.MkdirAll(filepath.Dir(cachePath), 0700)
	if err != nil {
		log.Err(err).Msg("Failed to create cache directory")
		return nil, mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to create cache directory: %v", err))
	}
	err = os.Rename(tempFile.Name(), cachePath)
	if err != nil {
		log.Err(err).Msg("Failed to rename temporary file")
		return nil, mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to rename temp file: %v", err))
	}
	return cacheEntry, nil
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

var ErrBadGateway = mautrix.RespError{
	ErrCode:    "FI.MAU.GOMUKS.BAD_GATEWAY",
	StatusCode: http.StatusBadGateway,
}

var ErrNotModified = errors.New("not modified")
var ErrUnsupportedMediaType = errors.New("unsupported media type for avatar")
var ErrThumbnailGenerationError = mautrix.RespError{
	ErrCode:    "FI.MAU.GOMUKS.THUMBNAIL_ERROR",
	Err:        "Failed to generate thumbnail",
	StatusCode: http.StatusInternalServerError,
}

func (gmx *Gomuks) OpenCacheFile(
	ctx context.Context,
	entry *database.Media,
	params jsoncmd.DownloadMediaParams,
	force bool,
	ifNoneMatch string,
) (*os.File, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if !entry.UseCache() {
		if force {
			return nil, mautrix.MNotFound.WithMessage("Media not found in cache")
		}
		return nil, nil
	}
	etag := entry.ETag(params.ThumbnailAvatar)
	if entry.Error != nil {
		return nil, entry.Error.AsRespError().WithExtraHeader("Mau-Cached-Error", "true")
	} else if etag != "" && ifNoneMatch == etag {
		return nil, ErrNotModified
	} else if entry.MimeType != "" && params.IsAvatar && !isAllowedAvatarMime(entry.MimeType) {
		return nil, fmt.Errorf("%w %s", ErrUnsupportedMediaType, entry.MimeType)
	}
	log := zerolog.Ctx(ctx)
	hash := entry.Hash
	if params.ThumbnailAvatar {
		if entry.ThumbnailError != "" {
			log.Debug().Str(zerolog.ErrorFieldName, entry.ThumbnailError).Msg("Returning cached thumbnail error")
			return nil, ErrThumbnailGenerationError.
				WithMessage(entry.ThumbnailError).
				WithExtraHeader("Mau-Cached-Error", "true")
		}
		if entry.ThumbnailHash == nil {
			err := gmx.generateAvatarThumbnail(entry, gmx.Config.Media.ThumbnailSize)
			if errors.Is(err, os.ErrNotExist) && !force {
				return nil, nil
			} else if err != nil {
				log.Err(err).Msg("Failed to generate avatar thumbnail")
				gmx.saveMediaCacheEntryWithThumbnail(ctx, entry, err)
				return nil, ErrThumbnailGenerationError.WithMessage(err.Error())
			}
			gmx.saveMediaCacheEntryWithThumbnail(ctx, entry, nil)
		}
		hash = entry.ThumbnailHash
	}
	cacheFile, err := os.Open(gmx.CacheEntryToPath(hash))
	if params.ThumbnailAvatar && errors.Is(err, os.ErrNotExist) {
		err = gmx.generateAvatarThumbnail(entry, gmx.Config.Media.ThumbnailSize)
		if errors.Is(err, os.ErrNotExist) && !force {
			return nil, nil
		} else if err != nil {
			log.Err(err).Msg("Failed to generate avatar thumbnail")
			gmx.saveMediaCacheEntryWithThumbnail(ctx, entry, err)
			return nil, ErrThumbnailGenerationError.WithMessage(err.Error())
		}
		gmx.saveMediaCacheEntryWithThumbnail(ctx, entry, nil)
		cacheFile, err = os.Open(gmx.CacheEntryToPath(hash))
	}
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !force {
			return nil, nil
		}
		log.Err(err).Msg("Failed to open cache file")
		return nil, mautrix.MUnknown.WithMessage(fmt.Sprintf("Failed to open cache file: %v", err))
	}
	return cacheFile, nil
}

func (gmx *Gomuks) CacheEntryToPath(hash *[32]byte) string {
	if hash == nil {
		return ""
	}
	hashPath := hex.EncodeToString(hash[:])
	return filepath.Join(gmx.CacheDir, "media", hashPath[0:2], hashPath[2:4], hashPath[4:])
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
	cacheFile, err := os.Open(gmx.CacheEntryToPath(entry.Hash))
	if err != nil {
		return fmt.Errorf("failed to open full file: %w", err)
	}
	img, err := decodeImageWithOrientationFix(cacheFile, maxAvatarDecodeMemory)
	_ = cacheFile.Close()
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
	cachePath := gmx.CacheEntryToPath(entry.ThumbnailHash)
	err = os.MkdirAll(filepath.Dir(cachePath), 0700)
	if err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	_ = tempFile.Close()
	err = os.Rename(tempFile.Name(), cachePath)
	if err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}
	return nil
}
