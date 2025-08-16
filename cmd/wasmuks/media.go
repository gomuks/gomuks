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

//go:build js

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/gomuks"
)

func uploadMedia(ctx context.Context, fileName string, encrypt bool, payload []byte) (*event.MessageEventContent, error) {
	msgType, info, defaultFileName, err := gmx.GenerateFileInfo(ctx, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to generate file info: %w", err)
	}
	info.Size = len(payload)
	if fileName == "" {
		fileName = defaultFileName
	}
	content := &event.MessageEventContent{
		MsgType:  msgType,
		Body:     fileName,
		Info:     info,
		FileName: fileName,
	}
	checksum := sha256.Sum256(payload)
	content.File, content.URL, err = gmx.UploadFile(
		ctx,
		checksum[:],
		bytes.NewReader(payload),
		encrypt,
		int64(info.Size),
		info.MimeType,
		fileName,
		nil,
	)
	return content, err
}

func realJSDownloadCallback(ctx context.Context, path, rawQuery string, callbacks js.Value) {
	resolved := false
	defer func() {
		if !resolved {
			callbacks.Call("reject")
		}
	}()
	log := zerolog.Ctx(ctx)
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		log.Error().Msg("Invalid media download path")
		return
	}
	mxc := id.ContentURI{
		Homeserver: parts[len(parts)-2],
		FileID:     parts[len(parts)-1],
	}
	query, err := url.ParseQuery(strings.TrimPrefix(rawQuery, "?"))
	if err != nil {
		log.Err(err).Msg("Failed to parse media download query")
		return
	}
	encrypted, _ := strconv.ParseBool(query.Get("encrypted"))
	useThumbnail := query.Get("thumbnail") == "avatar"
	fallback := query.Get("fallback")
	if fallback != "" {
		fallbackParts := strings.Split(fallback, ":")
		defer func() {
			if !resolved {
				log.Debug().Msg("Returning fallback avatar")
				data := gomuks.MakeFallbackAvatar(fallbackParts[0], fallbackParts[1])
				buf := js.Global().Get("Uint8Array").New(len(data))
				js.CopyBytesToJS(buf, data)
				callbacks.Call("resolve", js.ValueOf(map[string]any{
					"buffer":             buf,
					"contentType":        "image/svg+xml",
					"contentDisposition": "",
				}))
				resolved = true
			}
		}()
	}
	cacheEntry, err := gmx.Client.DB.Media.Get(ctx, mxc)
	if err != nil {
		log.Err(err).Msg("Failed to get cached media entry")
		return
	} else if (cacheEntry == nil || cacheEntry.EncFile == nil) && encrypted {
		log.Error().Msg("Tried to download encrypted media without keys")
		return
	} else if cacheEntry != nil && cacheEntry.EncFile != nil && !encrypted {
		log.Error().Msg("Tried to download encrypted media without encrypted flag")
		return
	}
	if useThumbnail {
		// TODO implement
	}
	resp, err := gmx.Client.Client.Download(mautrix.WithMaxRetries(ctx, 0), mxc)
	if err != nil {
		log.Err(err).Msg("Failed to download media")
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Err(err).Msg("Failed to read media data")
		return
	}
	if cacheEntry != nil && cacheEntry.EncFile != nil {
		err = cacheEntry.EncFile.DecryptInPlace(data)
		if err != nil {
			log.Err(err).Msg("Failed to decrypt media data")
			return
		}
	}
	contentType := resp.Header.Get("Content-Type")
	contentDisposition := resp.Header.Get("Content-Disposition")
	if cacheEntry != nil && cacheEntry.MimeType != "" {
		contentType = cacheEntry.MimeType
	}
	buf := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(buf, data)
	callbacks.Call("resolve", js.ValueOf(map[string]any{
		"buffer":             buf,
		"contentType":        contentType,
		"contentDisposition": contentDisposition,
	}))
	resolved = true
	log.Debug().
		Str("content_type", contentType).
		Str("content_disposition", contentDisposition).
		Int("length", len(data)).
		Msg("Download successful")
}

func jsDownloadCallback(_ js.Value, args []js.Value) any {
	path := args[0].String()
	query := args[1].String()
	ctx := gmx.Log.With().
		Str("action", "wasmuks download").
		Str("path", path).
		Str("query", query).
		Logger().
		WithContext(context.Background())
	go realJSDownloadCallback(ctx, path, query, args[2])
	return nil
}
