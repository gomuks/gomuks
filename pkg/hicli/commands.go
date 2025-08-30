// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"encoding/json"
	"fmt"

	"go.mau.fi/util/exerrors"

	"maunium.net/go/mautrix/event"
)

var BuiltInCommands = event.BotCommandsEventContent{
	Sigil:    "/",
	Commands: builtInCommandsList,
}

var builtInCommandsList = []*event.BotCommand{{
	Syntax:      "join",
	Description: event.MakeExtensibleText("Jump to the join room view by ID, alias or link"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Room identifier"),
	}},
}, {
	Syntax:      "leave",
	Aliases:     []string{"part"},
	Description: event.MakeExtensibleText("Leave the current room"),
}, {
	Syntax:      "invite {user_id} {reason}",
	Description: event.MakeExtensibleText("Invite a user to the current room"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeUserID,
		Description: event.MakeExtensibleText("User ID"),
	}, {
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Reason for invite"),
	}},
}, {
	Syntax:      "kick {user_id} {reason}",
	Description: event.MakeExtensibleText("Kick a user from the current room"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeUserID,
		Description: event.MakeExtensibleText("User ID"),
	}, {
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Reason for kick"),
	}},
}, {
	Syntax:      "ban {user_id} {reason}",
	Description: event.MakeExtensibleText("Ban a user from the current room"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeUserID,
		Description: event.MakeExtensibleText("User ID"),
	}, {
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Reason for ban"),
	}},
}, {
	Syntax:      "myroomnick {name}",
	Aliases:     []string{"roomnick {name}"},
	Description: event.MakeExtensibleText("Set your display name in the current room"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("New display name"),
	}},
}}

func init() {
	fmt.Println(string(exerrors.Must(json.Marshal(&BuiltInCommands))))
}
