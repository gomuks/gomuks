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
	"context"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/zyedidia/clipboard"
	"go.mau.fi/mauview"
	"go.mau.fi/util/ptr"
	"go.mau.fi/util/variationselector"
	"maunium.net/go/mautrix/crypto/attachment"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
	"go.mau.fi/gomuks/pkg/rpc/store"
	"go.mau.fi/gomuks/tui/config"
	"go.mau.fi/gomuks/tui/debug"
	"go.mau.fi/gomuks/tui/messages"
	"go.mau.fi/gomuks/tui/widget"
)

type RoomView struct {
	topic    *mauview.TextView
	content  *MessageView
	status   *mauview.TextField
	userList *MemberList
	ulBorder *widget.Border
	input    *mauview.InputArea
	Room     *store.RoomStore

	topicScreen    *mauview.ProxyScreen
	contentScreen  *mauview.ProxyScreen
	statusScreen   *mauview.ProxyScreen
	inputScreen    *mauview.ProxyScreen
	ulBorderScreen *mauview.ProxyScreen
	ulScreen       *mauview.ProxyScreen

	userListLoaded bool

	prevScreen mauview.Screen

	parent *MainView
	config *config.Config

	selecting     bool
	selectReason  SelectReason
	selectContent string

	replying *database.Event

	editing      *database.Event
	editMoveText string

	completions struct {
		list      []string
		textCache string
		time      time.Time
	}

	unlistenMeta     func()
	unlistenTimeline func()
}

func NewRoomView(parent *MainView, room *store.RoomStore) *RoomView {
	view := &RoomView{
		topic:    mauview.NewTextView(),
		status:   mauview.NewTextField(),
		userList: NewMemberList(),
		ulBorder: widget.NewBorder(),
		input:    mauview.NewInputArea(),
		Room:     room,

		topicScreen:    &mauview.ProxyScreen{OffsetX: 0, OffsetY: 0, Height: TopicBarHeight},
		contentScreen:  &mauview.ProxyScreen{OffsetX: 0, OffsetY: StatusBarHeight},
		statusScreen:   &mauview.ProxyScreen{OffsetX: 0, Height: StatusBarHeight},
		inputScreen:    &mauview.ProxyScreen{OffsetX: 0},
		ulBorderScreen: &mauview.ProxyScreen{OffsetY: StatusBarHeight, Width: UserListBorderWidth},
		ulScreen:       &mauview.ProxyScreen{OffsetY: StatusBarHeight, Width: UserListWidth},

		parent: parent,
		config: parent.config,
	}
	view.content = NewMessageView(view)

	view.input.
		SetTextColor(tcell.ColorDefault).
		SetBackgroundColor(tcell.ColorDefault).
		SetPlaceholder("Send a message...").
		SetPlaceholderTextColor(tcell.ColorGray).
		SetTabCompleteFunc(view.InputTabComplete).
		SetPressKeyUpAtStartFunc(view.EditPrevious).
		SetPressKeyDownAtEndFunc(view.EditNext)

	view.topic.
		SetTextColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorDarkGreen)

	view.status.SetBackgroundColor(tcell.ColorDimGray)

	view.Update(room.Meta.Current())

	view.unlistenMeta = room.Meta.Listen(view.Update)
	view.unlistenTimeline = room.TimelineCache.Listen(func(_ *[]*database.Event) {
		view.parent.parent.NeedsRender = true
	})

	return view
}

func (view *RoomView) Unload() {
	view.unlistenTimeline()
	view.unlistenMeta()
}

func (view *RoomView) SetInputChangedFunc(fn func(room *RoomView, text string)) *RoomView {
	view.input.SetChangedFunc(func(text string) {
		fn(view, text)
	})
	return view
}

func (view *RoomView) SetInputText(newText string) *RoomView {
	view.input.SetTextAndMoveCursor(newText)
	return view
}

