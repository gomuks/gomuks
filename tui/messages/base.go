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
	"fmt"
	"maps"
	"slices"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell/v2"
	"go.mau.fi/mauview"
	"maunium.net/go/mautrix/event"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/rpc/store"
	"go.mau.fi/gomuks/tui/config"
	"go.mau.fi/gomuks/tui/widget"
)

type MessageRenderer interface {
	Draw(screen mauview.Screen, msg *UIMessage)
	NotificationContent() string
	PlainText() string
	CalculateBuffer(prefs config.UserPreferences, width int, msg *UIMessage)
	Height() int
	Clone() MessageRenderer
	String() string
}

type ReactionItem struct {
	Key   string
	Count int
}

func (ri ReactionItem) String() string {
	return fmt.Sprintf("%d×%s", ri.Count, ri.Key)
}

type ReactionSlice []ReactionItem

func (rs ReactionSlice) Len() int {
	return len(rs)
}

func (rs ReactionSlice) Less(i, j int) bool {
	return rs[i].Key < rs[j].Key
}

func (rs ReactionSlice) Swap(i, j int) {
	rs[i], rs[j] = rs[j], rs[i]
}

type UIMessage struct {
	*database.Event
	Room               *store.RoomStore
	MsgType            event.MessageType
	OverrideSenderName string
	DefaultSenderColor tcell.Color
	IsService          bool
	IsSelected         bool
	ReplyTo            *UIMessage
	IsReplyBubble      bool
	Renderer           MessageRenderer
	bufferedWidth      int32
}

// InvalidateBuffer marks the cached render buffer as invalid so CalculateBuffer will recompute.
func (msg *UIMessage) InvalidateBuffer() {
	if msg != nil {
		atomic.StoreInt32(&msg.bufferedWidth, 0)
	}
}

// BufferedWidth returns the currently cached buffer width atomically.
func (msg *UIMessage) BufferedWidth() int {
	if msg == nil {
		return 0
	}
	return int(atomic.LoadInt32(&msg.bufferedWidth))
}

func (msg *UIMessage) GetEvent() *database.Event {
	if msg == nil {
		return nil
	}
	return msg.Event
}

const DateFormat = "January _2, 2006"
const TimeFormat = "15:04:05"

func newUIMessage(
	room *store.RoomStore,
	evt *database.Event,
	msgContent *event.MessageEventContent,
	displayname string,
	renderer MessageRenderer,
) *UIMessage {
	msgtype := msgContent.MsgType
	if len(msgtype) == 0 {
		msgtype = event.MessageType(evt.Type)
	}

	return &UIMessage{
		Room:               room,
		OverrideSenderName: displayname,
		DefaultSenderColor: widget.GetHashColor(evt.Sender),
		MsgType:            msgtype,
		IsService:          false,
		Event:              evt,
		Renderer:           renderer,
	}
}

// GetSenderName gets the string that should be displayed as the sender of this message.
//
// If the message is being sent, the sender is "Sending...".
// If sending has failed, the sender is "Error".
// If the message is an emote, the sender is blank.
// In any other case, the sender is the display name of the user who sent the message.
func (msg *UIMessage) GetSenderName() string {
	if msg.Event.SendError != "" && msg.Event.SendError != "not sent" {
		return "Error"
	} else if msg.Event.Pending {
		return "Sending..."
	}
	switch msg.MsgType {
	case "m.emote":
		// Emotes don't show a separate sender, it's included in the buffer.
		return ""
	default:
		return msg.GetRawSenderName()
	}
}

func (msg *UIMessage) GetRawSenderName() string {
	if msg.OverrideSenderName != "" {
		return msg.OverrideSenderName
	} else if msgContent, ok := msg.GetMautrixContent().Parsed.(*event.MessageEventContent); ok && msgContent.BeeperPerMessageProfile != nil && msgContent.BeeperPerMessageProfile.Displayname != "" {
		return msgContent.BeeperPerMessageProfile.Displayname
	}
	return msg.Room.GetDisplayname(msg.Sender)
}

