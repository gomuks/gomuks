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
	"cmp"
	"context"
	"net/http"
	"strconv"

	"github.com/rs/zerolog"
	"go.mau.fi/util/exhttp"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"

	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

func (gmx *Gomuks) GetURLPreviewHTTP(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	encrypt, _ := strconv.ParseBool(r.URL.Query().Get("encrypt"))
	resp, err := gmx.GetURLPreview(r.Context(), url, encrypt)
	if err != nil {
		writeMaybeRespError(err, w)
	} else {
		exhttp.WriteJSONResponse(w, http.StatusOK, resp)
	}
}

func (gmx *Gomuks) GetURLPreview(ctx context.Context, url string, encrypt bool) (*event.BeeperLinkPreview, error) {
	log := zerolog.Ctx(ctx)
	if url == "" {
		return nil, mautrix.MInvalidParam.WithMessage("URL must be provided to preview")
	}
	linkPreview, err := gmx.Client.Client.GetURLPreview(mautrix.WithMaxRetries(ctx, 0), url)
	if err != nil {
		log.Err(err).Msg("Failed to get URL preview")
		return nil, err
	}

	preview := event.BeeperLinkPreview{
		LinkPreview: *linkPreview,
		MatchedURL:  url,
	}

	if preview.ImageURL != "" {
		var content *event.MessageEventContent

		if encrypt {
			if fileInfo, ok := gmx.temporaryMXCToEncryptedFileInfo[preview.ImageURL]; ok {
				content = &event.MessageEventContent{File: fileInfo}
			}
		} else {
			if mxc, ok := gmx.temporaryMXCToPermanent[preview.ImageURL]; ok {
				content = &event.MessageEventContent{URL: mxc}
			}
		}

		parsedImageURL, err := preview.ImageURL.Parse()
		if content == nil && (err != nil || parsedImageURL.IsEmpty()) {
			log.Warn().Err(err).Str("image_url", string(preview.ImageURL)).Msg("Failed to parse URL preview image mxc")
		} else if content == nil && !parsedImageURL.IsEmpty() {
			resp, err := gmx.Client.Client.Download(ctx, parsedImageURL)
			if err != nil {
				log.Err(err).Msg("Failed to download URL preview image")
				return nil, err
			}
			defer resp.Body.Close()

			content, err = gmx.UploadMedia(ctx, resp.Body, jsoncmd.UploadMediaParams{Encrypt: encrypt}, nil)
			if err != nil {
				log.Err(err).Msg("Failed to upload URL preview image")
				return nil, err
			}

			if encrypt {
				gmx.temporaryMXCToEncryptedFileInfo[preview.ImageURL] = content.File
			} else {
				gmx.temporaryMXCToPermanent[preview.ImageURL] = content.URL
			}
			if content.Info != nil {
				gmx.temporaryMXCToBlurhash[preview.ImageURL] = cmp.Or(content.Info.Blurhash, content.Info.AnoaBlurhash)
			}
		}

		if content != nil {
			preview.ImageBlurhash = gmx.temporaryMXCToBlurhash[preview.ImageURL]
			preview.ImageURL = content.URL
			preview.ImageEncryption = content.File
		}
	}

	return &preview, nil
}
