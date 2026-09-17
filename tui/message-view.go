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

package tui

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"go.mau.fi/mauview"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/rpc/client"
	"go.mau.fi/gomuks/tui/config"
	"go.mau.fi/gomuks/tui/messages"
	"go.mau.fi/gomuks/tui/widget"
)

type MessageView struct {
	parent *RoomView
	config *config.Config
	matrix *client.GomuksClient
	lock   sync.RWMutex

	SenderWidth     int
	DateFormat      string
	TimestampFormat string
	TimestampWidth  int

	ScrollOffset atomic.Int32
	height       atomic.Uint32
	totalHeight  atomic.Uint32

	msgBuffer    []*messages.UIMessage
	prevTimeline *[]*database.Event
	prevWidth    int
	selected     database.EventRowID

	invalidated  atomic.Bool
}

// InvalidateTimeline forces the next update() call to recalculate message layout heights without resetting prevTimeline.
// Thread-safe, wait-free atomic store; does not require or block on view.lock.
func (view *MessageView) InvalidateTimeline() {
	view.invalidated.Store(true)
}

func NewMessageView(parent *RoomView) *MessageView {
	mv := &MessageView{
		parent: parent,
		config: parent.config,
		matrix: parent.parent.matrix,

		SenderWidth:    15,
		TimestampWidth: len(messages.TimeFormat),
	}
	return mv
}

func (view *MessageView) SetSelected(message *messages.UIMessage) {
	view.lock.Lock()
	defer view.lock.Unlock()
	if message == nil || (view.selected == message.RowID || message.IsService) {
		view.selected = 0
	} else {
		view.selected = message.RowID
	}
}

func (view *MessageView) GetSelected() *messages.UIMessage {
	view.lock.RLock()
	defer view.lock.RUnlock()
	evt := view.parent.Room.GetEventByRowID(view.selected)
	if evt == nil || evt.RenderMeta == nil {
		return nil
	}
	return evt.RenderMeta.(*messages.UIMessage)
}

func (view *MessageView) handleMessageClick(message *messages.UIMessage, mod tcell.ModMask) bool {
	//if msg, ok := message.Renderer.(*messages.FileMessage); ok && mod > 0 && !msg.Thumbnail.IsEmpty() {
	//	debug.Print("Opening thumbnail", msg.ThumbnailPath())
	//	open.Open(msg.ThumbnailPath())
	//	// No need to re-render
	//	return false
	//}
	view.SetSelected(message)
	view.parent.OnSelect(message)
	return true
}

func (view *MessageView) handleUsernameClick(message *messages.UIMessage, prevMessage *messages.UIMessage) bool {
	// TODO this is needed if senders are hidden for messages from the same sender (see Draw method)
	//if prevMessage != nil && prevMessage.SenderName == message.SenderName {
	//	return false
	//}

	senderName := message.GetRawSenderName()
	if senderName == "---" || senderName == "-->" || senderName == "<--" || message.MsgType == event.MsgEmote {
		return false
	}

	sender := format.MarkdownMentionWithName(senderName, message.Sender)

	cursorPos := view.parent.input.GetCursorOffset()
	text := view.parent.input.GetText()
	var buf strings.Builder
	if cursorPos == 0 {
		buf.WriteString(sender)
		buf.WriteRune(':')
		buf.WriteRune(' ')
		buf.WriteString(text)
	} else {
		textBefore := runewidth.Truncate(text, cursorPos, "")
		textAfter := text[len(textBefore):]
		buf.WriteString(textBefore)
		buf.WriteString(sender)
		buf.WriteRune(' ')
		buf.WriteString(textAfter)
	}
	newText := buf.String()
	view.parent.input.SetText(string(newText))
	view.parent.input.SetCursorOffset(cursorPos + len(newText) - len(text))
	return true
}

func (view *MessageView) GetScrollOffset() int {
	return int(view.ScrollOffset.Load())
}

func (view *MessageView) OnMouseEvent(event mauview.MouseEvent) bool {
	if event.HasMotion() {
		return false
	}
	switch event.Buttons() {
	case tcell.WheelUp:
		if view.IsAtTop() {
			go view.parent.parent.LoadHistory(view.parent.Room.ID)
		} else {
			view.AddScrollOffset(WheelScrollOffsetDiff)
			return true
		}
	case tcell.WheelDown:
		view.AddScrollOffset(-WheelScrollOffsetDiff)
		view.parent.parent.MarkRead(view.parent)
		return true
	case tcell.Button1:
		x, y := event.Position()
		line := view.TotalHeight() - int(view.ScrollOffset.Load()) - view.Height() + y
		if line < 0 || line >= view.TotalHeight() {
			return false
		}

		view.lock.RLock()
		message := view.msgBuffer[line]
		var prevMessage *messages.UIMessage
		if y != 0 && line > 0 {
			prevMessage = view.msgBuffer[line-1]
		}
		view.lock.RUnlock()

		usernameX := 0
		if !view.config.Preferences.HideTimestamp {
			usernameX += view.TimestampWidth + TimestampSenderGap
		}
		messageX := usernameX + view.SenderWidth + SenderMessageGap

		if x >= messageX {
			return view.handleMessageClick(message, event.Modifiers())
		} else if x >= usernameX && x < messageX-SenderMessageGap {
			return view.handleUsernameClick(message, prevMessage)
		}
	}
	return false
}

