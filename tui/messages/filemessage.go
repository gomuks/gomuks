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
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"runtime"
	"sync"

	"github.com/disintegration/imaging"
	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"
	_ "go.mau.fi/webp"
	_ "golang.org/x/image/bmp"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/rpc/client"
	"go.mau.fi/gomuks/pkg/rpc/store"
	"go.mau.fi/gomuks/tui/config"
	"go.mau.fi/gomuks/tui/debug"
	"go.mau.fi/gomuks/tui/lib/ansimage"
	"go.mau.fi/gomuks/tui/lib/termimg"
	"go.mau.fi/gomuks/tui/messages/tstring"
)

var (
	// ActiveRoomChecker checks if a given room is currently the active room in the TUI.
	ActiveRoomChecker func(roomID id.RoomID) bool
	// RequestRedraw requests a UI redraw when media finishes downloading for an active room.
	RequestRedraw func(roomID id.RoomID)

	oscOutputWriter io.Writer
)

// SetOSCOutputWriter sets an optional writer destination for emitted OSC 1337 escape sequences (useful for tests).
func SetOSCOutputWriter(w io.Writer) {
	oscOutputWriter = w
}

type FileMessage struct {
	mu sync.RWMutex

	Type event.MessageType
	Body string

	URL         id.ContentURI
	IsEncrypted bool

	ThumbnailURL       id.ContentURI
	ThumbnailEncrypted bool

	eventID id.EventID

	imageData   []byte
	imageErr    error
	buffer      []tstring.TString
	cachedWidth int
	cachedProto termimg.Protocol
	cachedOSC   []byte
	cachedKitty []byte
	imageID     uint32
	renderCache map[int][]tstring.TString

	mask        *termimg.ImageMask
	lastDrawnX  int
	lastDrawnY  int
	lastDrawnW  int
	lastDrawnH  int
	lastEmitted bool

	matrix *client.GomuksClient
	room   *store.RoomStore
	uiMsg  *UIMessage

	downloading        bool
	downloadDone       chan struct{}
	pendingCallbacks   []func()
	callbackRunnerGID  int64

	downloadFn         func(uri id.ContentURI, encrypted bool) ([]byte, error)
	ActiveRoomChecker  func() bool
	Redraw             func()
	onDownloadComplete func()
}

// NewFileMessage creates a new FileMessage object with the provided values and the default state.
func NewFileMessage(room *store.RoomStore, matrix *client.GomuksClient, evt *database.Event, content *event.MessageEventContent) *UIMessage {
	var url id.ContentURI
	var isEncrypted bool
	if content.File != nil {
		url = content.File.URL.ParseOrIgnore()
		isEncrypted = true
	} else {
		url = content.URL.ParseOrIgnore()
	}

	var thumbURL id.ContentURI
	var thumbEncrypted bool
	if content != nil {
		if info := content.GetInfo(); info != nil {
			if info.ThumbnailFile != nil {
				thumbURL = info.ThumbnailFile.URL.ParseOrIgnore()
				thumbEncrypted = true
			} else if info.ThumbnailURL != "" {
				thumbURL = info.ThumbnailURL.ParseOrIgnore()
				thumbEncrypted = false
			}
		}
	}

	var evtID id.EventID
	if evt != nil {
		evtID = evt.ID
	}

	fileMsg := &FileMessage{
		Type:               content.MsgType,
		Body:               content.Body,
		URL:                url,
		IsEncrypted:        isEncrypted,
		ThumbnailURL:       thumbURL,
		ThumbnailEncrypted: thumbEncrypted,
		eventID:            evtID,
		matrix:             matrix,
		room:               room,
	}

	uiMsg := newUIMessage(room, evt, content, "", fileMsg)
	fileMsg.uiMsg = uiMsg
	return uiMsg
}

// SetUIMessage sets the back-reference to the parent UIMessage.
func (msg *FileMessage) SetUIMessage(uiMsg *UIMessage) {
	msg.mu.Lock()
	defer msg.mu.Unlock()
	msg.uiMsg = uiMsg
}

