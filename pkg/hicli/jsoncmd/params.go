// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package jsoncmd

import (
	"encoding/json"

	"go.mau.fi/util/jsontime"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
)

type CommandName string

type CancelRequestParams struct {
	RequestID int64  `json:"request_id"`
	Reason    string `json:"reason,omitempty"`
}

type SendMessageParams struct {
	RoomID id.RoomID `json:"room_id"`
	// Non-text event content
	BaseContent *event.MessageEventContent `json:"base_content,omitempty"`
	// Non-text event content that isn't supported by the MessageEventContent struct
	Extra map[string]any `json:"extra,omitempty"`
	// The text to send. If set, this will be used to fill the message `body`, `formatted_body`,
	// `format` and `msgtype` fields. Media captions should be put here even when using
	// `base_content` for the rest of the media. Some special-cased commands are also parsed from
	// this field (but most commands use the MSC4332 fields in base_content).
	Text string `json:"text"`
	// Standard Matrix `m.relates_to` data (replies, threading, edits).
	RelatesTo *event.RelatesTo `json:"relates_to,omitempty"`
	// Standard Matrix `m.mentions` data.
	Mentions *event.Mentions `json:"mentions,omitempty"`
	// Beeper URL previews to attach to the message.
	URLPreviews []*event.BeeperLinkPreview `json:"url_previews,omitempty"`
}

type SendEventParams struct {
	RoomID            id.RoomID       `json:"room_id"`
	EventType         event.Type      `json:"type"`
	Content           json.RawMessage `json:"content"`
	DisableEncryption bool            `json:"disable_encryption,omitempty"`
	Synchronous       bool            `json:"synchronous,omitempty"`
}

type ResendEventParams struct {
	TransactionID string `json:"transaction_id"`
}

type ReportEventParams struct {
	RoomID  id.RoomID  `json:"room_id"`
	EventID id.EventID `json:"event_id"`
	Reason  string     `json:"reason,omitempty"`
}

type RedactEventParams struct {
	RoomID  id.RoomID  `json:"room_id"`
	EventID id.EventID `json:"event_id"`
	Reason  string     `json:"reason,omitempty"`
}

type SendStateEventParams struct {
	RoomID    id.RoomID       `json:"room_id"`
	EventType event.Type      `json:"type"`
	StateKey  string          `json:"state_key"`
	Content   json.RawMessage `json:"content"`
	DelayMS   int             `json:"delay_ms,omitempty"`
}

type UpdateDelayedEventParams struct {
	DelayID id.DelayID        `json:"delay_id"`
	Action  event.DelayAction `json:"action"`
}

type SetMembershipParams struct {
	Action string    `json:"action"`
	RoomID id.RoomID `json:"room_id"`
	UserID id.UserID `json:"user_id"`
	Reason string    `json:"reason,omitempty"`
	// If true, the ban event will set a flag to suggest that clients hide all the user's messages.
	MSC4293RedactEvents bool `json:"msc4293_redact_events,omitempty"`
}

type SetAccountDataParams struct {
	// If set, the request will set room account data rather than global.
	RoomID  id.RoomID       `json:"room_id,omitempty"`
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
}

type MarkReadParams struct {
	RoomID      id.RoomID         `json:"room_id"`
	EventID     id.EventID        `json:"event_id"`
	ReceiptType event.ReceiptType `json:"receipt_type"`
}

type SetTypingParams struct {
	RoomID  id.RoomID `json:"room_id"`
	Timeout int       `json:"timeout"`
}

type GetProfileParams struct {
	UserID id.UserID `json:"user_id"`
}

type SetProfileFieldParams struct {
	Field string `json:"field"`
	Value any    `json:"value"`
}

type GetEventParams struct {
	RoomID   id.RoomID  `json:"room_id"`
	EventID  id.EventID `json:"event_id"`
	Unredact bool       `json:"unredact"`
}

type GetEventContextParams struct {
	RoomID  id.RoomID  `json:"room_id"`
	EventID id.EventID `json:"event_id"`
	Limit   int        `json:"limit"`
}

type GetMentionsParams struct {
	// The maximum event timestamp to return. For the first query, this should be set to the current timestamp.
	MaxTimestamp jsontime.UnixMilli `json:"max_timestamp"`
	// The unread type to filter for. Usually you want [database.UnreadTypeHighlight].
	Type database.UnreadType `json:"type"`
	// Maximum number of events to return.
	Limit int `json:"limit"`
	// Optional room ID to filter mentions to a specific room.
	RoomID id.RoomID `json:"room_id,omitempty"`
}

type GetRelatedEventsParams struct {
	RoomID  id.RoomID  `json:"room_id"`
	EventID id.EventID `json:"event_id"`

	RelationType event.RelationType `json:"relation_type"`
}

