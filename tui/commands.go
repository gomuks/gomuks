// gomuks - A terminal Matrix client written in Go.
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

package tui

import (
	"context"
	"fmt"

	"github.com/tidwall/gjson"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/event/cmdschema"

	"go.mau.fi/gomuks/pkg/hicli/cmdspec"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
	"go.mau.fi/gomuks/pkg/rpc/store"
	"go.mau.fi/gomuks/tui/debug"
)

const (
	CmdReply  = "reply"
	CmdReact  = "react"
	CmdRedact = "redact"
	CmdQuit   = "quit"
	CmdEdit   = "edit"
	CmdCopy   = "copy"
)

var LocalCommands = []*cmdschema.EventContent{{
	Command:     CmdReply,
	Description: event.MakeExtensibleText("Reply to an event"),
	Parameters: []*cmdschema.Parameter{{
		Key:         "text",
		Schema:      cmdschema.PrimitiveTypeString.Schema(),
		Description: event.MakeExtensibleText("The text to reply with"),
	}},
}, {
	Command:     CmdReact,
	Description: event.MakeExtensibleText("React to an event"),
	Parameters: []*cmdschema.Parameter{{
		Key:         "key",
		Schema:      cmdschema.PrimitiveTypeString.Schema(),
		Description: event.MakeExtensibleText("The emoji or other text to react with"),
	}},
}, {
	Command:     CmdRedact,
	Aliases:     []string{"delete"},
	Description: event.MakeExtensibleText("Redact an event"),
	Parameters: []*cmdschema.Parameter{{
		Key:         "reason",
		Schema:      cmdschema.PrimitiveTypeString.Schema(),
		Description: event.MakeExtensibleText("The reason for the redaction"),
		Optional:    true,
	}},
}, {
	Command:     CmdEdit,
	Description: event.MakeExtensibleText("Start editing an event"),
}, {
	Command:     CmdCopy,
	Description: event.MakeExtensibleText("Copy text from an event"),
	Parameters: []*cmdschema.Parameter{{
		Key:          "register",
		Schema:       cmdschema.Enum("clipboard", "primary"),
		Optional:     true,
		DefaultValue: "clipboard",
	}},
}, {
	Command:     CmdQuit,
	Description: event.MakeExtensibleText("Quit gomuks terminal"),
}}

func (view *RoomView) allCommands(yield func(command *store.WrappedCommand) bool) {
	for _, cmd := range LocalCommands {
		if !yield(&store.WrappedCommand{
			EventContent: cmd,
			Source:       cmdspec.FakeGomuksSender,
		}) {
			return
		}
	}
	for _, cmd := range cmdspec.CommandDefinitions {
		if !yield(&store.WrappedCommand{
			EventContent: cmd,
			Source:       cmdspec.FakeGomuksSender,
		}) {
			return
		}
	}
	for _, cmd := range view.Room.GetBotCommands() {
		if !yield(cmd) {
			return
		}
	}
}

var cmdSigils = []string{"/"}

func (view *RoomView) ParseCommand(input string) (*event.MessageEventContent, error) {
	var firstError error
	view.Room.GetPowerLevels()
	for cmd := range view.allCommands {
		if parsed, err := cmd.ParseInput(cmd.Source, cmdSigils, input); parsed != nil {
			if err == nil {
				return parsed, nil
			} else if firstError == nil {
				firstError = fmt.Errorf("failed to parse %s: %w", cmd.Command, err)
			}
		}
	}
	return nil, firstError
}

func (view *RoomView) HandleCommand(cmd *event.MessageEventContent) {
	if cmd.Mentions.Has(cmdspec.FakeGomuksSender) &&
		len(cmd.Mentions.UserIDs) == 1 &&
		view.handleInternalCommand(cmd.MSC4391BotCommand) {
		// Handled internally
		return
	}
	mentions := cmd.Mentions
	cmd.Mentions = nil
	err := view.parent.matrix.SendMessage(context.TODO(), &jsoncmd.SendMessageParams{
		RoomID:      view.Room.ID,
		BaseContent: cmd,
		Mentions:    mentions,
	})
	if err != nil {
		debug.Print("Failed to send message:", err)
	}
	view.parent.parent.Render()
}

func (view *RoomView) handleInternalCommand(cmd *event.MSC4391BotCommandInput) bool {
	switch cmd.Command {
	case CmdReply:
		view.StartSelecting(SelectReply, gjson.GetBytes(cmd.Arguments, "text").Str)
	case CmdReact:
		view.StartSelecting(SelectReact, gjson.GetBytes(cmd.Arguments, "key").Str)
	case CmdRedact:
		view.StartSelecting(SelectRedact, gjson.GetBytes(cmd.Arguments, "reason").Str)
	case CmdEdit:
		view.StartSelecting(SelectEdit, "")
	case CmdCopy:
		view.StartSelecting(SelectCopy, gjson.GetBytes(cmd.Arguments, "register").Str)
	case CmdQuit:
		view.parent.parent.Stop()
	default:
		return false
	}
	return true
}
