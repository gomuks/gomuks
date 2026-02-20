// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package jsoncmd

import (
	"encoding/json"
	"fmt"

	"go.mau.fi/util/jsontime"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
)

func EventTypeName(evt any) Name {
	switch evt.(type) {
	case *SyncComplete:
		return EventSyncComplete
	case *SyncStatus:
		return EventSyncStatus
	case *EventsDecrypted:
		return EventEventsDecrypted
	case *Typing:
		return EventTyping
	case *SendComplete:
		return EventSendComplete
	case *ClientState:
		return EventClientState
	case *InitComplete:
		return EventInitComplete
	default:
		panic(fmt.Errorf("unknown event type %T", evt))
	}
}

type SyncRoom struct {
	// Metadata about the room. The frontend should replace the entire cached object rather than merging.
	Meta *database.Room `json:"meta"`
	// New timeline events to append to the existing list
	// (except if `reset` is set, in which case this replaces the existing list).
	Timeline []database.TimelineRowTuple `json:"timeline"`
	// If true, the frontend should discard the existing timeline cache for this room.
	Reset bool `json:"reset"`
	// New state events. This nested map should be deeply merged into the existing state map.
	State map[event.Type]map[string]database.EventRowID `json:"state"`
	// New room account data events. Like global account data, only changes are listed,
	// but the entire content of changed events should be replaced.
	AccountData map[event.Type]*database.AccountData `json:"account_data"`
	// Events that the frontend needs to handle this sync. This may include old events, as well as
	// events the frontend already has. The `timeline` and `state` fields are the ones that decide
	// where the events are actually used.
	Events []*database.Event `json:"events"`
	// New read receipts. The frontend should only keep the latest receipt per user.
	Receipts map[id.EventID][]*database.Receipt `json:"receipts"`

	DismissNotifications bool               `json:"dismiss_notifications"`
	Notifications        []SyncNotification `json:"notifications"`
}

type SyncNotification struct {
	RowID     database.EventRowID `json:"event_rowid"`
	Sound     bool                `json:"sound"`
	Highlight bool                `json:"highlight"`
	Event     *database.Event     `json:"-"`
	Room      *database.Room      `json:"-"`
}

type SyncToDevice struct {
	Sender    id.UserID       `json:"sender"`
	Type      event.Type      `json:"type"`
	Content   json.RawMessage `json:"content"`
	Encrypted bool            `json:"encrypted"`
}

type SyncComplete struct {
	// The `since` token sent to the server in the /sync request.
	// This is only for debugging, the frontend doesn't need to care about it.
	Since *string `json:"since,omitempty"`
	// If true, the frontend should throw away all state it has before applying this sync.
	// This is used on the first payload after connecting if resuming wasn't used or didn't succeed.
	ClearState bool `json:"clear_state,omitempty"`
	// New global account data events. Only changed events are listed here, but the entire content
	// of each changed event should be replaced.
	AccountData map[event.Type]*database.AccountData `json:"account_data"`
	// List of rooms that the user is participating in that have new data available.
	Rooms map[id.RoomID]*SyncRoom `json:"rooms"`
	// List of rooms that the user has left. The frontend should delete all state associated with these rooms.
	LeftRooms []id.RoomID `json:"left_rooms"`
	// List of new rooms that the user has been invited to.
	InvitedRooms []*database.InvitedRoom `json:"invited_rooms"`
	// List of spaces and their edges. When an edge in a space changes, all edges in that space are resent,
	// so the frontend should replace the entire list for that space.
	SpaceEdges map[id.RoomID][]*database.SpaceEdge `json:"space_edges"`
	// List of room IDs that should be considered as top-level spaces.
	// The frontend should replace the entire list if this field is set.
	TopLevelSpaces []id.RoomID `json:"top_level_spaces"`

	// New to-device events. This is only used for widgets and only emitted
	// if opted in with the send_to_device command.
	ToDevice []*SyncToDevice `json:"to_device,omitempty"`
}

func (c *SyncComplete) Notifications(yield func(SyncNotification) bool) {
	for _, room := range c.Rooms {
		for _, notif := range room.Notifications {
			if !yield(notif) {
				return
			}
		}
	}
}

func (c *SyncComplete) IsEmpty() bool {
	return len(c.Rooms) == 0 && len(c.LeftRooms) == 0 && len(c.InvitedRooms) == 0 && len(c.AccountData) == 0 && len(c.ToDevice) == 0
}

type SyncStatusType string

const (
	SyncStatusOK       SyncStatusType = "ok"
	SyncStatusWaiting  SyncStatusType = "waiting"
	SyncStatusErroring SyncStatusType = "erroring"
	SyncStatusFailed   SyncStatusType = "permanently-failed"
)

type SyncStatus struct {
	Type       SyncStatusType     `json:"type"`
	Error      string             `json:"error,omitempty"`
	ErrorCount int                `json:"error_count"`
	LastSync   jsontime.UnixMilli `json:"last_sync,omitempty"`
}

type EventsDecrypted struct {
	RoomID            id.RoomID           `json:"room_id"`
	PreviewEventRowID database.EventRowID `json:"preview_event_rowid,omitempty"`
	Events            []*database.Event   `json:"events"`
}

type Typing struct {
	RoomID id.RoomID `json:"room_id"`
	event.TypingEventContent
}

type SendComplete struct {
	Event *database.Event `json:"event"`
	Error error           `json:"error"`
}

type ClientState struct {
	Initialized   bool        `json:"is_initialized"`
	IsLoggedIn    bool        `json:"is_logged_in"`
	IsVerified    bool        `json:"is_verified"`
	UserID        id.UserID   `json:"user_id,omitempty"`
	DeviceID      id.DeviceID `json:"device_id,omitempty"`
	HomeserverURL string      `json:"homeserver_url,omitempty"`
}

type ImageAuthToken string

type InitComplete struct{}

type RunData struct {
	RunID    string `json:"run_id"`
	ETag     string `json:"etag"`
	VAPIDKey string `json:"vapid_key"`
}