type GetRoomStateParams struct {
	RoomID id.RoomID `json:"room_id"`
	// Force refetch the entire state from the homeserver.
	Refetch bool `json:"refetch,omitempty"`
	// Fetch membership events from homeserver. The client should always set this when opening a
	// room if `has_member_list` is false in the room metadata.
	FetchMembers bool `json:"fetch_members,omitempty"`
	// Whether to include the member list in the response. This can be used with `fetch_members` to
	// tell the backend to fetch the list in the background rather than waiting for it.
	IncludeMembers bool `json:"include_members,omitempty"`
}

type GetSpecificRoomStateParams struct {
	Keys []database.RoomStateGUID `json:"keys"`
}

type EnsureGroupSessionSharedParams struct {
	RoomID id.RoomID `json:"room_id"`
}

type SendToDeviceParams struct {
	*mautrix.ReqSendToDevice
	EventType event.Type `json:"event_type"`
	Encrypted bool       `json:"encrypted"`
}

type ResolveAliasParams struct {
	Alias id.RoomAlias `json:"alias"`
}

type LoginParams struct {
	HomeserverURL string `json:"homeserver_url"`
	Username      string `json:"username"`
	Password      string `json:"password"`
}

type LoginCustomParams struct {
	HomeserverURL string            `json:"homeserver_url"`
	Request       *mautrix.ReqLogin `json:"request"`
}

type VerifyParams struct {
	RecoveryKey string `json:"recovery_key"`
}

type DiscoverHomeserverParams struct {
	UserID id.UserID `json:"user_id"`
}

type GetLoginFlowsParams struct {
	HomeserverURL string `json:"homeserver_url"`
}

type PaginateParams struct {
	RoomID id.RoomID `json:"room_id"`
	// The oldest known timeline row ID. All returned messages will have a lower ID than this (hence max ID).
	// This should be omitted or set to zero when resetting.
	MaxTimelineID database.TimelineRowID `json:"max_timeline_id,omitempty"`
	// Maximum number of messages to return.
	Limit int `json:"limit"`
	// If true, the backend will throw away any locally cached timeline state and reload it from the server.
	Reset bool `json:"reset,omitempty"`
}

type PaginateManualParams struct {
	RoomID id.RoomID `json:"room_id"`
	// Root event ID for thread pagination. Omit for non-thread pagination.
	ThreadRoot id.EventID `json:"thread_root,omitempty"`
	// `next_batch` token from previous request or the `start`/`end` fields of `get_event_context`.
	// Can be empty for starting thread pagination.
	Since     string            `json:"since,omitempty"`
	Direction mautrix.Direction `json:"direction"`
	Limit     int               `json:"limit"`
}

type JoinRoomParams struct {
	RoomIDOrAlias string `json:"room_id_or_alias"`
	// Via servers to attempt to join through.
	// This is required when using a room ID to join a server that the homeserver isn't participating in.
	Via    []string `json:"via,omitempty"`
	Reason string   `json:"reason,omitempty"`
}

type GetRoomSummaryParams struct {
	RoomIDOrAlias string `json:"room_id_or_alias"`
	// Via servers to attempt to join through.
	// This is required when using a room ID to join a server that the homeserver isn't participating in.
	Via []string `json:"via,omitempty"`
}

type GetHierarchyParams struct {
	RoomID        id.RoomID `json:"room_id"`
	From          string    `json:"from,omitempty"`
	Limit         int       `json:"limit"`
	MaxDepth      *int      `json:"max_depth,omitempty"`
	SuggestedOnly bool      `json:"suggested_only,omitempty"`
}

type LeaveRoomParams struct {
	RoomID id.RoomID `json:"room_id"`
	Reason string    `json:"reason"`
}

type GetReceiptsParams struct {
	RoomID   id.RoomID    `json:"room_id"`
	EventIDs []id.EventID `json:"event_ids"`
}

type MuteRoomParams struct {
	RoomID id.RoomID `json:"room_id"`
	Muted  bool      `json:"muted"`
}

type PingParams struct {
	LastReceivedID int64 `json:"last_received_id"`
}

type CalculateRoomIDParams struct {
	Timestamp       int64           `json:"timestamp"`
	CreationContent json.RawMessage `json:"content"`
}

type UploadMediaParams struct {
	Path string `json:"path"`
	// The file name for the upload. If empty, the base name of Path will be used.
	Filename string `json:"filename,omitempty"`
	// Whether the upload should be encrypted.
	Encrypt bool `json:"encrypt,omitempty"`
	// Whether the upload is a voice message. If true, a waveform will be generated.
	VoiceMessage bool `json:"voice_message,omitempty"`

	// Mime type to re-encode media to. Options below only apply if this is set.
	EncodeTo string `json:"encode_to,omitempty"`
	// Absolute width and height to resize to.
	ResizeWidth  int `json:"resize_width,omitempty"`
	ResizeHeight int `json:"resize_height,omitempty"`
	// Percentage to resize by. Must be between 1 and 100.
	ResizePercent int `json:"resize_percent,omitempty"`
	// For jpeg and webp files, the quality to encode as. Defaults to 80.
	Quality int `json:"quality,omitempty"`
}

type ExportKeysParams struct {
	Passphrase string    `json:"passphrase"`
	RoomID     id.RoomID `json:"room_id,omitempty"`
}