const PaddingAtTop = 5

func (view *MessageView) AddScrollOffset(diff int) {
	totalHeight := view.TotalHeight()
	height := view.Height()
	scrollOffset := int(view.ScrollOffset.Load())
	if diff >= 0 && scrollOffset+diff >= totalHeight-height+PaddingAtTop {
		scrollOffset = totalHeight - height + PaddingAtTop
	} else {
		scrollOffset += diff
	}

	if scrollOffset > totalHeight-height+PaddingAtTop {
		scrollOffset = totalHeight - height + PaddingAtTop
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}
	view.ScrollOffset.Store(int32(scrollOffset))
}

func (view *MessageView) Height() int {
	return int(view.height.Load())
}

func (view *MessageView) TotalHeight() int {
	return int(view.totalHeight.Load())
}

func (view *MessageView) IsAtTop() bool {
	return view.GetScrollOffset() >= view.TotalHeight()-view.Height()+PaddingAtTop
}

const (
	TimestampSenderGap = 1
	SenderSeparatorGap = 1
	SenderMessageGap   = 3
)

func getScrollbarStyle(scrollbarHere, isTop, isBottom bool) (char rune, style tcell.Style) {
	char = '│'
	style = tcell.StyleDefault
	if scrollbarHere {
		style = style.Foreground(tcell.ColorGreen)
	}
	if isTop {
		if scrollbarHere {
			char = '╥'
		} else {
			char = '┬'
		}
	} else if isBottom {
		if scrollbarHere {
			char = '╨'
		} else {
			char = '┴'
		}
	} else if scrollbarHere {
		char = '║'
	}
	return
}

func (view *MessageView) calculateScrollBar(height int) (scrollBarHeight, scrollBarPos int) {
	viewportHeight := float64(height)
	contentHeight := float64(view.TotalHeight())

	scrollBarHeight = int(math.Ceil(viewportHeight / (contentHeight / viewportHeight)))

	scrollBarPos = height - int(math.Round(float64(view.GetScrollOffset())/contentHeight*viewportHeight))

	return
}

func (view *MessageView) getIndexOffset(screen mauview.Screen, height, messageX int) (indexOffset int) {
	indexOffset = view.TotalHeight() - view.GetScrollOffset() - height
	if indexOffset <= -PaddingAtTop {
		message := "Scroll up to load more messages."
		if view.parent.Room.Paginating.Load() {
			message = "Loading more messages..."
		}
		widget.WriteLineSimpleColor(screen, message, messageX, 0, tcell.ColorGreen)
	}
	return
}

func (view *MessageView) CapturePlaintext(height int) string {
	var buf strings.Builder
	indexOffset := view.TotalHeight() - view.GetScrollOffset() - height
	var prevMessage *messages.UIMessage
	view.lock.RLock()
	for line := 0; line < height; line++ {
		index := indexOffset + line
		if index < 0 {
			continue
		}

		message := view.msgBuffer[index]
		if message != prevMessage {
			var sender string
			if len(message.GetSenderName()) > 0 {
				sender = fmt.Sprintf(" <%s>", message.GetSenderName())
			} else if message.MsgType == event.MsgEmote {
				sender = fmt.Sprintf(" * %s", message.GetRawSenderName())
			}
			fmt.Fprintf(&buf, "%s%s %s\n", message.FormatTime(), sender, message.PlainText())
			prevMessage = message
		}
	}
	view.lock.RUnlock()
	return buf.String()
}