func (view *RoomView) GetInputText() string {
	return view.input.GetText()
}

func (view *RoomView) Focus() {
	view.input.Focus()
}

func (view *RoomView) Blur() {
	view.StopSelecting()
	view.input.Blur()
}

type SelectReason string

const (
	SelectReply    SelectReason = "reply to"
	SelectReact    SelectReason = "react to"
	SelectRedact   SelectReason = "redact"
	SelectEdit     SelectReason = "edit"
	SelectDownload SelectReason = "download"
	SelectOpen     SelectReason = "open"
	SelectCopy     SelectReason = "copy"
)

func (view *RoomView) StartSelecting(reason SelectReason, content string) {
	view.selecting = true
	view.selectReason = reason
	view.selectContent = content
	msgView := view.MessageView()
	if selected := msgView.GetSelected(); selected != nil {
		view.OnSelect(selected)
	} else {
		view.input.Blur()
		view.SelectPrevious()
	}
}

func (view *RoomView) StopSelecting() {
	view.selecting = false
	view.selectContent = ""
	view.MessageView().SetSelected(nil)
}

func (view *RoomView) OnSelect(message *messages.UIMessage) {
	if !view.selecting || message == nil {
		return
	}
	switch view.selectReason {
	case SelectReply:
		view.replying = message.Event
		if len(view.selectContent) > 0 {
			go view.SendMessage(event.MsgText, view.selectContent)
		}
	case SelectEdit:
		view.SetEditing(message.Event)
	case SelectReact:
		go view.SendReaction(message.ID, view.selectContent)
	case SelectRedact:
		go view.Redact(message.ID, view.selectContent)
	case SelectDownload, SelectOpen:
		//msg, ok := message.Renderer.(*messages.FileMessage)
		//if ok {
		//	path := ""
		//	if len(view.selectContent) > 0 {
		//		path = view.selectContent
		//	} else if view.selectReason == SelectDownload {
		//		path = msg.Body
		//	}
		//	go view.Download(msg.URL, msg.IsEncrypted, path, view.selectReason == SelectOpen)
		//}
	case SelectCopy:
		go view.CopyToClipboard(message.Renderer.PlainText(), view.selectContent)
	}
	view.selecting = false
	view.selectContent = ""
	view.MessageView().SetSelected(nil)
	view.input.Focus()
}

func (view *RoomView) GetStatus() string {
	var buf strings.Builder

	if view.editing != nil {
		buf.WriteString("Editing message - ")
	} else if view.replying != nil {
		buf.WriteString("Replying to ")
		buf.WriteString(string(view.replying.Sender))
		buf.WriteString(" - ")
	} else if view.selecting {
		buf.WriteString("Selecting message to ")
		buf.WriteString(string(view.selectReason))
		buf.WriteString(" - ")
	}

	if len(view.completions.list) > 0 {
		if view.completions.textCache != view.input.GetText() || view.completions.time.Add(10*time.Second).Before(time.Now()) {
			view.completions.list = []string{}
		} else {
			buf.WriteString(strings.Join(view.completions.list, ", "))
			buf.WriteString(" - ")
		}
	}

	typing := view.Room.Typing.Current()
	if len(typing) == 1 {
		buf.WriteString("Typing: " + string(typing[0]))
		buf.WriteString(" - ")
	} else if len(typing) > 1 {
		buf.WriteString("Typing: ")
		for i, userID := range typing {
			if i == len(typing)-1 {
				buf.WriteString(" and ")
			} else if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(string(userID))
		}
		buf.WriteString(" - ")
	}

	return strings.TrimSuffix(buf.String(), " - ")
}

// Constants defining the size of the room view grid.
const (
	UserListBorderWidth   = 1
	UserListWidth         = 20
	StaticHorizontalSpace = UserListBorderWidth + UserListWidth

	TopicBarHeight  = 1
	StatusBarHeight = 1

	MaxInputHeight = 5
)