func (msg *UIMessage) NotificationContent() string {
	return msg.Renderer.NotificationContent()
}

func (msg *UIMessage) getStateSpecificColor() tcell.Color {
	if msg.Event.SendError != "" && msg.Event.SendError != "not sent" {
		return tcell.ColorRed
	} else if msg.Event.Pending {
		return tcell.ColorGray
	}
	return tcell.ColorDefault
}

// SenderColor returns the color the name of the sender should be shown in.
//
// If the message is being sent, the color is gray.
// If sending has failed, the color is red.
//
// In any other case, the color is whatever is specified in the Message struct.
// Usually that means it is the hash-based color of the sender (see ui/widget/color.go)
func (msg *UIMessage) SenderColor() tcell.Color {
	stateColor := msg.getStateSpecificColor()
	switch {
	case stateColor != tcell.ColorDefault:
		return stateColor
	//case msg.Type == event.StateMember.Type:
	//	return widget.GetHashColor(msg.SenderName)
	case msg.IsService:
		return tcell.ColorGray
	default:
		return msg.DefaultSenderColor
	}
}

// TextColor returns the color the actual content of the message should be shown in.
func (msg *UIMessage) TextColor() tcell.Color {
	stateColor := msg.getStateSpecificColor()
	switch {
	case stateColor != tcell.ColorDefault:
		return stateColor
	case msg.IsService, msg.MsgType == event.MsgNotice:
		return tcell.ColorGray
	case msg.UnreadType.Is(database.UnreadTypeHighlight):
		return tcell.ColorYellow
	case msg.Type == event.StateMember.Type:
		return tcell.ColorGreen
	default:
		return tcell.ColorDefault
	}
}

// TimestampColor returns the color the timestamp should be shown in.
//
// As with SenderColor(), messages being sent and messages that failed to be sent are
// gray and red respectively.
//
// However, other messages are the default color instead of a color stored in the struct.
func (msg *UIMessage) TimestampColor() tcell.Color {
	if msg.IsService {
		return tcell.ColorGray
	}
	return msg.getStateSpecificColor()
}

func (msg *UIMessage) ReplyHeight() int {
	if msg.ReplyTo != nil {
		return 1 + msg.ReplyTo.Height()
	}
	return 0
}

func (msg *UIMessage) ReactionHeight() int {
	if len(msg.Event.Reactions) > 0 && !msg.IsReplyBubble {
		return 1
	}
	return 0
}

// Height returns the number of rows in the computed buffer (see Buffer()).
func (msg *UIMessage) Height() int {
	return msg.ReplyHeight() + msg.Renderer.Height() + msg.ReactionHeight()
}

func (msg *UIMessage) Time() time.Time {
	return msg.Timestamp.Time
}

// FormatTime returns the formatted time when the message was sent.
func (msg *UIMessage) FormatTime() string {
	return msg.Timestamp.Format(TimeFormat)
}

// FormatDate returns the formatted date when the message was sent.
func (msg *UIMessage) FormatDate() string {
	return msg.Timestamp.Format(DateFormat)
}

func (msg *UIMessage) SameDate(message *UIMessage) bool {
	if message == nil {
		return false
	}
	year1, month1, day1 := msg.Timestamp.Date()
	year2, month2, day2 := message.Timestamp.Date()
	return day1 == day2 && month1 == month2 && year1 == year2
}

func (msg *UIMessage) DrawReactions(screen mauview.Screen) {
	if len(msg.Event.Reactions) == 0 || msg.IsReplyBubble {
		return
	}
	width, height := screen.Size()
	screen = mauview.NewProxyScreen(screen, 0, height-1, width, 1)

	x := 0
	reactionKeys := slices.Sorted(maps.Keys(msg.Event.Reactions))
	for _, reaction := range reactionKeys {
		count := msg.Event.Reactions[reaction]
		if count == 0 {
			continue
		}
		_, drawn := mauview.PrintWithStyle(screen, fmt.Sprintf("%d×%s", count, reaction), x, 0, width-x, mauview.AlignLeft, tcell.StyleDefault.Foreground(mauview.Styles.PrimaryTextColor).Background(tcell.ColorDarkGreen))
		x += drawn + 1
		if x >= width {
			break
		}
	}
}