// Draw renders the message timeline to the screen.
// Phase 1: Calls view.update(width) before acquiring view.lock, preventing recursive mutex deadlocks.
// Phase 2: Acquires view.lock.RLock() to safely iterate view.msgBuffer.
func (view *MessageView) Draw(screen mauview.Screen) {
	width, height := screen.Size()
	view.height.Store(uint32(height))
	view.update(width)

	view.lock.RLock()
	defer view.lock.RUnlock()
	scrollOffset := view.GetScrollOffset()

	if len(view.msgBuffer) == 0 {
		widget.WriteLineSimple(screen, "It's quite empty in here.", 0, height)
		return
	}

	usernameX := 0
	if !view.config.Preferences.HideTimestamp {
		usernameX += view.TimestampWidth + TimestampSenderGap
	}
	messageX := usernameX + view.SenderWidth + SenderMessageGap

	bareMode := view.config.Preferences.BareMessageView
	if bareMode {
		messageX = 0
	}

	indexOffset := view.getIndexOffset(screen, height, messageX)

	viewStart := 0
	if indexOffset < 0 {
		viewStart = -indexOffset
	}

	if !bareMode {
		separatorX := usernameX + view.SenderWidth + SenderSeparatorGap
		scrollBarHeight, scrollBarPos := view.calculateScrollBar(height)

		for line := viewStart; line < height; line++ {
			showScrollbar := line-viewStart >= scrollBarPos-scrollBarHeight && line-viewStart < scrollBarPos
			isTop := line == viewStart && scrollOffset+height >= view.TotalHeight()
			isBottom := line == height-1 && scrollOffset == 0

			borderChar, borderStyle := getScrollbarStyle(showScrollbar, isTop, isBottom)

			screen.SetContent(separatorX, line, borderChar, nil, borderStyle)
		}
	}

	isFirst := true
	for line := viewStart; line < height && indexOffset+line < len(view.msgBuffer); {
		index := indexOffset + line

		msg := view.msgBuffer[index]
		if isFirst {
			isFirst = false
			for i := index - 1; i >= 0 && view.msgBuffer[i] == msg; i-- {
				line--
			}
		}

		if len(msg.FormatTime()) > 0 && !view.config.Preferences.HideTimestamp {
			widget.WriteLineSimpleColor(screen, msg.FormatTime(), 0, line, msg.TimestampColor())
		}
		widget.WriteLineColor(
			screen, mauview.AlignRight, msg.GetSenderName(),
			usernameX, line, view.SenderWidth,
			msg.SenderColor())
		if msg.LastEditRef != nil {
			screen.SetCell(usernameX+view.SenderWidth, line, tcell.StyleDefault.Foreground(tcell.ColorDarkRed), '*')
		}

		msg.IsSelected = view.selected != 0 && msg.RowID == view.selected
		msg.Draw(mauview.NewProxyScreen(screen, messageX, line, width-messageX, msg.Height()))

		nextIndex := index + 1
		for nextIndex < len(view.msgBuffer) && view.msgBuffer[nextIndex] == msg {
			nextIndex++
		}
		line = nextIndex - indexOffset
	}
}

// update recalculates message buffer layout.
// Thread-safe: fully protected under view.lock.Lock(). Callers must NOT hold view.lock when calling.
func (view *MessageView) update(width int) {
	view.lock.Lock()
	defer view.lock.Unlock()

	timelinePtr := view.parent.Room.TimelineCache.Current()
	invalidated := view.invalidated.Load()
	if timelinePtr == nil || (!invalidated && timelinePtr == view.prevTimeline && width == view.prevWidth) {
		return
	}
	view.invalidated.Store(false)
	timeline := *timelinePtr
	var prevTimeline []*database.Event
	if view.prevTimeline != nil {
		prevTimeline = *view.prevTimeline
	}

	newBuffer := make([]*messages.UIMessage, 0, len(timeline)*2)
	var lastRowIDInPrevTimeline database.EventRowID
	if len(prevTimeline) > 0 {
		lastRowIDInPrevTimeline = prevTimeline[len(prevTimeline)-1].RowID
	}
	increaseScrollOffset := false

	// Preserve outer screen width; compute inner column width separately to preserve frame caching.
	contentWidth := width
	bare := view.config.Preferences.BareMessageView
	if !bare {
		contentWidth -= view.SenderWidth + SenderMessageGap
		if !view.config.Preferences.HideTimestamp {
			contentWidth -= view.TimestampWidth + TimestampSenderGap
		}
	}
	scrollOffset := view.GetScrollOffset()
	newScrollOffset := scrollOffset
	appendBuffer := func(msg *messages.UIMessage) {
		if contentWidth < 5 {
			return
		}
		msg.CalculateBuffer(view.config.Preferences, contentWidth)
		height := msg.Height()
		for i := 0; i < height; i++ {
			newBuffer = append(newBuffer, msg)
		}
		if scrollOffset > 0 && increaseScrollOffset {
			newScrollOffset += height
		}
	}
	var prev *messages.UIMessage
	prevLastEventNotFound := lastRowIDInPrevTimeline != 0
	for _, evt := range timeline {
		startIncreasingScrollOffset := false
		if !increaseScrollOffset && scrollOffset > 0 && evt.RowID != 0 && evt.RowID == lastRowIDInPrevTimeline {
			startIncreasingScrollOffset = true
			prevLastEventNotFound = true
		}
		if evt.RenderMeta == nil {
			evt.RenderMeta = messages.ParseEvent(view.matrix, &view.config.Preferences, view.parent.Room, evt)
		}
		uiMsg := evt.RenderMeta.(*messages.UIMessage)
		if uiMsg == nil {
			continue
		}
		if !uiMsg.SameDate(prev) {
			dateChange := messages.NewDateChangeMessage(view.parent.Room, fmt.Sprintf("Date changed to %s", uiMsg.FormatDate()))
			appendBuffer(dateChange)
		}
		appendBuffer(uiMsg)
		prev = uiMsg
		if startIncreasingScrollOffset {
			increaseScrollOffset = true
		}
	}
	if scrollOffset > 0 && !increaseScrollOffset && !prevLastEventNotFound {
		// Previous last message wasn't found, so reset scroll position
		newScrollOffset = 0
	}
	if newScrollOffset != scrollOffset {
		view.ScrollOffset.Store(int32(newScrollOffset))
	}
	view.msgBuffer = newBuffer
	view.totalHeight.Store(uint32(len(newBuffer)))
	view.prevTimeline = timelinePtr
	view.prevWidth = width
}
