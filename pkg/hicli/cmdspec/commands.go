// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmdspec

import (
	"maunium.net/go/mautrix/event"
)

const (
	CmdJoin           = "join {room_reference}"
	CmdLeave          = "leave"
	CmdInvite         = "invite {user_id} {reason}"
	CmdKick           = "kick {user_id} {reason}"
	CmdBan            = "ban {user_id} {reason}"
	CmdMyRoomNick     = "myroomnick {name}"
	CmdRaw            = "raw {event_type} {json}"
	CmdRawState       = "rawstate {event_type} {state_key} {json}"
	CmdDiscardSession = "discardsession"
)

var BuiltInCommands = []*event.BotCommand{{
	Syntax:      CmdJoin,
	Description: event.MakeExtensibleText("Jump to the join room view by ID, alias or link"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Room identifier"),
	}},
}, {
	Syntax:      CmdLeave,
	Aliases:     []string{"part"},
	Description: event.MakeExtensibleText("Leave the current room"),
}, {
	Syntax:      CmdInvite,
	Description: event.MakeExtensibleText("Invite a user to the current room"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeUserID,
		Description: event.MakeExtensibleText("User ID"),
	}, {
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Reason for invite"),
	}},
}, {
	Syntax:      CmdKick,
	Description: event.MakeExtensibleText("Kick a user from the current room"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeUserID,
		Description: event.MakeExtensibleText("User ID"),
	}, {
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Reason for kick"),
	}},
}, {
	Syntax:      CmdBan,
	Description: event.MakeExtensibleText("Ban a user from the current room"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeUserID,
		Description: event.MakeExtensibleText("User ID"),
	}, {
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Reason for ban"),
	}},
}, {
	Syntax:      CmdMyRoomNick,
	Aliases:     []string{"roomnick {name}"},
	Description: event.MakeExtensibleText("Set your display name in the current room"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("New display name"),
	}},
}, {
	Syntax:      CmdRaw,
	Description: event.MakeExtensibleText("Send a raw timeline event to the current room"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Event type"),
	}, {
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Event content as JSON"),
	}},
}, {
	Syntax:      CmdRawState,
	Description: event.MakeExtensibleText("Send a raw state event to the current room"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Event type"),
	}, {
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("State key"),
	}, {
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Event content as JSON"),
	}},
}, {
	Syntax:      CmdDiscardSession,
	Description: event.MakeExtensibleText("Discard the outbound Megolm session in the current room"),
}}