func (view *RoomView) Draw(screen mauview.Screen) {
	width, height := screen.Size()
	if width <= 0 || height <= 0 {
		return
	}

	if view.prevScreen != screen {
		view.topicScreen.Parent = screen
		view.contentScreen.Parent = screen
		view.statusScreen.Parent = screen
		view.inputScreen.Parent = screen
		view.ulBorderScreen.Parent = screen
		view.ulScreen.Parent = screen
		view.prevScreen = screen
	}

	view.input.PrepareDraw(width)
	inputHeight := view.input.GetTextHeight()
	if inputHeight > MaxInputHeight {
		inputHeight = MaxInputHeight
	} else if inputHeight < 1 {
		inputHeight = 1
	}
	contentHeight := height - inputHeight - TopicBarHeight - StatusBarHeight
	contentWidth := width - StaticHorizontalSpace
	if view.config.Preferences.HideUserList {
		contentWidth = width
	}

	view.topicScreen.Width = width
	view.contentScreen.Width = contentWidth
	view.contentScreen.Height = contentHeight
	view.statusScreen.OffsetY = view.contentScreen.YEnd()
	view.statusScreen.Width = width
	view.inputScreen.Width = width
	view.inputScreen.OffsetY = view.statusScreen.YEnd()
	view.inputScreen.Height = inputHeight
	view.ulBorderScreen.OffsetX = view.contentScreen.XEnd()
	view.ulBorderScreen.Height = contentHeight
	view.ulScreen.OffsetX = view.ulBorderScreen.XEnd()
	view.ulScreen.Height = contentHeight

	// Draw everything
	view.topic.Draw(view.topicScreen)
	view.content.Draw(view.contentScreen)
	view.status.SetText(view.GetStatus())
	view.status.Draw(view.statusScreen)
	view.input.Draw(view.inputScreen)
	if !view.config.Preferences.HideUserList {
		view.ulBorder.Draw(view.ulBorderScreen)
		view.userList.Draw(view.ulScreen)
	}
}

func (view *RoomView) ClearAllContext() {
	view.SetEditing(nil)
	view.StopSelecting()
	view.replying = nil
	view.input.Focus()
}

func (view *RoomView) OnKeyEvent(event mauview.KeyEvent) bool {
	msgView := view.MessageView()
	kb := config.Keybind{
		Key: event.Key(),
		Ch:  event.Rune(),
		Mod: event.Modifiers(),
	}

	if view.selecting {
		switch view.config.Keybindings.Visual[kb] {
		case "clear":
			view.ClearAllContext()
		case "select_prev":
			view.SelectPrevious()
		case "select_next":
			view.SelectNext()
		case "confirm":
			view.OnSelect(msgView.GetSelected())
		default:
			return false
		}
		return true
	}

	switch view.config.Keybindings.Room[kb] {
	case "clear":
		view.ClearAllContext()
		return true
	case "scroll_up":
		if msgView.IsAtTop() {
			go view.parent.LoadHistory(view.Room.ID)
		}
		msgView.AddScrollOffset(+msgView.Height() / 2)
		return true
	case "scroll_down":
		msgView.AddScrollOffset(-msgView.Height() / 2)
		return true
	case "send":
		view.InputSubmit(view.input.GetText())
		return true
	}
	return view.input.OnKeyEvent(event)
}

func (view *RoomView) OnPasteEvent(event mauview.PasteEvent) bool {
	return view.input.OnPasteEvent(event)
}

func (view *RoomView) OnMouseEvent(event mauview.MouseEvent) bool {
	switch {
	case view.contentScreen.IsInArea(event.Position()):
		return view.content.OnMouseEvent(view.contentScreen.OffsetMouseEvent(event))
	case view.topicScreen.IsInArea(event.Position()):
		return view.topic.OnMouseEvent(view.topicScreen.OffsetMouseEvent(event))
	case view.inputScreen.IsInArea(event.Position()):
		return view.input.OnMouseEvent(view.inputScreen.OffsetMouseEvent(event))
	}
	return false
}

