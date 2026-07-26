// gomuks - A Matrix client written in Go.
// Copyright (C) 2026 Tulir Asokan
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
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/buckket/go-blurhash"
	"github.com/disintegration/imaging"
	"github.com/gabriel-vasile/mimetype"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"go.mau.fi/util/exhttp"
	"go.mau.fi/util/ffmpeg"
	"go.mau.fi/util/ffmpeg/waveform"
	"go.mau.fi/util/ptr"
	"go.mau.fi/util/random"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/attachment"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
	"go.mau.fi/gomuks/pkg/orientation"
)

const progressMime = "application/x-mau-progress-stream+json"

func (gmx *Gomuks) UploadMediaHTTP(w http.ResponseWriter, r *http.Request) {
	log := hlog.FromRequest(r)
	progress, _ := strconv.ParseBool(r.URL.Query().Get("progress"))
	progress = progress || r.Header.Get("Accept") == progressMime
	var respEnc *json.Encoder
	var progressCallback func(progress float64)
	if progress {
		w.Header().Set("Content-Type", progressMime)
		w.WriteHeader(http.StatusOK)
		respEnc = json.NewEncoder(w)
		lastFlush := time.Now()
		var progressLock sync.Mutex
		var done bool
		defer func() {
			progressLock.Lock()
			done = true
			progressLock.Unlock()
		}()
		updateInterval := 250 * time.Millisecond
		realProgressCallback := func(progress float64) {
			defer progressLock.Unlock()
			if r.Context().Err() != nil || done {
				return
			}
			start := time.Now()
			_ = respEnc.Encode(progress)
			w.(http.Flusher).Flush()
			if time.Since(start) > 50*time.Millisecond {
				updateInterval = 500 * time.Millisecond
			}
			lastFlush = time.Now()
		}
		progressCallback = func(progress float64) {
			if time.Since(lastFlush) > updateInterval && progressLock.TryLock() {
				lastFlush = time.Now()
				go realProgressCallback(progress)
			}
		}
	}
	query := r.URL.Query()
	encrypt, _ := strconv.ParseBool(r.URL.Query().Get("encrypt"))
	voiceMessage, _ := strconv.ParseBool(r.URL.Query().Get("voice_message"))
	forceFile, _ := strconv.ParseBool(r.URL.Query().Get("force_file"))
	params := jsoncmd.UploadMediaParams{
		Filename:     query.Get("filename"),
		Encrypt:      encrypt,
		VoiceMessage: voiceMessage,
		ForceFile:    forceFile,
		EncodeTo:     query.Get("encode_to"),
	}
	resizeWidthVal := query.Get("resize_width")
	resizeHeightVal := query.Get("resize_height")
	resizePercentVal := query.Get("resize_percent")
	var err error
	if resizeWidthVal != "" && resizeHeightVal != "" {
		params.ResizeWidth, err = strconv.Atoi(resizeWidthVal)
		if err != nil {
			mautrix.MInvalidParam.WithMessage("Invalid resize width").Write(w)
			return
		}
		params.ResizeHeight, err = strconv.Atoi(resizeHeightVal)
		if err != nil {
			mautrix.MInvalidParam.WithMessage("Invalid resize height").Write(w)
			return
		}
	} else if resizePercentVal != "" {
		params.ResizePercent, err = strconv.Atoi(resizePercentVal)
		if err != nil || (params.ResizePercent < 1 || params.ResizePercent > 100) {
			mautrix.MInvalidParam.WithMessage("Invalid resize percent").Write(w)
			return
		}
	}
	if qualityVal := query.Get("quality"); qualityVal != "" {
		params.Quality, err = strconv.Atoi(qualityVal)
		if err != nil {
			mautrix.MInvalidParam.WithMessage("Invalid quality value").Write(w)
			return
		}
	}

	content, err := gmx.UploadMedia(r.Context(), r.Body, params, progressCallback)
	if err != nil {
		log.Err(err).Msg("Failed to upload media")
		if respEnc != nil {
			_ = respEnc.Encode(ptr.Ptr(ToRespError(err)))
		} else {
			writeMaybeRespError(err, w)
		}
		return
	}
	if respEnc != nil {
		_ = respEnc.Encode(content)
	} else {
		exhttp.WriteJSONResponse(w, http.StatusOK, content)
	}
}

