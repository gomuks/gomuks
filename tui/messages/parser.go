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
	"strings"

	"github.com/gdamore/tcell/v2"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/rpc/client"
	"go.mau.fi/gomuks/pkg/rpc/store"
	"go.mau.fi/gomuks/tui/config"
	"go.mau.fi/gomuks/tui/debug"
	"go.mau.fi/gomuks/tui/messages/html"
	"go.mau.fi/gomuks/tui/messages/tstring"
	"go.mau.fi/gomuks/tui/widget"
)

func ParseEvent(matrix *client.GomuksClient, prefs *config.UserPreferences, room *store.RoomStore, evt *database.Event) *UIMessage {
	msg := directParseEvent(matrix, prefs, room, evt)
	if msg == nil {
		return nil
	}
	if replyTo := evt.GetReplyTo(); len(replyTo) > 0 {
		if replyToEvt := room.GetEventByID(replyTo); replyToEvt != nil {
			if replyToMsg, ok := replyToEvt.RenderMeta.(*UIMessage); ok {
				if replyToMsg != nil {
					msg.ReplyTo = replyToMsg.Clone()
					msg.ReplyTo.IsReplyBubble = true
				} else {
					// TODO something
				}
			} else if replyToMsg = directParseEvent(matrix, prefs, room, replyToEvt); replyToMsg != nil {
				msg.ReplyTo = replyToMsg
				msg.ReplyTo.IsReplyBubble = true
			} else {
				// TODO add unrenderable reply header
			}
		} else {
			// TODO request reply event from backend
			// TODO add unknown reply header
		}
	}
	return msg
}

func directParseEvent(matrix *client.GomuksClient, prefs *config.UserPreferences, room *store.RoomStore, evt *database.Event) *UIMessage {
	if evt.DecryptionError != "" {
		return NewExpandedTextMessage(evt, room, tstring.NewStyleTString(evt.DecryptionError, tcell.StyleDefault.Italic(true)))
	}
	switch evt.GetType() {
	case event.EventMessage, event.EventSticker:
		if evt.RelationType == event.RelReplace {
			return nil
		} else if evt.RedactedBy != "" {
			return NewRedactedMessage(evt, room)
		}
		return ParseMessage(matrix, prefs, room, evt)
	case event.StateTopic, event.StateRoomName, event.StateCanonicalAlias:
		return ParseStateEvent(room, evt)
	case event.StateMember:
		return ParseMembershipEvent(room, evt)
	default:
		return nil
	}
}

func findAltAliasDifference(newList, oldList []id.RoomAlias) (addedStr, removedStr tstring.TString) {
	var addedList, removedList []tstring.TString
OldLoop:
	for _, oldAlias := range oldList {
		for _, newAlias := range newList {
			if oldAlias == newAlias {
				continue OldLoop
			}
		}
		removedList = append(removedList, tstring.NewStyleTString(string(oldAlias), tcell.StyleDefault.Foreground(widget.GetHashColor(oldAlias)).Underline(true)))
	}
NewLoop:
	for _, newAlias := range newList {
		for _, oldAlias := range oldList {
			if newAlias == oldAlias {
				continue NewLoop
			}
		}
		addedList = append(addedList, tstring.NewStyleTString(string(newAlias), tcell.StyleDefault.Foreground(widget.GetHashColor(newAlias)).Underline(true)))
	}
	if len(addedList) == 1 {
		addedStr = tstring.NewColorTString("added alternative address ", tcell.ColorGreen).AppendTString(addedList[0])
	} else if len(addedList) != 0 {
		addedStr = tstring.Join(addedList[:len(addedList)-1], ", ").
			PrependColor("added alternative addresses ", tcell.ColorGreen).
			AppendColor(" and ", tcell.ColorGreen).
			AppendTString(addedList[len(addedList)-1])
	}
	if len(removedList) == 1 {
		removedStr = tstring.NewColorTString("removed alternative address ", tcell.ColorGreen).AppendTString(removedList[0])
	} else if len(removedList) != 0 {
		removedStr = tstring.Join(removedList[:len(removedList)-1], ", ").
			PrependColor("removed alternative addresses ", tcell.ColorGreen).
			AppendColor(" and ", tcell.ColorGreen).
			AppendTString(removedList[len(removedList)-1])
	}
	return
}