func (view *RoomView) SetCompletions(completions []string) {
	view.completions.list = completions
	view.completions.textCache = view.input.GetText()
	view.completions.time = time.Now()
}

//var editHTMLParser = &format.HTMLParser{
//	PillConverter: func(displayname, mxid, eventID string, ctx format.Context) string {
//		if len(eventID) > 0 {
//			return fmt.Sprintf(`[%s](https://matrix.to/#/%s/%s)`, displayname, mxid, eventID)
//		} else {
//			return fmt.Sprintf(`[%s](https://matrix.to/#/%s)`, displayname, mxid)
//		}
//	},
//	Newline:        "\n",
//	HorizontalLine: "\n---\n",
//}

func (view *RoomView) SetEditing(evt *database.Event) {
	//if evt == nil {
	//	view.editing = nil
	//	view.SetInputText(view.editMoveText)
	//	view.editMoveText = ""
	//} else {
	//	if view.editing == nil {
	//		view.editMoveText = view.GetInputText()
	//	}
	//	view.editing = evt
	//	// replying should never be non-nil when SetEditing, but do this just to be safe
	//	view.replying = nil
	//	msgContent := view.editing.Content.AsMessage()
	//	if len(view.editing.Gomuks.Edits) > 0 {
	//		// This feels kind of dangerous, but I think it works
	//		msgContent = view.editing.Gomuks.Edits[len(view.editing.Gomuks.Edits)-1].Content.AsMessage().NewContent
	//	}
	//	text := msgContent.Body
	//	if len(msgContent.FormattedBody) > 0 && (!view.config.Preferences.DisableMarkdown || !view.config.Preferences.DisableHTML) {
	//		if view.config.Preferences.DisableMarkdown {
	//			text = msgContent.FormattedBody
	//		} else {
	//			text = editHTMLParser.Parse(msgContent.FormattedBody, make(format.Context))
	//		}
	//	}
	//	if msgContent.MsgType == event.MsgEmote {
	//		text = "/me " + text
	//	}
	//	view.input.SetText(text)
	//}
	//view.status.SetText(view.GetStatus())
	//view.input.SetCursorOffset(-1)
}

type findFilter func(evt *database.Event) bool

func (view *RoomView) filterOwnOnly(evt *database.Event) bool {
	return evt.Sender == view.parent.matrix.UserID && evt.GetType() == event.EventMessage
}

//func (view *RoomView) filterMediaOnly(evt *database.Event) bool {
//	msgtype := event.MessageType(gjson.GetBytes(evt.GetContent(), "msgtype").Str)
//	switch msgtype {
//	case event.MsgFile, event.MsgImage, event.MsgAudio, event.MsgVideo:
//		return true
//	default:
//		return false
//	}
//}

func (view *RoomView) findMessage(current *database.Event, forward bool, allow findFilter) *messages.UIMessage {
	//currentFound := current == nil
	//msgs := view.MessageView().messages
	//for i := 0; i < len(msgs); i++ {
	//	index := i
	//	if !forward {
	//		index = len(msgs) - i - 1
	//	}
	//	evt := msgs[index]
	//	if evt.EventID == "" || string(evt.EventID) == evt.TxnID || evt.IsService {
	//		continue
	//	} else if currentFound {
	//		if allow == nil || allow(evt.Event) {
	//			return evt
	//		}
	//	} else if evt.EventID == current.ID {
	//		currentFound = true
	//	}
	//}
	return nil
}

func (view *RoomView) EditNext() {
	if view.editing == nil {
		return
	}
	foundMsg := view.findMessage(view.editing, true, view.filterOwnOnly)
	view.SetEditing(foundMsg.GetEvent())
}