func (msg *UIMessage) Draw(screen mauview.Screen) {
	proxyScreen := msg.DrawReply(screen)
	msg.Renderer.Draw(proxyScreen, msg)
	msg.DrawReactions(proxyScreen)
	if msg.IsSelected {
		w, h := screen.Size()
		for x := 0; x < w; x++ {
			for y := 0; y < h; y++ {
				mainc, combc, style, _ := screen.GetContent(x, y)
				_, bg, _ := style.Decompose()
				if bg == tcell.ColorDefault {
					screen.SetContent(x, y, mainc, combc, style.Background(tcell.ColorDarkGreen))
				}
			}
		}
	}
}

func (msg *UIMessage) Clone() *UIMessage {
	clone := &UIMessage{
		Event:              msg.Event,
		Room:               msg.Room,
		MsgType:            msg.MsgType,
		OverrideSenderName: msg.OverrideSenderName,
		DefaultSenderColor: msg.DefaultSenderColor,
		IsService:          msg.IsService,
		IsSelected:         msg.IsSelected,
		ReplyTo:            nil,
		IsReplyBubble:      msg.IsReplyBubble,
		bufferedWidth:      atomic.LoadInt32(&msg.bufferedWidth),
	}
	if msg.Renderer != nil {
		clone.Renderer = msg.Renderer.Clone()
		if fileMsg, ok := clone.Renderer.(*FileMessage); ok {
			fileMsg.SetUIMessage(clone)
		}
	}
	return clone
}

func (msg *UIMessage) calculateReplyBuffer(preferences config.UserPreferences, width int) {
	if msg.ReplyTo == nil {
		return
	}
	msg.ReplyTo.CalculateBuffer(preferences, width-1)
}

func (msg *UIMessage) CalculateBuffer(preferences config.UserPreferences, width int) {
	// TODO check preferences (at least disable images and bare message view)
	if int(atomic.LoadInt32(&msg.bufferedWidth)) == width {
		return
	}
	msg.Renderer.CalculateBuffer(preferences, width, msg)
	msg.calculateReplyBuffer(preferences, width)
	atomic.StoreInt32(&msg.bufferedWidth, int32(width))
}

func (msg *UIMessage) DrawReply(screen mauview.Screen) mauview.Screen {
	if msg.ReplyTo == nil {
		return screen
	}
	width, height := screen.Size()
	replyHeight := msg.ReplyTo.Height()
	widget.WriteLineSimpleColor(screen, "In reply to", 1, 0, tcell.ColorGreen)
	widget.WriteLineSimpleColor(screen, msg.ReplyTo.GetRawSenderName(), 13, 0, msg.ReplyTo.SenderColor())
	for y := 0; y < 1+replyHeight; y++ {
		screen.SetCell(0, y, tcell.StyleDefault, '▊')
	}
	replyScreen := mauview.NewProxyScreen(screen, 1, 1, width-1, replyHeight)
	msg.ReplyTo.Draw(replyScreen)
	return mauview.NewProxyScreen(screen, 0, replyHeight+1, width, height-replyHeight-1)
}

func (msg *UIMessage) String() string {
	return fmt.Sprintf(`&messages.UIMessage{
    ID="%s", TxnID="%s",
    MsgType="%s", Timestamp=%s,
    Sender={ID="%s", OverrideName="%s", Color=#%X},
    IsService=%t,
    Renderer=%s,
}`,
		msg.ID, msg.TransactionID,
		msg.MsgType, msg.Timestamp.String(),
		msg.Sender, msg.OverrideSenderName, msg.DefaultSenderColor.Hex(),
		msg.IsService, msg.Renderer.String())
}

func (msg *UIMessage) PlainText() string {
	return msg.Renderer.PlainText()
}