func ParseStateEvent(room *store.RoomStore, evt *database.Event) *UIMessage {
	mEvt := evt.AsMautrix()

	// TODO make this update
	displayname := room.GetDisplayname(evt.Sender)
	text := tstring.NewColorTString(displayname, widget.GetHashColor(evt.Sender)).Append(" ")
	switch content := mEvt.Content.Parsed.(type) {
	case *event.TopicEventContent:
		if len(content.Topic) == 0 {
			text = text.AppendColor("removed the topic.", tcell.ColorGreen)
		} else {
			text = text.AppendColor("changed the topic to ", tcell.ColorGreen).
				AppendStyle(content.Topic, tcell.StyleDefault.Underline(true)).
				AppendColor(".", tcell.ColorGreen)
		}
	case *event.RoomNameEventContent:
		if len(content.Name) == 0 {
			text = text.AppendColor("removed the room name.", tcell.ColorGreen)
		} else {
			text = text.AppendColor("changed the room name to ", tcell.ColorGreen).
				AppendStyle(content.Name, tcell.StyleDefault.Underline(true)).
				AppendColor(".", tcell.ColorGreen)
		}
	case *event.CanonicalAliasEventContent:
		prevContent := &event.CanonicalAliasEventContent{}
		if mEvt.Unsigned.PrevContent != nil {
			_ = mEvt.Unsigned.PrevContent.ParseRaw(mEvt.Type)
			prevContent = mEvt.Unsigned.PrevContent.AsCanonicalAlias()
		}
		debug.Printf("%+v -> %+v", prevContent, content)
		if len(content.Alias) == 0 && len(prevContent.Alias) != 0 {
			text = text.AppendColor("removed the main address of the room", tcell.ColorGreen)
		} else if content.Alias != prevContent.Alias {
			text = text.
				AppendColor("changed the main address of the room to ", tcell.ColorGreen).
				AppendStyle(string(content.Alias), tcell.StyleDefault.Underline(true))
		} else {
			added, removed := findAltAliasDifference(content.AltAliases, prevContent.AltAliases)
			if len(added) > 0 {
				if len(removed) > 0 {
					text = text.
						AppendTString(added).
						AppendColor(" and ", tcell.ColorGreen).
						AppendTString(removed)
				} else {
					text = text.AppendTString(added)
				}
			} else if len(removed) > 0 {
				text = text.AppendTString(removed)
			} else {
				text = text.AppendColor("changed nothing", tcell.ColorGreen)
			}
			text = text.AppendColor(" for this room", tcell.ColorGreen)
		}
	}
	return NewExpandedTextMessage(evt, room, text)
}

func ParseMessage(matrix *client.GomuksClient, prefs *config.UserPreferences, room *store.RoomStore, evt *database.Event) *UIMessage {
	content := evt.GetMautrixContent().AsMessage()
	switch content.MsgType {
	case event.MsgText, event.MsgNotice, event.MsgEmote:
		var htmlEntity html.Entity
		if content.Format == event.FormatHTML && len(content.FormattedBody) > 0 {
			// TODO make this update
			displayname := room.GetDisplayname(evt.Sender)
			htmlEntity = html.Parse(prefs, room, content, evt, displayname)
			if htmlEntity == nil {
				htmlEntity = html.NewTextEntity("Malformed message")
				htmlEntity.AdjustStyle(html.AdjustStyleTextColor(tcell.ColorRed), html.AdjustStyleReasonNormal)
			}
		} else if len(content.Body) > 0 {
			content.Body = strings.Replace(content.Body, "\t", "    ", -1)
			htmlEntity = html.TextToEntity(content.Body, evt.ID, prefs.EnableInlineURLs())
		} else {
			htmlEntity = html.NewTextEntity("Blank message")
			htmlEntity.AdjustStyle(html.AdjustStyleTextColor(tcell.ColorRed), html.AdjustStyleReasonNormal)
		}
		return NewHTMLMessage(room, evt, content, htmlEntity)
	case event.MsgImage, event.MsgVideo, event.MsgAudio, event.MsgFile:
		msg := NewFileMessage(room, matrix, evt, content)
		if prefs != nil && !prefs.DisableDownloads && !prefs.DisableImages {
			renderer := msg.Renderer.(*FileMessage)
			renderer.DownloadPreview()
		}
		return msg
	}
	return nil
}