func (view *RoomView) EditPrevious() {
	if view.replying != nil {
		return
	}
	foundMsg := view.findMessage(view.editing, false, view.filterOwnOnly)
	if foundMsg != nil {
		view.SetEditing(foundMsg.GetEvent())
	}
}

func (view *RoomView) SelectNext() {
	//msgView := view.MessageView()
	//if msgView.selected == 0 {
	//	return
	//}
	//var filter findFilter
	//if view.selectReason == SelectDownload || view.selectReason == SelectOpen {
	//	filter = view.filterMediaOnly
	//}
	//foundMsg := view.findMessage(msgView.selected.GetEvent(), true, filter)
	//if foundMsg != nil {
	//	msgView.SetSelected(foundMsg)
	//	// TODO scroll selected message into view
	//}
}

func (view *RoomView) SelectPrevious() {
	//msgView := view.MessageView()
	//var filter findFilter
	//if view.selectReason == SelectDownload || view.selectReason == SelectOpen {
	//	filter = view.filterMediaOnly
	//}
	//foundMsg := view.findMessage(msgView.selected.GetEvent(), false, filter)
	//if foundMsg != nil {
	//	msgView.SetSelected(foundMsg)
	//	// TODO scroll selected message into view
	//}
}

type completion struct {
	displayName string
	id          string
}

func (view *RoomView) AutocompleteUser(existingText string) (completions []completion) {
	textWithoutPrefix := strings.TrimPrefix(existingText, "@")
	for _, user := range view.Room.GetMembers() {
		if user.Displayname == textWithoutPrefix || string(user.UserID) == existingText {
			// Exact match, return that.
			return []completion{{user.Displayname, string(user.UserID)}}
		}

		if strings.HasPrefix(user.Displayname, textWithoutPrefix) || strings.HasPrefix(string(user.UserID), existingText) {
			completions = append(completions, completion{user.Displayname, string(user.UserID)})
		}
	}
	return
}

func (view *RoomView) AutocompleteRoom(existingText string) (completions []completion) {
	//for _, room := range view.parent.rooms {
	//	alias := string(room.Room.GetCanonicalAlias())
	//	if alias == existingText {
	//		// Exact match, return that.
	//		return []completion{{alias, string(room.Room.ID)}}
	//	}
	//	if strings.HasPrefix(alias, existingText) {
	//		completions = append(completions, completion{alias, string(room.Room.ID)})
	//		continue
	//	}
	//}
	return
}

func (view *RoomView) AutocompleteEmoji(word string) (completions []string) {
	//if word[0] != ':' {
	//	return
	//}
	//var valueCompletion1 string
	//var manyValues bool
	//for name, value := range emoji.CodeMap() {
	//	if name == word {
	//		return []string{value}
	//	} else if strings.HasPrefix(name, word) {
	//		completions = append(completions, name)
	//		if valueCompletion1 == "" {
	//			valueCompletion1 = value
	//		} else if valueCompletion1 != value {
	//			manyValues = true
	//		}
	//	}
	//}
	//if !manyValues && len(completions) > 0 {
	//	return []string{emoji.CodeMap()[completions[0]]}
	//}
	return
}

//func findWordToTabComplete(text string) string {
//	output := ""
//	runes := []rune(text)
//	for i := len(runes) - 1; i >= 0; i-- {
//		if unicode.IsSpace(runes[i]) {
//			break
//		}
//		output = string(runes[i]) + output
//	}
//	return output
//}

