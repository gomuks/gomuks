// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmdspec

import (
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/event/cmdschema"
	"maunium.net/go/mautrix/id"
)

const FakeGomuksSender id.UserID = "@gomuks"

const (
	Join           = "join"
	Leave          = "leave"
	Invite         = "invite"
	Kick           = "kick"
	Ban            = "ban"
	MyRoomNick     = "myroomnick"
	MyRoomAvatar   = "myroomavatar"
	GlobalNick     = "globalnick"
	GlobalAvatar   = "globalavatar"
	RoomName       = "roomname"
	RoomAvatar     = "roomavatar"
	Redact         = "redact"
	Raw            = "raw"
	UnencryptedRaw = "unencryptedraw"
	RawState       = "rawstate"
	DiscardSession = "discardsession"
	Devtools       = "devtools"
	Meow           = "meow"
	AddAlias       = "alias add"
	DelAlias       = "alias del"
)

var CommandDefinitions = []*cmdschema.EventContent{{
	Command:     Meow,
	Description: event.MakeExtensibleText("Meow"),
	Parameters: []*cmdschema.Parameter{{
		Key:         "meow",
		Schema:      cmdschema.PrimitiveTypeString.Schema(),
		Description: event.MakeExtensibleText("Meow"),
	}},
}, {
	Command:     Join,
	Description: event.MakeExtensibleText("Jump to the join room view by ID, alias or link"),
	Parameters: []*cmdschema.Parameter{{
		Key:         "room_reference",
		Schema:      cmdschema.PrimitiveTypeString.Schema(),
		Description: event.MakeExtensibleText("Room identifier"),
	}, {
		Key:         "reason",
		Schema:      cmdschema.PrimitiveTypeString.Schema(),
		Description: event.MakeExtensibleText("Reason for joining"),
		Optional:    true,
	}},
}, {
	Command:     Leave,
	Aliases:     []string{"part"},
	Description: event.MakeExtensibleText("Leave the current room"),
}, {
	Command:     Invite,
	Description: event.MakeExtensibleText("Invite a user to the current room"),
	Parameters: []*cmdschema.Parameter{{
		Key:         "user_id",
		Schema:      cmdschema.PrimitiveTypeUserID.Schema(),
		Description: event.MakeExtensibleText("User ID"),
	}, {
		Key:         "reason",
		Schema:      cmdschema.PrimitiveTypeString.Schema(),
		Description: event.MakeExtensibleText("Reason for invite"),
		Optional:    true,
	}},
}, {
	Command:     Kick,
	Description: event.MakeExtensibleText("Kick a user from the current room"),
	Parameters: []*cmdschema.Parameter{{
		Key:         "user_id",
		Schema:      cmdschema.PrimitiveTypeUserID.Schema(),
		Description: event.MakeExtensibleText("User ID"),
	}, {
		Key:         "reason",
		Schema:      cmdschema.PrimitiveTypeString.Schema(),
		Description: event.MakeExtensibleText("Reason for kick"),
		Optional:    true,
	}},
}, {
	Command:     Ban,
	Description: event.MakeExtensibleText("Ban a user from the current room"),
	Parameters: []*cmdschema.Parameter{{
		Key:         "user_id",
		Schema:      cmdschema.PrimitiveTypeUserID.Schema(),
		Description: event.MakeExtensibleText("User ID"),
	}, {
		Key:         "reason",
		Schema:      cmdschema.PrimitiveTypeString.Schema(),
		Description: event.MakeExtensibleText("Reason for ban"),
		Optional:    true,
	}},
}, {
	Command:     MyRoomNick,
	Aliases:     []string{"roomnick"},
	Description: event.MakeExtensibleText("Set your display name in the current room"),
	Parameters: []*cmdschema.Parameter{{
		Key:         "name",
		Schema:      cmdschema.PrimitiveTypeString.Schema(),
		Description: event.MakeExtensibleText("New display name"),
	}},
}, {
	Command:     MyRoomAvatar,
	Description: event.MakeExtensibleText("Set your avatar in the current room"),
}, {
	Command:     GlobalNick,
	Aliases:     []string{"globalname"},
	Description: event.MakeExtensibleText("Set your global display name"),
	Parameters: []*cmdschema.Parameter{{
		Key:         "name",
		Schema:      cmdschema.PrimitiveTypeString.Schema(),
		Description: event.MakeExtensibleText("New display name"),
	}},
}, {
	Command:     GlobalAvatar,
	Description: event.MakeExtensibleText("Set your global avatar"),
}, {
	Command:     RoomName,
	Description: event.MakeExtensibleText("Set the current room name"),
	Parameters: []*cmdschema.Parameter{{
		Key:         "name",
		Schema:      cmdschema.PrimitiveTypeString.Schema(),
		Description: event.MakeExtensibleText("New display name"),
	}},
}, {
	Command:     RoomAvatar,
	Description: event.MakeExtensibleText("Set the current room avatar"),
}, {
	Command:     Redact,
	Description: event.MakeExtensibleText("Redact an event"),
	Parameters: []*cmdschema.Parameter{{
		Key:         "event_id",
		Schema:      cmdschema.PrimitiveTypeEventID.Schema(),
		Description: event.MakeExtensibleText("Event ID or link"),
	}, {
		Key:         "reason",
		Schema:      cmdschema.PrimitiveTypeString.Schema(),
		Description: event.MakeExtensibleText("Reason for redaction"),
		Optional:    true,
	}},
}, {
	Command:     Raw,
	Description: event.MakeExtensibleText("Send a raw timeline event to the current room"),
	Parameters: []*cmdschema.Parameter{{
		Key:         "event_type",
		Schema:      cmdschema.PrimitiveTypeString.Schema(),
		Description: event.MakeExtensibleText("Event type"),
	}, {
		Key:          "json",
		Schema:       cmdschema.PrimitiveTypeString.Schema(),
		Description:  event.MakeExtensibleText("Event content as JSON"),
		DefaultValue: "{}",
	}},
}, {
	Command:     UnencryptedRaw,
	Description: event.MakeExtensibleText("Send an unencrypted raw timeline event to the current room"),
	Parameters: []*cmdschema.Parameter{{
		Key:         "event_type",
		Schema:      cmdschema.PrimitiveTypeString.Schema(),
		Description: event.MakeExtensibleText("Event type"),
	}, {
		Key:          "json",
		Schema:       cmdschema.PrimitiveTypeString.Schema(),
		Description:  event.MakeExtensibleText("Event content as JSON"),
		DefaultValue: "{}",
	}},
}, {
	Command:     RawState,
	Description: event.MakeExtensibleText("Send a raw state event to the current room"),
	Parameters: []*cmdschema.Parameter{{
		Key:         "event_type",
		Schema:      cmdschema.PrimitiveTypeString.Schema(),
		Description: event.MakeExtensibleText("Event type"),
	}, {
		Key:         "state_key",
		Schema:      cmdschema.PrimitiveTypeString.Schema(),
		Description: event.MakeExtensibleText("State key"),
	}, {
		Key:          "json",
		Schema:       cmdschema.PrimitiveTypeString.Schema(),
		Description:  event.MakeExtensibleText("Event content as JSON"),
		DefaultValue: "{}",
	}},
}, {
	Command:     DiscardSession,
	Description: event.MakeExtensibleText("Discard the outbound Megolm session in the current room"),
}, {
	Command:     Devtools,
	Description: event.MakeExtensibleText("Open the room state explorer"),
}, {
	Command:     AddAlias,
	Description: event.MakeExtensibleText("Add a room alias to the current room. Does not update the canonical alias event."),
	Parameters: []*cmdschema.Parameter{{
		Key:         "name",
		Schema:      cmdschema.PrimitiveTypeString.Schema(),
		Description: event.MakeExtensibleText("Room alias name to add (without the # and domain)"),
	}},
	Aliases: []string{"alias create"},
}, {
	Command:     DelAlias,
	Description: event.MakeExtensibleText("Remove a room alias from the current room. Does not update the canonical alias event."),
	Parameters: []*cmdschema.Parameter{{
		Key:         "name",
		Schema:      cmdschema.PrimitiveTypeString.Schema(),
		Description: event.MakeExtensibleText("Room alias name to remove (without the # and domain)"),
	}},
	Aliases: []string{"alias remove", "alias rm", "alias delete"},
}}
