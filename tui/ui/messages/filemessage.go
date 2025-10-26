// gomuks - A terminal Matrix client written in Go.
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

package messages

import (
	"bytes"
	"fmt"
	"image"
	"image/color"

	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/rpc/client"
	"go.mau.fi/gomuks/tui/config"
	"go.mau.fi/gomuks/tui/debug"
	"go.mau.fi/gomuks/tui/lib/ansimage"
	"go.mau.fi/gomuks/tui/ui/messages/tstring"
)

type FileMessage struct {
	Type event.MessageType
	Body string

	URL         id.ContentURI
	IsEncrypted bool

	eventID id.EventID

	imageData []byte
	buffer    []tstring.TString

	matrix *client.GomuksClient
}

// NewFileMessage creates a new FileMessage object with the provided values and the default state.
func NewFileMessage(matrix *client.GomuksClient, evt *database.Event, content *event.MessageEventContent, displayname string) *UIMessage {
	var url id.ContentURI
	var isEncrypted bool
	if content.File != nil {
		url = content.File.URL.ParseOrIgnore()
		isEncrypted = true
	} else {
		url = content.URL.ParseOrIgnore()
	}
	return newUIMessage(evt, content, displayname, &FileMessage{
		Type:        content.MsgType,
		Body:        content.Body,
		URL:         url,
		IsEncrypted: isEncrypted,
		eventID:     evt.ID,
		matrix:      matrix,
	})
}

func (msg *FileMessage) Clone() MessageRenderer {
	data := make([]byte, len(msg.imageData))
	copy(data, msg.imageData)
	return &FileMessage{
		Body:        msg.Body,
		URL:         msg.URL,
		IsEncrypted: msg.IsEncrypted,
		imageData:   data,
		matrix:      msg.matrix,
	}
}

func (msg *FileMessage) NotificationContent() string {
	switch msg.Type {
	case event.MsgImage:
		return "Sent an image"
	case event.MsgAudio:
		return "Sent an audio file"
	case event.MsgVideo:
		return "Sent a video"
	case event.MsgFile:
		fallthrough
	default:
		return "Sent a file"
	}
}

func (msg *FileMessage) PlainText() string {
	return fmt.Sprintf("%s: %s", msg.Body, msg.matrix.GetDownloadURL(msg.URL, msg.IsEncrypted, true))
}

func (msg *FileMessage) String() string {
	return fmt.Sprintf(`&messages.FileMessage{Body="%s", URL="%s", Encrypted=%t}`, msg.Body, msg.URL, msg.IsEncrypted)
}

func (msg *FileMessage) DownloadPreview() {
	//var url id.ContentURI
	//var file *attachment.EncryptedFile
	//if !msg.Thumbnail.IsEmpty() {
	//	url = msg.Thumbnail
	//	file = msg.ThumbnailFile
	//} else if msg.Type == event.MsgImage && !msg.URL.IsEmpty() {
	//	msg.Thumbnail = msg.URL
	//	url = msg.URL
	//	file = msg.File
	//} else {
	//	return
	//}
	//debug.Print("Loading file:", url)
	//data, err := msg.matrix.Download(url, file != nil)
	//if err != nil {
	//	debug.Printf("Failed to download file %s: %v", url, err)
	//	return
	//}
	//debug.Print("File", url, "loaded.")
	//msg.imageData = data
}

func (msg *FileMessage) ThumbnailPath() string {
	return "" // FIXME
	//return msg.matrix.GetCachePath(msg.Thumbnail)
}

func (msg *FileMessage) CalculateBuffer(prefs config.UserPreferences, width int, uiMsg *UIMessage) {
	if width < 2 {
		return
	}

	if prefs.BareMessageView || prefs.DisableImages || len(msg.imageData) == 0 {
		url := msg.matrix.GetDownloadURL(msg.URL, msg.IsEncrypted, true)
		var urlTString tstring.TString
		if prefs.EnableInlineURLs() {
			urlTString = tstring.NewStyleTString(url, tcell.StyleDefault.Url(url).UrlId(msg.eventID.String()))
		} else {
			urlTString = tstring.NewTString(url)
		}
		text := tstring.NewTString(msg.Body).
			Append(": ").
			AppendTString(urlTString)
		msg.buffer = calculateBufferWithText(prefs, text, width, uiMsg)
		return
	}

	img, _, err := image.DecodeConfig(bytes.NewReader(msg.imageData))
	if err != nil {
		debug.Print("File could not be decoded:", err)
	}
	imgWidth := img.Width
	if img.Width > width {
		imgWidth = width / 3
	}

	ansFile, err := ansimage.NewScaledFromReader(bytes.NewReader(msg.imageData), 0, imgWidth, color.Black)
	if err != nil {
		msg.buffer = []tstring.TString{tstring.NewColorTString("Failed to display image", tcell.ColorRed)}
		debug.Print("Failed to display image:", err)
		return
	}

	msg.buffer = ansFile.Render()
}

func (msg *FileMessage) Height() int {
	return len(msg.buffer)
}

func (msg *FileMessage) Draw(screen mauview.Screen, _ *UIMessage) {
	for y, line := range msg.buffer {
		line.Draw(screen, 0, y)
	}
}