//var (
//	mentionMarkdown  = "[%[1]s](https://matrix.to/#/%[2]s)"
//	mentionHTML      = `<a href="https://matrix.to/#/%[2]s">%[1]s</a>`
//	mentionPlaintext = "%[1]s"
//)
//
//func (view *RoomView) defaultAutocomplete(word string, startIndex int) (strCompletions []string, strCompletion string) {
//	if len(word) == 0 {
//		return []string{}, ""
//	}
//
//	completions := view.AutocompleteUser(word)
//	completions = append(completions, view.AutocompleteRoom(word)...)
//
//	if len(completions) == 1 {
//		completion := completions[0]
//		template := mentionMarkdown
//		if view.config.Preferences.DisableMarkdown {
//			if view.config.Preferences.DisableHTML {
//				template = mentionPlaintext
//			} else {
//				template = mentionHTML
//			}
//		}
//		strCompletion = fmt.Sprintf(template, completion.displayName, completion.id)
//		if startIndex == 0 && completion.id[0] == '@' {
//			strCompletion = strCompletion + ":"
//		}
//	} else if len(completions) > 1 {
//		for _, completion := range completions {
//			strCompletions = append(strCompletions, completion.displayName)
//		}
//	}
//
//	//strCompletions = append(strCompletions, view.parent.cmdProcessor.AutocompleteCommand(word)...)
//	strCompletions = append(strCompletions, view.AutocompleteEmoji(word)...)
//
//	return
//}

func (view *RoomView) InputTabComplete(text string, cursorOffset int) {
	//if len(text) == 0 {
	//	return
	//}
	//
	//str := runewidth.Truncate(text, cursorOffset, "")
	//word := findWordToTabComplete(str)
	//startIndex := len(str) - len(word)
	//
	//var strCompletion string
	//
	//strCompletions, newText, ok := view.parent.cmdProcessor.Autocomplete(view, text, cursorOffset)
	//if !ok {
	//	strCompletions, strCompletion = view.defaultAutocomplete(word, startIndex)
	//}
	//
	//if len(strCompletions) > 0 {
	//	strCompletion = exstrings.LongestCommonPrefix(strCompletions)
	//	sort.Sort(sort.StringSlice(strCompletions))
	//}
	//if len(strCompletion) > 0 && len(strCompletions) < 2 {
	//	strCompletion += " "
	//	strCompletions = []string{}
	//}
	//
	//if len(strCompletion) > 0 && newText == text {
	//	newText = str[0:startIndex] + strCompletion + text[len(str):]
	//}
	//
	//view.input.SetTextAndMoveCursor(newText)
	//view.SetCompletions(strCompletions)
}

func (view *RoomView) InputSubmit(text string) {
	if len(text) == 0 {
		return
	} else if cmd, err := view.ParseCommand(text); err != nil {
		view.Room.ApplyPending(database.MakeFakeEvent(view.Room.ID,
			fmt.Sprintf("Failed to parse command: <code>%v</code>", html.EscapeString(err.Error()))))
		view.parent.parent.Render()
	} else if cmd != nil {
		go view.HandleCommand(cmd)
	} else {
		go view.SendMessage(event.MsgText, text)
	}
	view.editMoveText = ""
	view.SetInputText("")
}

func (view *RoomView) CopyToClipboard(text string, register string) {
	if register == "clipboard" || register == "primary" {
		err := clipboard.WriteAll(text, register)
		if err != nil {
			//view.AddServiceMessage(fmt.Sprintf("Clipboard unsupported: %v", err))
			//view.parent.parent.Render()
		}
	} else {
		//view.AddServiceMessage(fmt.Sprintf("Clipboard register %v unsupported", register))
		//view.parent.parent.Render()
	}
}

func (view *RoomView) Download(url id.ContentURI, file *attachment.EncryptedFile, filename string, openFile bool) {
	//path, err := view.parent.matrix.DownloadToDisk(url, file, filename)
	//if err != nil {
	//	view.AddServiceMessage(fmt.Sprintf("Failed to download media: %v", err))
	//	view.parent.parent.Render()
	//	return
	//}
	//view.AddServiceMessage(fmt.Sprintf("File downloaded to %s", path))
	//view.parent.parent.Render()
	//if openFile {
	//	debug.Print("Opening file", path)
	//	open.Open(path)
	//}
}

