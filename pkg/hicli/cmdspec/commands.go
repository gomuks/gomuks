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
	Join           = "join {room_reference} {reason}"
	Leave          = "leave"
	Invite         = "invite {user_id} {reason}"
	Kick           = "kick {user_id} {reason}"
	Ban            = "ban {user_id} {reason}"
	MyRoomNick     = "myroomnick {name}"
	MyRoomAvatar   = "myroomavatar"
	GlobalNick     = "globalnick {name}"
	GlobalAvatar   = "globalavatar"
	RoomName       = "roomname"
	RoomAvatar     = "roomavatar"
	Redact         = "redact {event_id} {reason}"
	Raw            = "raw {event_type} {json}"
	UnencryptedRaw = "unencryptedraw {event_type} {json}"
	RawState       = "rawstate {event_type} {state_key} {json}"
	DiscardSession = "discardsession"
	Meow           = "meow {meow}"
	AddAlias       = "alias add {name}"
	DelAlias       = "alias del {name}"
)

var CommandDefinitions = []*event.BotCommand{{
	Syntax:      Meow,
	Description: event.MakeExtensibleText("Meow"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Meow"),
	}},
}, {
	Syntax:      Join,
	Description: event.MakeExtensibleText("Jump to the join room view by ID, alias or link"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Room identifier"),
	}, {
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Reason for joining"),
	}},
}, {
	Syntax:      Leave,
	Aliases:     []string{"part"},
	Description: event.MakeExtensibleText("Leave the current room"),
}, {
	Syntax:      Invite,
	Description: event.MakeExtensibleText("Invite a user to the current room"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeUserID,
		Description: event.MakeExtensibleText("User ID"),
	}, {
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Reason for invite"),
	}},
}, {
	Syntax:      Kick,
	Description: event.MakeExtensibleText("Kick a user from the current room"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeUserID,
		Description: event.MakeExtensibleText("User ID"),
	}, {
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Reason for kick"),
	}},
}, {
	Syntax:      Ban,
	Description: event.MakeExtensibleText("Ban a user from the current room"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeUserID,
		Description: event.MakeExtensibleText("User ID"),
	}, {
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Reason for ban"),
	}},
}, {
	Syntax:      MyRoomNick,
	Aliases:     []string{"roomnick {name}"},
	Description: event.MakeExtensibleText("Set your display name in the current room"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("New display name"),
	}},
}, {
	Syntax:      MyRoomAvatar,
	Description: event.MakeExtensibleText("Set your avatar in the current room"),
}, {
	Syntax:      GlobalNick,
	Aliases:     []string{"globalname {name}"},
	Description: event.MakeExtensibleText("Set your global display name"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("New display name"),
	}},
}, {
	Syntax:      GlobalAvatar,
	Description: event.MakeExtensibleText("Set your global avatar"),
}, {
	Syntax:      RoomName,
	Description: event.MakeExtensibleText("Set the current room name"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("New display name"),
	}},
}, {
	Syntax:      RoomAvatar,
	Description: event.MakeExtensibleText("Set the current room avatar"),
}, {
	Syntax:      Redact,
	Description: event.MakeExtensibleText("Redact an event"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeEventID,
		Description: event.MakeExtensibleText("Event ID or link"),
	}, {
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Reason for redaction"),
	}},
}, {
	Syntax:      Raw,
	Description: event.MakeExtensibleText("Send a raw timeline event to the current room"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Event type"),
	}, {
		Type:         event.BotArgumentTypeString,
		Description:  event.MakeExtensibleText("Event content as JSON"),
		DefaultValue: "{}",
	}},
}, {
	Syntax:      UnencryptedRaw,
	Description: event.MakeExtensibleText("Send an unencrypted raw timeline event to the current room"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Event type"),
	}, {
		Type:         event.BotArgumentTypeString,
		Description:  event.MakeExtensibleText("Event content as JSON"),
		DefaultValue: "{}",
	}},
}, {
	Syntax:      RawState,
	Description: event.MakeExtensibleText("Send a raw state event to the current room"),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Event type"),
	}, {
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("State key"),
	}, {
		Type:         event.BotArgumentTypeString,
		Description:  event.MakeExtensibleText("Event content as JSON"),
		DefaultValue: "{}",
	}},
}, {
	Syntax:      DiscardSession,
	Description: event.MakeExtensibleText("Discard the outbound Megolm session in the current room"),
}, {
	Syntax:      AddAlias,
	Description: event.MakeExtensibleText("Add a room alias to the current room. Does not update the canonical alias event."),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Room alias name to add (without the # and domain)"),
	}},
	Aliases: []string{"alias create {name}"},
}, {
	Syntax:      DelAlias,
	Description: event.MakeExtensibleText("Remove a room alias from the current room. Does not update the canonical alias event."),
	Arguments: []*event.BotCommandArgument{{
		Type:        event.BotArgumentTypeString,
		Description: event.MakeExtensibleText("Room alias name to remove (without the # and domain)"),
	}},
	Aliases: []string{"alias remove {name}", "alias rm {name}", "alias delete {name}"},
}}