func (gmx *Gomuks) UploadMedia(
	ctx context.Context,
	reader io.Reader,
	params jsoncmd.UploadMediaParams,
	progressCallback func(float64),
) (*event.MessageEventContent, error) {
	if reader == nil && params.Path != "" {
		file, err := os.Open(params.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()
		reader = file
	}
	if progressCallback == nil {
		progressCallback = func(_ float64) {}
	}
	log := zerolog.Ctx(ctx)
	tempFile, err := os.CreateTemp(gmx.TempDir, "upload-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file %w", err)
	}
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()
	hasher := sha256.New()
	_, err = io.Copy(tempFile, io.TeeReader(reader, hasher))
	if err != nil {
		return nil, fmt.Errorf("failed to copy upload media to temp file: %w", err)
	}
	checksum := hasher.Sum(nil)
	if newHash, err := gmx.reencodeMedia(ctx, params, tempFile); err != nil {
		return nil, fmt.Errorf("failed to reencode media: %w", err)
	} else if newHash != nil {
		checksum = newHash
	}

	cachePath := gmx.CacheEntryToPath((*[32]byte)(checksum))
	if _, err = os.Stat(cachePath); err == nil {
		log.Debug().Str("path", cachePath).Msg("Media already exists in cache, removing temp file")
	} else {
		err = os.MkdirAll(filepath.Dir(cachePath), 0700)
		if err != nil {
			return nil, fmt.Errorf("failed to create cache directory: %w", err)
		}
		err = os.Rename(tempFile.Name(), cachePath)
		if err != nil {
			return nil, fmt.Errorf("failed to rename temp file: %w", err)
		}
	}

	cacheFile, err := os.Open(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache file: %w", err)
	}

	msgType, info, defaultFileName, err := gmx.GenerateFileInfo(ctx, cacheFile)
	if err != nil {
		return nil, fmt.Errorf("failed to generate file info: %w", err)
	}
	if msgType == event.MsgVideo {
		err = gmx.generateVideoThumbnail(ctx, cacheFile.Name(), params.Encrypt, info)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to generate video thumbnail")
		}
	}
	if params.Filename == "" {
		params.Filename = defaultFileName
	}
	content := &event.MessageEventContent{
		MsgType:  msgType,
		Body:     params.Filename,
		Info:     info,
		FileName: params.Filename,
	}
	if params.ForceFile {
		content.MsgType = event.MsgFile
	} else if params.VoiceMessage {
		samples := 80
		if info.Duration != 0 {
			samples = min(max(info.Duration/125, 30), 120)
		}
		wf, err := waveform.Generate(ctx, cachePath, samples, 256)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to generate waveform")
		}
		content.MSC1767Audio = &event.MSC1767Audio{
			Duration: info.Duration,
			Waveform: wf,
		}
		content.MSC3245Voice = &event.MSC3245Voice{}
	}
	content.File, content.URL, err = gmx.uploadFileDirect(
		ctx, checksum, cacheFile, params.Encrypt, int64(info.Size), info.MimeType, params.Filename, progressCallback,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upload media: %w", err)
	}
	return content, nil
}

type progressReader struct {
	cb    func(float64)
	read  int64
	total int64
	r     io.ReadSeekCloser
}

var _ io.ReadSeekCloser = (*progressReader)(nil)

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.r.Read(p)
	pr.read += int64(n)
	pr.cb(float64(pr.read) / float64(pr.total))
	return n, err
}

func (pr *progressReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		pr.read = offset
	case io.SeekCurrent:
		pr.read += offset
	case io.SeekEnd:
		pr.read = pr.total + offset
	}
	pr.cb(float64(pr.read) / float64(pr.total))
	return pr.r.Seek(offset, whence)
}

func (pr *progressReader) Close() error {
	return pr.r.Close()
}

func (gmx *Gomuks) uploadFileDirect(
	ctx context.Context,
	checksum []byte,
	cacheReader io.Reader,
	encrypt bool,
	fileSize int64,
	mimeType,
	fileName string,
	progressCallback func(float64),
) (*event.EncryptedFileInfo, id.ContentURIString, error) {
	cm := &database.Media{
		FileName: fileName,
		MimeType: mimeType,
		Size:     fileSize,
		Hash:     (*[32]byte)(checksum),
	}
	if encrypt {
		cm.EncFile = attachment.NewEncryptedFile()
		cacheReader = cm.EncFile.EncryptStream(cacheReader)
		mimeType = "application/octet-stream"
		fileName = ""
	}
	if progressCallback != nil {
		cacheReader = &progressReader{
			cb:    progressCallback,
			total: fileSize,
			r:     cacheReader.(io.ReadSeekCloser),
		}
	}
	resp, err := gmx.Client.Client.UploadMedia(ctx, mautrix.ReqUploadMedia{
		Content:       cacheReader,
		ContentLength: fileSize,
		ContentType:   mimeType,
		FileName:      fileName,
	})
	var err2 error
	if readCloser, ok := cacheReader.(io.ReadCloser); ok {
		err2 = readCloser.Close()
	}
	if err != nil {
		return nil, "", err
	} else if err2 != nil {
		return nil, "", fmt.Errorf("failed to close cache reader: %w", err)
	}
	cm.MXC = resp.ContentURI
	err = gmx.Client.DB.Media.Put(ctx, cm)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).
			Stringer("mxc", cm.MXC).
			Hex("checksum", checksum).
			Msg("Failed to save cache entry")
	}
	if cm.EncFile != nil {
		return &event.EncryptedFileInfo{
			EncryptedFile: *cm.EncFile,
			URL:           resp.ContentURI.CUString(),
		}, "", nil
	} else {
		return nil, resp.ContentURI.CUString(), nil
	}
}