func (msg *FileMessage) Clone() MessageRenderer {
	msg.mu.RLock()
	defer msg.mu.RUnlock()

	var data []byte
	if msg.imageData != nil {
		data = make([]byte, len(msg.imageData))
		copy(data, msg.imageData)
	}

	var bufCopy []tstring.TString
	if msg.buffer != nil {
		bufCopy = make([]tstring.TString, len(msg.buffer))
		for i, line := range msg.buffer {
			bufCopy[i] = line.Clone()
		}
	}

	var oscCopy []byte
	if msg.cachedOSC != nil {
		oscCopy = make([]byte, len(msg.cachedOSC))
		copy(oscCopy, msg.cachedOSC)
	}

	var cacheCopy map[int][]tstring.TString
	if msg.renderCache != nil {
		cacheCopy = make(map[int][]tstring.TString, len(msg.renderCache))
		for k, v := range msg.renderCache {
			lines := make([]tstring.TString, len(v))
			for i, l := range v {
				lines[i] = l.Clone()
			}
			cacheCopy[k] = lines
		}
	}

	return &FileMessage{
		Type:               msg.Type,
		Body:               msg.Body,
		URL:                msg.URL,
		IsEncrypted:        msg.IsEncrypted,
		ThumbnailURL:       msg.ThumbnailURL,
		ThumbnailEncrypted: msg.ThumbnailEncrypted,
		eventID:            msg.eventID,
		imageData:          data,
		imageErr:           msg.imageErr,
		buffer:             bufCopy,
		cachedWidth:        msg.cachedWidth,
		cachedProto:        msg.cachedProto,
		cachedOSC:          oscCopy,
		renderCache:        cacheCopy,
		matrix:             msg.matrix,
		room:               msg.room,
		uiMsg:              nil, // Linked by UIMessage.Clone() to avoid stale aliasing
		downloadFn:         msg.downloadFn,
		ActiveRoomChecker:  msg.ActiveRoomChecker,
		Redraw:             msg.Redraw,
		onDownloadComplete: msg.onDownloadComplete,
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
	msg.mu.RLock()
	defer msg.mu.RUnlock()
	url := msg.URL.String()
	if msg.matrix != nil {
		url = msg.matrix.GetDownloadURL(msg.URL, msg.IsEncrypted, true)
	}
	return fmt.Sprintf("%s: %s", msg.Body, url)
}

func (msg *FileMessage) String() string {
	msg.mu.RLock()
	defer msg.mu.RUnlock()
	return fmt.Sprintf(`&messages.FileMessage{Body="%s", URL="%s", Encrypted=%t, HasData=%t}`, msg.Body, msg.URL, msg.IsEncrypted, len(msg.imageData) > 0)
}

func (msg *FileMessage) download(uri id.ContentURI, encrypted bool) ([]byte, error) {
	msg.mu.RLock()
	dlFn := msg.downloadFn
	matrixClient := msg.matrix
	msg.mu.RUnlock()

	if dlFn != nil {
		return dlFn(uri, encrypted)
	}
	if matrixClient == nil {
		return nil, errors.New("gomuks client not initialized")
	}
	return matrixClient.Download(uri, encrypted)
}

func (msg *FileMessage) isRoomActive() bool {
	msg.mu.RLock()
	checker := msg.ActiveRoomChecker
	room := msg.room
	msg.mu.RUnlock()

	if checker != nil {
		return checker()
	}
	if ActiveRoomChecker != nil && room != nil {
		return ActiveRoomChecker(room.ID)
	}
	if room == nil {
		return true
	}
	return false
}

func (msg *FileMessage) requestRedraw() {
	msg.mu.RLock()
	redraw := msg.Redraw
	room := msg.room
	msg.mu.RUnlock()

	if redraw != nil {
		redraw()
	} else if RequestRedraw != nil && room != nil {
		RequestRedraw(room.ID)
	}
}

func (msg *FileMessage) notifyCallbacks(callbacks []func()) {
	for _, fn := range callbacks {
		if fn != nil {
			fn()
		}
	}
}

func parseGoroutineIDs() (curID int64, parentID int64) {
	var buf [4096]byte
	n := runtime.Stack(buf[:], false)
	stack := buf[:n]

	const gPrefix = "goroutine "
	if bytes.HasPrefix(stack, []byte(gPrefix)) {
		i := len(gPrefix)
		for i < len(stack) && stack[i] >= '0' && stack[i] <= '9' {
			curID = curID*10 + int64(stack[i]-'0')
			i++
		}
	}

	const pPrefix = " in goroutine "
	if idx := bytes.LastIndex(stack, []byte(pPrefix)); idx != -1 {
		i := idx + len(pPrefix)
		for i < len(stack) && stack[i] >= '0' && stack[i] <= '9' {
			parentID = parentID*10 + int64(stack[i]-'0')
			i++
		}
	}
	return curID, parentID
}

func (msg *FileMessage) DownloadPreview(onDone ...func()) {
	// Only proceed if preview is needed (images, or media with thumbnails)
	if msg.Type != event.MsgImage && msg.ThumbnailURL.IsEmpty() {
		msg.notifyCallbacks(onDone)
		return
	}

	msg.mu.Lock()
	// 1. If already downloaded, fire callbacks immediately
	if len(msg.imageData) > 0 || msg.imageErr != nil {
		msg.mu.Unlock()
		msg.notifyCallbacks(onDone)
		return
	}

	// 2. If download is already in-flight, queue callbacks to be executed when it finishes
	if msg.downloading || msg.downloadDone != nil {
		for _, fn := range onDone {
			if fn != nil {
				msg.pendingCallbacks = append(msg.pendingCallbacks, fn)
			}
		}
		msg.mu.Unlock()
		return
	}

	// 3. Start download
	msg.downloading = true
	doneCh := make(chan struct{})
	msg.downloadDone = doneCh
	for _, fn := range onDone {
		if fn != nil {
			msg.pendingCallbacks = append(msg.pendingCallbacks, fn)
		}
	}
	msg.mu.Unlock()

	go func(myDone chan struct{}) {
		var closeOnce sync.Once
		closeDone := func() {
			closeOnce.Do(func() {
				close(myDone)
			})
		}
		curGID, _ := parseGoroutineIDs()
		// Safety defer: guarantee channel closure and callback execution even on panic
		defer func() {
			msg.mu.Lock()
			if msg.downloading || msg.callbackRunnerGID != 0 || msg.downloadDone == myDone {
				msg.downloading = false
				msg.callbackRunnerGID = 0
				if msg.downloadDone == myDone {
					msg.downloadDone = nil
				}
				cbs := msg.pendingCallbacks
				msg.pendingCallbacks = nil
				onDoneComplete := msg.onDownloadComplete
				msg.mu.Unlock()
				if onDoneComplete != nil {
					cbs = append(cbs, onDoneComplete)
				}
				msg.notifyCallbacks(cbs)
				closeDone()
			} else {
				msg.mu.Unlock()
			}
		}()

		var data []byte
		var err error

		// 1. Try thumbnail first if available
		if !msg.ThumbnailURL.IsEmpty() {
			data, err = msg.download(msg.ThumbnailURL, msg.ThumbnailEncrypted)
			if err == nil && len(data) > 0 {
				_, valErr := termimg.ValidateImageSafety(bytes.NewReader(data))
				if valErr != nil {
					debug.Print("Thumbnail failed safety validation:", valErr)
					data = nil
					err = valErr
				}
			}
		}

		// 2. If thumbnail unavailable or failed, fallback to full media ONLY for event.MsgImage
		if len(data) == 0 && msg.Type == event.MsgImage && !msg.URL.IsEmpty() {
			data, err = msg.download(msg.URL, msg.IsEncrypted)
			if err == nil && len(data) > 0 {
				_, valErr := termimg.ValidateImageSafety(bytes.NewReader(data))
				if valErr != nil {
					debug.Print("Full media failed safety validation:", valErr)
					data = nil
					err = valErr
				}
			}
		}

		// 3. Finalize state under lock
		msg.mu.Lock()
		if err != nil && len(data) == 0 {
			debug.Print("Failed to download media preview:", err)
			msg.imageErr = err
			msg.imageData = nil
		} else if len(data) > 0 {
			msg.imageData = data
			msg.imageErr = nil
		}
		msg.cachedWidth = 0
		msg.cachedProto = ""
		msg.cachedOSC = nil
		msg.cachedKitty = nil
		msg.imageID = 0
		msg.renderCache = nil
		msg.buffer = nil
		msg.lastEmitted = false
		if msg.uiMsg != nil {
			msg.uiMsg.InvalidateBuffer()
		}

		// Mark downloading as complete so IsDownloading() returns false
		msg.downloading = false
		msg.mu.Unlock()

		// 4. Trigger redraw if active BEFORE notifying callbacks and BEFORE closing downloadDone
		if msg.isRoomActive() {
			msg.requestRedraw()
		}

		// 5. Notify all callbacks (and any newly queued callbacks in a loop so none are orphaned)
		onDoneInvoked := false
		for {
			msg.mu.Lock()
			cbs := msg.pendingCallbacks
			msg.pendingCallbacks = nil
			if !onDoneInvoked && msg.onDownloadComplete != nil {
				cbs = append(cbs, msg.onDownloadComplete)
				onDoneInvoked = true
			}
			if len(cbs) == 0 {
				msg.callbackRunnerGID = 0
				if msg.downloadDone == myDone {
					msg.downloadDone = nil
				}
				msg.mu.Unlock()
				closeDone()
				break
			}
			msg.callbackRunnerGID = curGID
			msg.mu.Unlock()

			msg.notifyCallbacks(cbs)
		}
	}(doneCh)
}

// WaitDownload blocks until any active background download finishes.
func (msg *FileMessage) WaitDownload() {
	msg.mu.RLock()
	cbGID := msg.callbackRunnerGID
	done := msg.downloadDone
	msg.mu.RUnlock()

	if cbGID != 0 {
		curID, parentID := parseGoroutineIDs()
		if curID == cbGID || parentID == cbGID {
			return
		}
	}

	if done != nil {
		<-done
	}
}

// ImageData returns a copy of the downloaded image data.
func (msg *FileMessage) ImageData() []byte {
	msg.mu.RLock()
	defer msg.mu.RUnlock()
	if msg.imageData == nil {
		return nil
	}
	data := make([]byte, len(msg.imageData))
	copy(data, msg.imageData)
	return data
}

// SetImageData sets the image data under write lock.
func (msg *FileMessage) SetImageData(data []byte) {
	var valErr error
	if len(data) > 0 {
		_, valErr = termimg.ValidateImageSafety(bytes.NewReader(data))
	}

	msg.mu.Lock()
	defer msg.mu.Unlock()
	if valErr != nil {
		msg.imageData = nil
		msg.imageErr = valErr
	} else {
		msg.imageData = data
		msg.imageErr = nil
	}
	msg.cachedWidth = 0
	msg.cachedProto = ""
	msg.cachedOSC = nil
	msg.cachedKitty = nil
	msg.imageID = 0
	msg.renderCache = nil
	msg.buffer = nil
	msg.lastEmitted = false
	if msg.uiMsg != nil {
		msg.uiMsg.InvalidateBuffer()
	}
}

// ImageError returns any image decode/validation error recorded.
func (msg *FileMessage) ImageError() error {
	msg.mu.RLock()
	defer msg.mu.RUnlock()
	return msg.imageErr
}

// IsDownloading returns true if a background download is currently in progress.
func (msg *FileMessage) IsDownloading() bool {
	msg.mu.RLock()
	defer msg.mu.RUnlock()
	return msg.downloading
}

// SetDownloadFunc sets a custom downloader function, useful for tests.
func (msg *FileMessage) SetDownloadFunc(fn func(uri id.ContentURI, encrypted bool) ([]byte, error)) {
	msg.mu.Lock()
	defer msg.mu.Unlock()
	msg.downloadFn = fn
}

// SetOnDownloadComplete sets a completion callback.
func (msg *FileMessage) SetOnDownloadComplete(fn func()) {
	msg.mu.Lock()
	defer msg.mu.Unlock()
	msg.onDownloadComplete = fn
}

func (msg *FileMessage) ThumbnailPath() string {
	return "" // FIXME
	//return msg.matrix.GetCachePath(msg.Thumbnail)
}

func (msg *FileMessage) CalculateBuffer(prefs config.UserPreferences, width int, uiMsg *UIMessage) {
	if width < 2 {
		return
	}

	msg.mu.RLock()
	dataLen := len(msg.imageData)
	imgErr := msg.imageErr
	var dataCopy []byte
	if dataLen > 0 {
		dataCopy = make([]byte, dataLen)
		copy(dataCopy, msg.imageData)
	}
	msgURL := msg.URL
	msgIsEncrypted := msg.IsEncrypted
	msgMatrix := msg.matrix
	msgBody := msg.Body
	msgEventID := msg.eventID
	cachedW := msg.cachedWidth
	cachedProto := msg.cachedProto
	bufLen := len(msg.buffer)
	msg.mu.RUnlock()

	effectiveProto := termimg.ResolveProtocol(prefs.ImagePreviewProtocol)

	// Quick return if width and protocol match cached buffer
	if cachedW == width && cachedProto == effectiveProto && bufLen > 0 {
		return
	}

	// Fallback to text link if disabled, bare mode, or no image data
	if prefs.BareMessageView || prefs.DisableImages || dataLen == 0 || effectiveProto == termimg.ProtocolDisabled {
		url := msgURL.String()
		if msgMatrix != nil {
			url = msgMatrix.GetDownloadURL(msgURL, msgIsEncrypted, true)
		}
		var urlTString tstring.TString
		if prefs.EnableInlineURLs() {
			urlTString = tstring.NewStyleTString("Download media", tcell.StyleDefault.Url(url).UrlId(msgEventID.String()))
		} else {
			urlTString = tstring.NewTString(url)
		}
		text := tstring.NewTString(msgBody).
			Append(": ").
			AppendTString(urlTString)
		if imgErr != nil {
			text = text.AppendColor(fmt.Sprintf(" (%v)", imgErr), tcell.ColorRed)
		}
		buf := calculateBufferWithText(prefs, text, width, uiMsg)

		msg.mu.Lock()
		msg.buffer = buf
		msg.cachedWidth = width
		msg.cachedProto = effectiveProto
		msg.cachedOSC = nil
		msg.cachedKitty = nil
		msg.imageID = 0
		msg.lastEmitted = false
		msg.mu.Unlock()
		return
	}

	// Pre-validate safety (rejecting bombs > 16MP) and extract dimensions
	cfg, err := termimg.ValidateImageSafety(bytes.NewReader(dataCopy))
	if err != nil {
		debug.Print("Image failed safety validation:", err)
		msg.mu.Lock()
		msg.imageErr = err
		msg.cachedProto = effectiveProto
		msg.cachedOSC = nil
		msg.cachedKitty = nil
		msg.imageID = 0
		msg.lastEmitted = false
		msg.mu.Unlock()

		url := msgURL.String()
		if msgMatrix != nil {
			url = msgMatrix.GetDownloadURL(msgURL, msgIsEncrypted, true)
		}
		var urlTString tstring.TString
		if prefs.EnableInlineURLs() {
			urlTString = tstring.NewStyleTString("Download media", tcell.StyleDefault.Url(url).UrlId(msgEventID.String()))
		} else {
			urlTString = tstring.NewTString(url)
		}
		text := tstring.NewTString(msgBody).
			Append(": ").
			AppendTString(urlTString).
			AppendColor(fmt.Sprintf(" (%v)", err), tcell.ColorRed)
		buf := calculateBufferWithText(prefs, text, width, uiMsg)

		msg.mu.Lock()
		msg.buffer = buf
		msg.cachedWidth = width
		msg.mu.Unlock()
		return
	}

	maxCols := prefs.ImagePreviewMaxWidth
	if maxCols <= 0 {
		maxCols = termimg.DefaultMaxWidth
	}
	maxRows := prefs.ImagePreviewMaxHeight
	if maxRows <= 0 {
		maxRows = termimg.DefaultMaxHeight
	}

	bbox := termimg.CalculateClampedDimensions(cfg.Width, cfg.Height, width, maxCols, maxRows)
	if bbox.Cols < 1 {
		bbox.Cols = 1
	}
	if bbox.Rows < 1 {
		bbox.Rows = termimg.MinHeight
	}

	switch effectiveProto {
	case termimg.ProtocolKitty:
		// Tier 1: WezTerm & Kitty via Kitty Graphics Protocol with Unicode Placeholders
		img, dErr := imaging.Decode(bytes.NewReader(dataCopy), imaging.AutoOrientation(true))
		if dErr != nil {
			var stdErr error
			img, _, stdErr = image.Decode(bytes.NewReader(dataCopy))
			if stdErr != nil {
				debug.Print("Failed to decode image for Kitty protocol:", dErr)
				msg.mu.Lock()
				msg.imageErr = dErr
				msg.buffer = []tstring.TString{tstring.NewColorTString("Failed to decode image", tcell.ColorRed)}
				msg.cachedWidth = width
				msg.cachedProto = effectiveProto
				msg.cachedOSC = nil
				msg.cachedKitty = nil
				msg.imageID = 0
				msg.lastEmitted = false
				msg.mu.Unlock()
				return
			}
		}

		targetPxW := bbox.Cols * 10
		targetPxH := bbox.Rows * 20
		resized := imaging.Fit(img, targetPxW, targetPxH, imaging.Lanczos)

		id := termimg.NextKittyImageID()
		isTmux := termimg.DetectTerminal().IsTmux
		kittySeq := termimg.FormatKittyTransmit(resized, id, isTmux)
		placeholder := termimg.CreateKittyPlaceholderBuffer(bbox.Cols, bbox.Rows, id)

		msg.mu.Lock()
		msg.buffer = placeholder
		msg.cachedKitty = kittySeq
		msg.imageID = id
		msg.cachedWidth = width
		msg.cachedProto = effectiveProto
		msg.cachedOSC = nil
		msg.lastEmitted = false
		msg.mu.Unlock()

	case termimg.ProtocolITerm2:
		// Tier 2: iTerm2 via OSC 1337
		downscaled, downErr := termimg.PreDownscaleForPTY(dataCopy, 1920, 1080)
		if downErr != nil {
			downscaled = dataCopy
		}
		isTmux := termimg.DetectTerminal().IsTmux
		oscBytes := termimg.FormatOSC1337(downscaled, bbox.Cols, bbox.Rows, isTmux)
		placeholder := termimg.CreatePlaceholderBuffer(bbox.Cols, bbox.Rows)

		// Pre-populate renderCache with half-blocks for viewport clipping fallback
		msg.mu.Lock()
		if msg.renderCache == nil {
			msg.renderCache = make(map[int][]tstring.TString)
		}
		cachedHalfblocks, hasHalfblocks := msg.renderCache[bbox.Cols]
		msg.mu.Unlock()

		if !hasHalfblocks || len(cachedHalfblocks) != bbox.Rows {
			pixelHeight := bbox.Rows * 2
			pixelWidth := bbox.Cols
			ansFile, aErr := ansimage.NewScaledFromReader(bytes.NewReader(dataCopy), pixelHeight, pixelWidth, color.Transparent)
			if aErr == nil {
				rendered := ansFile.Render()
				msg.mu.Lock()
				if msg.renderCache == nil {
					msg.renderCache = make(map[int][]tstring.TString)
				}
				msg.renderCache[bbox.Cols] = rendered
				msg.mu.Unlock()
			}
		}

		msg.mu.Lock()
		msg.buffer = placeholder
		msg.cachedOSC = oscBytes
		msg.cachedKitty = nil
		msg.imageID = 0
		msg.cachedWidth = width
		msg.cachedProto = effectiveProto
		msg.lastEmitted = false
		msg.mu.Unlock()

	default:
		// Tier 3: Universal TrueColor Half-Blocks
		msg.mu.Lock()
		if msg.renderCache == nil {
			msg.renderCache = make(map[int][]tstring.TString)
		}
		cachedBuf, found := msg.renderCache[bbox.Cols]
		if found && len(cachedBuf) == bbox.Rows {
			msg.buffer = cachedBuf
			msg.cachedWidth = width
			msg.cachedProto = effectiveProto
			msg.cachedOSC = nil
			msg.cachedKitty = nil
			msg.imageID = 0
			msg.lastEmitted = false
			msg.mu.Unlock()
			return
		}
		msg.mu.Unlock()

		pixelHeight := bbox.Rows * 2
		pixelWidth := bbox.Cols
		ansFile, aErr := ansimage.NewScaledFromReader(bytes.NewReader(dataCopy), pixelHeight, pixelWidth, color.Transparent)
		if aErr != nil {
			debug.Print("Failed to render ansimage:", aErr)
			msg.mu.Lock()
			msg.imageErr = aErr
			msg.buffer = []tstring.TString{tstring.NewColorTString("Failed to display image", tcell.ColorRed)}
			msg.cachedWidth = width
			msg.cachedProto = effectiveProto
			msg.cachedOSC = nil
			msg.cachedKitty = nil
			msg.imageID = 0
			msg.lastEmitted = false
			msg.mu.Unlock()
			return
		}

		rendered := ansFile.Render()
		msg.mu.Lock()
		if msg.renderCache == nil {
			msg.renderCache = make(map[int][]tstring.TString)
		}
		msg.renderCache[bbox.Cols] = rendered
		msg.buffer = rendered
		msg.cachedWidth = width
		msg.cachedProto = effectiveProto
		msg.cachedOSC = nil
		msg.cachedKitty = nil
		msg.imageID = 0
		msg.lastEmitted = false
		msg.mu.Unlock()
	}
}

func (msg *FileMessage) Height() int {
	msg.mu.RLock()
	defer msg.mu.RUnlock()
	return len(msg.buffer)
}

func (msg *FileMessage) Draw(screen mauview.Screen, _ *UIMessage) {
	msg.mu.Lock()
	defer msg.mu.Unlock()

	if len(msg.buffer) == 0 {
		return
	}

	rootScreen, screenX, screenY := termimg.GetRootScreenAndOffset(screen)

	switch msg.cachedProto {
	case termimg.ProtocolKitty:
		if len(msg.cachedKitty) > 0 && !msg.lastEmitted {
			msg.lastEmitted = true
			emitKitty(rootScreen, msg.cachedKitty)
		}
		for y, line := range msg.buffer {
			line.Draw(screen, 0, y)
		}
		return

	case termimg.ProtocolITerm2:
		if rootScreen == nil || len(msg.cachedOSC) == 0 {
			if msg.mask != nil {
				msg.mask.Clear()
				msg.mask = nil
			}
			for y, line := range msg.buffer {
				line.Draw(screen, 0, y)
			}
			return
		}

		rows := len(msg.buffer)
		cols := 0
		if rows > 0 {
			cols = len(msg.buffer[0])
		}
		if cols == 0 || rows == 0 {
			return
		}

		// Determine visible viewport height
		rootW, rootH := rootScreen.Size()
		maxAllowedY := rootH - 1
		if proxy, ok := screen.(*mauview.ProxyScreen); ok && proxy.Parent != nil {
			_, parentH := proxy.Parent.Size()
			if parentH > 0 && 1+parentH < maxAllowedY {
				maxAllowedY = 1 + parentH
			}
		}

		isPartiallyClipped := screenY < 1 || (screenY+rows) > maxAllowedY || screenX < 0 || (screenX+cols) > rootW

		if isPartiallyClipped {
			if msg.mask != nil {
				msg.mask.Clear()
				msg.mask = nil
			}
			msg.lastEmitted = false

			if cachedHalfblocks, ok := msg.renderCache[cols]; ok && len(cachedHalfblocks) == rows {
				for y, line := range cachedHalfblocks {
					line.Draw(screen, 0, y)
				}
			} else {
				for y, line := range msg.buffer {
					line.Draw(screen, 0, y)
				}
			}
			return
		}

		// Apply cell mask
		if msg.mask == nil || msg.mask.RootScreen != rootScreen || msg.mask.ScreenX != screenX || msg.mask.ScreenY != screenY || msg.mask.Cols != cols || msg.mask.Rows != rows {
			if msg.mask != nil {
				msg.mask.Clear()
			}
			msg.mask = termimg.NewImageMask(screen, 0, 0, cols, rows)
		}
		msg.mask.Apply()

		for y, line := range msg.buffer {
			line.Draw(screen, 0, y)
		}

		if screenX != msg.lastDrawnX || screenY != msg.lastDrawnY || cols != msg.lastDrawnW || rows != msg.lastDrawnH || !msg.lastEmitted {
			msg.lastDrawnX = screenX
			msg.lastDrawnY = screenY
			msg.lastDrawnW = cols
			msg.lastDrawnH = rows
			msg.lastEmitted = true

			emitOSC1337(rootScreen, screenX, screenY, msg.cachedOSC)
		}

	default:
		if msg.mask != nil {
			msg.mask.Clear()
			msg.mask = nil
		}
		for y, line := range msg.buffer {
			line.Draw(screen, 0, y)
		}
	}
}

func emitKitty(rootScreen tcell.Screen, seq []byte) {
	if len(seq) == 0 {
		return
	}
	if oscOutputWriter != nil {
		_, _ = oscOutputWriter.Write(seq)
		return
	}
	if rootScreen != nil {
		if tty, ok := rootScreen.Tty(); ok && tty != nil {
			_, _ = tty.Write(seq)
			return
		}
		if _, isSim := rootScreen.(tcell.SimulationScreen); isSim {
			return
		}
	}
	_, _ = os.Stdout.Write(seq)
}

func emitOSC1337(rootScreen tcell.Screen, x, y int, oscSeq []byte) {
	if len(oscSeq) == 0 {
		return
	}
	cup := fmt.Sprintf("\x1b[%d;%dH", y+1, x+1)
	if oscOutputWriter != nil {
		_, _ = oscOutputWriter.Write([]byte(cup))
		_, _ = oscOutputWriter.Write(oscSeq)
		return
	}
	if rootScreen != nil {
		if tty, ok := rootScreen.Tty(); ok && tty != nil {
			_, _ = tty.Write([]byte(cup))
			_, _ = tty.Write(oscSeq)
			return
		}
		if _, isSim := rootScreen.(tcell.SimulationScreen); isSim {
			return
		}
	}
	_, _ = os.Stdout.WriteString(cup)
	_, _ = os.Stdout.Write(oscSeq)
}

// CurrentMask returns the active image mask, if any.
func (msg *FileMessage) CurrentMask() *termimg.ImageMask {
	msg.mu.RLock()
	defer msg.mu.RUnlock()
	return msg.mask
}

// ClearMask releases any active cell-masking lock.
func (msg *FileMessage) ClearMask() {
	msg.mu.Lock()
	defer msg.mu.Unlock()
	if msg.mask != nil {
		msg.mask.Clear()
		msg.mask = nil
	}
}

// RenderCache returns a copy of the rendered buffer cache for inspection in tests.
func (msg *FileMessage) RenderCache() map[int][]tstring.TString {
	msg.mu.RLock()
	defer msg.mu.RUnlock()
	if msg.renderCache == nil {
		return nil
	}
	cp := make(map[int][]tstring.TString, len(msg.renderCache))
	for k, v := range msg.renderCache {
		lines := make([]tstring.TString, len(v))
		for i, l := range v {
			lines[i] = l.Clone()
		}
		cp[k] = lines
	}
	return cp
}

// CachedOSC returns a copy of the cached OSC 1337 escape sequence.
func (msg *FileMessage) CachedOSC() []byte {
	msg.mu.RLock()
	defer msg.mu.RUnlock()
	if msg.cachedOSC == nil {
		return nil
	}
	cp := make([]byte, len(msg.cachedOSC))
	copy(cp, msg.cachedOSC)
	return cp
}

// CachedProto returns the cached protocol.
func (msg *FileMessage) CachedProto() termimg.Protocol {
	msg.mu.RLock()
	defer msg.mu.RUnlock()
	return msg.cachedProto
}