func (view *RoomView) Redact(eventID id.EventID, reason string) {
	defer debug.Recover()
	_, err := view.parent.matrix.RedactEvent(context.TODO(), &jsoncmd.RedactEventParams{
		RoomID:  view.Room.ID,
		EventID: eventID,
		Reason:  reason,
	})
	if err != nil {
		//view.AddServiceMessage(fmt.Sprintf("Failed to redact message: %v", err))
		//view.parent.parent.Render()
	}
}

func (view *RoomView) SendReaction(eventID id.EventID, reaction string) {
	defer debug.Recover()
	reaction = variationselector.Add(strings.TrimSpace(reaction))
	debug.Print("Reacting to", eventID, "in", view.Room.ID, "with", reaction)
	contentJSON, _ := json.Marshal(&event.ReactionEventContent{RelatesTo: event.RelatesTo{
		Type:    event.RelAnnotation,
		EventID: eventID,
		Key:     reaction,
	}})
	_, err := view.parent.matrix.SendEvent(context.TODO(), &jsoncmd.SendEventParams{
		RoomID:    view.Room.ID,
		EventType: event.EventReaction,
		Content:   contentJSON,
	})
	if err != nil {
		view.AddServiceMessage("Failed to send reaction: %v", err)
		view.parent.parent.Render()
	}
}

func (view *RoomView) SendMessage(msgtype event.MessageType, text string) {
	defer debug.Recover()
	err := view.parent.matrix.SendMessage(context.TODO(), &jsoncmd.SendMessageParams{
		RoomID:      view.Room.ID,
		BaseContent: nil,
		Extra:       nil,
		Text:        text,
		RelatesTo:   nil,
		Mentions:    nil,
		URLPreviews: nil,
	})
	if err != nil {
		debug.Print("Failed to send message:", err)
		view.AddServiceMessage("Failed to send message: %v", err)
	}
	debug.Print("Rendering after sending message")
	view.parent.parent.Render()
}

func (view *RoomView) SendMessageHTML(msgtype event.MessageType, text, html string) {
	//defer debug.Recover()
	//debug.Print("Sending message", msgtype, text, "to", view.Room.ID)
	//if !view.config.Preferences.DisableEmojis {
	//	text = emoji.Sprint(text)
	//}
	//rel := view.getRelationForNewEvent()
	//evt := view.parent.matrix.PrepareMarkdownMessage(view.Room.ID, msgtype, text, html, rel)
	//view.addLocalEcho(evt)
}

func (view *RoomView) MessageView() *MessageView {
	return view.content
}

func (view *RoomView) Update(meta *database.Room) {
	topicStr := strings.TrimSpace(strings.ReplaceAll(ptr.Val(meta.Topic), "\n", " "))
	if view.config.Preferences.HideRoomList {
		if len(topicStr) > 0 {
			topicStr = fmt.Sprintf("%s - %s", ptr.Val(meta.Name), topicStr)
		} else {
			topicStr = ptr.Val(meta.Name)
		}
		topicStr = strings.TrimSpace(topicStr)
	}
	view.topic.SetText(topicStr)
	if meta.EncryptionEvent != nil && meta.EncryptionEvent.Algorithm == id.AlgorithmMegolmV1 {
		view.input.SetPlaceholder("Send an encrypted message...")
	}
	if !view.userListLoaded && view.Room.FullMembersLoaded.Load() {
		view.UpdateUserList()
	}
	view.parent.parent.NeedsRender = true
}

func (view *RoomView) UpdateUserList() {
	view.userList.Update(view.Room.GetMembers(), view.Room.GetPowerLevels())
	view.userListLoaded = true
}

func (view *RoomView) AddServiceMessage(text string, args ...any) {
	if len(args) > 0 {
		text = fmt.Sprintf(text, args...)
	}
	view.Room.ApplyPending(database.MakeFakeEvent(view.Room.ID, html.EscapeString(text)))
}