var magickPath string

func init() {
	magickPath, _ = exec.LookPath("magick")
}

func getDimensionsWithMagick(ctx context.Context, file *os.File) (w, h int) {
	if stdout, err := exec.CommandContext(ctx, magickPath, "identify", "-format", "%w %h", file.Name()+"[0]").Output(); err != nil {
		var stderr []byte
		var e *exec.ExitError
		if errors.As(err, &e) {
			stderr = e.Stderr
		}
		zerolog.Ctx(ctx).Err(err).Bytes("stderr", stderr).Msg("Failed to get image dimensions with magick")
	} else if spaceIdx := bytes.IndexByte(stdout, ' '); spaceIdx == -1 {
		zerolog.Ctx(ctx).Error().Bytes("stdout", stdout).Msg("Failed to parse magick output")
	} else if width, err := strconv.Atoi(string(stdout[:spaceIdx])); err != nil {
		zerolog.Ctx(ctx).Err(err).Bytes("stdout", stdout).Msg("Failed to parse width in magick output")
	} else if height, err := strconv.Atoi(string(stdout[spaceIdx+1:])); err != nil {
		zerolog.Ctx(ctx).Err(err).Bytes("stdout", stdout).Msg("Failed to parse height in magick output")
	} else {
		return width, height
	}
	return 0, 0
}

func (gmx *Gomuks) GenerateFileInfo(ctx context.Context, file io.ReadSeeker) (event.MessageType, *event.FileInfo, string, error) {
	mimeType, err := mimetype.DetectReader(file)
	if err != nil {
		return "", nil, "", fmt.Errorf("failed to detect mime type: %w", err)
	}
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return "", nil, "", fmt.Errorf("failed to seek to start of file: %w", err)
	}
	info := &event.FileInfo{
		MimeType: mimeType.String(),
	}
	osFile, _ := file.(*os.File)
	if osFile != nil {
		fileInfo, err := osFile.Stat()
		if err != nil {
			return "", nil, "", fmt.Errorf("failed to stat cache file: %w", err)
		}
		info.Size = int(fileInfo.Size())
	}
	var msgType event.MessageType
	var defaultFileName string
	switch strings.Split(mimeType.String(), "/")[0] {
	case "image":
		msgType = event.MsgImage
		defaultFileName = "image" + mimeType.Extension()
		img, _, err := image.Decode(file)
		if err != nil && strings.Contains(err.Error(), "webp: invalid format") {
			img, err = decodeAnimatedWebp(file)
		}
		if err != nil {
			if magickPath != "" && osFile != nil {
				zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to decode image config, trying with magick")
				info.Width, info.Height = getDimensionsWithMagick(ctx, osFile)
			} else {
				zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to decode image config and magick not installed")
			}
		} else {
			bounds := img.Bounds()
			info.Width = bounds.Dx()
			info.Height = bounds.Dy()
			blurhashSrc := img
			if info.Width > 256 || info.Height > 256 {
				if info.Width > info.Height {
					blurhashSrc = imaging.Resize(img, 128, 0, imaging.Linear)
				} else {
					blurhashSrc = imaging.Resize(img, 0, 128, imaging.Linear)
				}
			}
			hash, err := blurhash.Encode(4, 3, blurhashSrc)
			if err != nil {
				zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to generate image blurhash")
			}
			info.AnoaBlurhash = hash
			if mimeType.String() == "image/jpeg" {
				_, err = file.Seek(0, io.SeekStart)
				if err != nil {
					return "", nil, "", fmt.Errorf("failed to seek to start of file: %w", err)
				}
				info.Width, info.Height = orientation.Read(file).ApplyToDimensions(info.Width, info.Height)
			}
		}
	case "video":
		msgType = event.MsgVideo
		defaultFileName = "video" + mimeType.Extension()
	case "audio":
		msgType = event.MsgAudio
		defaultFileName = "audio" + mimeType.Extension()
	default:
		msgType = event.MsgFile
		defaultFileName = "file" + mimeType.Extension()
	}
	if osFile != nil && (msgType == event.MsgVideo || msgType == event.MsgAudio) {
		probe, err := ffmpeg.Probe(ctx, osFile.Name())
		if err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to probe video")
		} else if probe != nil && probe.Format != nil {
			info.Duration = int(probe.Format.Duration * 1000)
			for _, stream := range probe.Streams {
				if stream.Width != 0 {
					info.Width = stream.Width
					info.Height = stream.Height
					break
				}
			}
		}
	}
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return "", nil, "", fmt.Errorf("failed to seek to start of file: %w", err)
	}
	return msgType, info, defaultFileName, nil
}