func getMembershipChangeMessage(evt *database.Event, content *event.MemberEventContent, prevMembership event.Membership, senderDisplayname, displayname, prevDisplayname string) (sender string, text tstring.TString) {
	switch content.Membership {
	case "invite":
		sender = "---"
		text = tstring.NewColorTString(fmt.Sprintf("%s invited %s.", senderDisplayname, displayname), tcell.ColorGreen)
		text.Colorize(0, len(senderDisplayname), widget.GetHashColor(evt.Sender))
		text.Colorize(len(senderDisplayname)+len(" invited "), len(displayname), widget.GetHashColor(evt.StateKey))
	case "join":
		sender = "-->"
		if prevMembership == event.MembershipInvite {
			text = tstring.NewColorTString(fmt.Sprintf("%s accepted the invite.", displayname), tcell.ColorGreen)
		} else {
			text = tstring.NewColorTString(fmt.Sprintf("%s joined the room.", displayname), tcell.ColorGreen)
		}
		text.Colorize(0, len(displayname), widget.GetHashColor(evt.StateKey))
	case "leave":
		sender = "<--"
		if evt.Sender != id.UserID(*evt.StateKey) {
			if prevMembership == event.MembershipBan {
				text = tstring.NewColorTString(fmt.Sprintf("%s unbanned %s", senderDisplayname, displayname), tcell.ColorGreen)
				text.Colorize(len(senderDisplayname)+len(" unbanned "), len(displayname), widget.GetHashColor(evt.StateKey))
			} else {
				text = tstring.NewColorTString(fmt.Sprintf("%s kicked %s: %s", senderDisplayname, displayname, content.Reason), tcell.ColorRed)
				text.Colorize(len(senderDisplayname)+len(" kicked "), len(displayname), widget.GetHashColor(evt.StateKey))
			}
			text.Colorize(0, len(senderDisplayname), widget.GetHashColor(evt.Sender))
		} else {
			if displayname == *evt.StateKey {
				displayname = prevDisplayname
			}
			if prevMembership == event.MembershipInvite {
				text = tstring.NewColorTString(fmt.Sprintf("%s rejected the invite.", displayname), tcell.ColorRed)
			} else {
				text = tstring.NewColorTString(fmt.Sprintf("%s left the room.", displayname), tcell.ColorRed)
			}
			text.Colorize(0, len(displayname), widget.GetHashColor(evt.StateKey))
		}
	case "ban":
		text = tstring.NewColorTString(fmt.Sprintf("%s banned %s: %s", senderDisplayname, displayname, content.Reason), tcell.ColorRed)
		text.Colorize(len(senderDisplayname)+len(" banned "), len(displayname), widget.GetHashColor(evt.StateKey))
		text.Colorize(0, len(senderDisplayname), widget.GetHashColor(evt.Sender))
	}
	return
}

func getMembershipEventContent(room *store.RoomStore, evt *database.Event) (sender string, text tstring.TString) {
	member := room.GetMember(evt.Sender)
	senderDisplayname := string(evt.Sender)
	if member != nil {
		senderDisplayname = member.Displayname
	}

	mEvt := evt.AsMautrix()
	content := mEvt.Content.AsMember()
	displayname := content.Displayname
	if len(displayname) == 0 {
		displayname = *evt.StateKey
	}

	prevMembership := event.MembershipLeave
	prevDisplayname := *evt.StateKey
	if mEvt.Unsigned.PrevContent != nil {
		_ = mEvt.Unsigned.PrevContent.ParseRaw(mEvt.Type)
		prevContent := mEvt.Unsigned.PrevContent.AsMember()
		prevMembership = prevContent.Membership
		prevDisplayname = prevContent.Displayname
		if len(prevDisplayname) == 0 {
			prevDisplayname = *evt.StateKey
		}
	}

	if content.Membership != prevMembership {
		sender, text = getMembershipChangeMessage(evt, content, prevMembership, senderDisplayname, displayname, prevDisplayname)
	} else if displayname != prevDisplayname {
		sender = "---"
		color := widget.GetHashColor(evt.StateKey)
		text = tstring.NewBlankTString().
			AppendColor(prevDisplayname, color).
			AppendColor(" changed their display name to ", tcell.ColorGreen).
			AppendColor(displayname, color).
			AppendColor(".", tcell.ColorGreen)
	}
	return
}

func ParseMembershipEvent(room *store.RoomStore, evt *database.Event) *UIMessage {
	displayname, text := getMembershipEventContent(room, evt)
	if len(text) == 0 {
		return nil
	}

	ui := NewExpandedTextMessage(evt, room, text)
	ui.OverrideSenderName = displayname
	return ui
}