const maxUploadDecodeMemory = 512 * 1024 * 1024

func (gmx *Gomuks) reencodeMedia(ctx context.Context, params jsoncmd.UploadMediaParams, tempFile *os.File) ([]byte, error) {
	defer func() {
		_ = tempFile.Close()
	}()
	if params.EncodeTo == "" {
		return nil, nil
	}
	switch params.EncodeTo {
	case "image/webp", "image/jpeg", "image/png", "image/gif":
		_, err := tempFile.Seek(0, io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("failed to seek to start of temp file: %w", err)
		}
		if params.Quality == 0 {
			params.Quality = 80
		}
		decoded, err := decodeImageWithOrientationFix(tempFile, maxUploadDecodeMemory)
		if err != nil {
			return nil, err
		}
		if params.ResizeWidth > 0 && params.ResizeHeight > 0 {
			decoded = imaging.Resize(decoded, params.ResizeWidth, params.ResizeHeight, imaging.Lanczos)
		} else if params.ResizePercent != 0 {
			params.ResizeWidth = int(float64(decoded.Bounds().Dx()) * float64(params.ResizePercent) / 100)
			params.ResizeHeight = int(float64(decoded.Bounds().Dy()) * float64(params.ResizePercent) / 100)
			decoded = imaging.Resize(decoded, params.ResizeWidth, params.ResizeHeight, imaging.Lanczos)
		}
		_, err = tempFile.Seek(0, io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("failed to seek to start of temp file: %w", err)
		}
		err = tempFile.Truncate(0)
		if err != nil {
			return nil, fmt.Errorf("failed to truncate temp file: %w", err)
		}
		switch params.EncodeTo {
		case "image/webp":
			err = encodeWebp(tempFile, decoded, float32(params.Quality), params.Quality >= 100)
		case "image/jpeg":
			err = jpeg.Encode(tempFile, decoded, &jpeg.Options{Quality: params.Quality})
		case "image/png":
			err = png.Encode(tempFile, decoded)
		case "image/gif":
			err = gif.Encode(tempFile, decoded, nil)
		default:
			panic("unreachable")
		}
		if err != nil {
			return nil, fmt.Errorf("failed to encode image: %w", err)
		}
		_, err = tempFile.Seek(0, io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("failed to seek to start of temp file: %w", err)
		}
	case "video/webm", "video/mp4", "image/webp+anim":
		_ = tempFile.Close()
		var encToExtension string
		var inputArgs, outputArgs []string
		switch params.EncodeTo {
		case "video/webm":
			encToExtension = ".webm"
			outputArgs = []string{"-c:v", "libvpx-vp9", "-c:a", "libopus", "-pix_fmt", "yuva420p"}
		case "video/mp4":
			encToExtension = ".mp4"
			outputArgs = []string{"-c:v", "libx264", "-c:a", "aac", "-pix_fmt", "yuv420p"}
		case "image/webp+anim":
			encToExtension = ".webp"
			outputArgs = []string{"-c:v", "libwebp_anim", "-pix_fmt", "yuva420p", "-loop", "0"}
		default:
			panic("unreachable")
		}
		if params.ResizeWidth > 0 && params.ResizeHeight > 0 {
			outputArgs = append(outputArgs, "-vf", fmt.Sprintf("scale=%d:%d,setsar=1:1", params.ResizeWidth, params.ResizeHeight))
		}
		outputPath, err := ffmpeg.ConvertPath(ctx, tempFile.Name(), encToExtension, inputArgs, outputArgs, true)
		if err != nil {
			return nil, fmt.Errorf("failed to convert video: %w", err)
		}
		err = os.Rename(outputPath, tempFile.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to rename converted video: %w", err)
		}
		tempFile, err = os.OpenFile(tempFile.Name(), os.O_RDONLY, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to reopen converted video: %w", err)
		}
	case "audio/ogg; codecs=opus", "audio/mp4", "audio/mpeg":
		_ = tempFile.Close()
		var encToExtension, encToCodec string
		switch params.EncodeTo {
		case "audio/ogg; codecs=opus":
			encToExtension = ".ogg"
			encToCodec = "libopus"
		case "audio/mp4":
			encToExtension = ".m4a"
			encToCodec = "aac"
		case "audio/mpeg":
			encToExtension = ".mp3"
			encToCodec = "libmp3lame"
		default:
			panic("unreachable")
		}
		// TODO allow customizing bitrate?
		outputPath, err := ffmpeg.ConvertPath(ctx, tempFile.Name(), encToExtension, nil, []string{"-c:a", encToCodec}, true)
		if err != nil {
			return nil, fmt.Errorf("failed to convert audio: %w", err)
		}
		err = os.Rename(outputPath, tempFile.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to rename converted audio: %w", err)
		}
		tempFile, err = os.OpenFile(tempFile.Name(), os.O_RDONLY, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to reopen converted audio: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported encoding target %q", params.EncodeTo)
	}
	hasher := sha256.New()
	_, err := io.Copy(hasher, tempFile)
	if err != nil {
		return nil, fmt.Errorf("failed to hash re-encoded image: %w", err)
	}
	checksum := hasher.Sum(nil)
	return checksum, nil
}

func (gmx *Gomuks) generateVideoThumbnail(ctx context.Context, filePath string, encrypt bool, saveInto *event.FileInfo) error {
	tempPath := filepath.Join(gmx.TempDir, "thumbnail-"+random.String(12)+".jpeg")
	defer os.Remove(tempPath)
	err := ffmpeg.ConvertPathWithDestination(
		ctx, filePath, tempPath, nil,
		[]string{"-frames:v", "1", "-update", "1", "-f", "image2"},
		false,
	)
	if err != nil {
		return err
	}
	tempFile, err := os.Open(tempPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer tempFile.Close()
	fileInfo, err := tempFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	hasher := sha256.New()
	_, err = io.Copy(hasher, tempFile)
	if err != nil {
		return fmt.Errorf("failed to hash file: %w", err)
	}
	thumbnailInfo := &event.FileInfo{
		MimeType: "image/jpeg",
		Size:     int(fileInfo.Size()),
	}
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek to start of file: %w", err)
	}
	img, _, err := image.Decode(tempFile)
	if err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to decode thumbnail image config")
	} else {
		bounds := img.Bounds()
		thumbnailInfo.Width = bounds.Dx()
		thumbnailInfo.Height = bounds.Dy()
		hash, err := blurhash.Encode(4, 3, img)
		if err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to generate image blurhash")
		}
		thumbnailInfo.AnoaBlurhash = hash
	}
	_ = tempFile.Close()
	checksum := hasher.Sum(nil)
	cachePath := gmx.CacheEntryToPath((*[32]byte)(checksum))
	if _, err = os.Stat(cachePath); err == nil {
		zerolog.Ctx(ctx).Debug().Str("path", cachePath).Msg("Media already exists in cache, removing temp file")
	} else {
		err = os.MkdirAll(filepath.Dir(cachePath), 0700)
		if err != nil {
			return fmt.Errorf("failed to create cache directory: %w", err)
		}
		err = os.Rename(tempPath, cachePath)
		if err != nil {
			return fmt.Errorf("failed to rename file: %w", err)
		}
	}
	tempFile, err = os.Open(cachePath)
	if err != nil {
		return fmt.Errorf("failed to open renamed file: %w", err)
	}
	saveInto.ThumbnailFile, saveInto.ThumbnailURL, err = gmx.uploadFileDirect(
		ctx, checksum, tempFile, encrypt, fileInfo.Size(), "image/jpeg", "thumbnail.jpeg", func(_ float64) {},
	)
	if err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}
	saveInto.ThumbnailInfo = thumbnailInfo
	return nil
}

func ToRespError(err error) mautrix.RespError {
	var httpErr mautrix.HTTPError
	if errors.As(err, &httpErr) {
		if httpErr.WrappedError != nil {
			return ErrBadGateway.WithMessage(httpErr.WrappedError.Error())
		} else if httpErr.RespError != nil {
			return *httpErr.RespError
		} else {
			return mautrix.MUnknown.WithMessage("Server returned non-JSON error with status %d", httpErr.Response.StatusCode)
		}
	} else {
		return mautrix.MUnknown.WithMessage(err.Error())
	}
}

func writeMaybeRespError(err error, w http.ResponseWriter) {
	ToRespError(err).Write(w)
}
