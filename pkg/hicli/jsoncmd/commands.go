// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package jsoncmd

import (
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
)

type Container[T any] struct {
	Command   Name  `json:"command"`
	RequestID int64 `json:"request_id"`
	Data      T     `json:"data"`
}

type Name string

func (n Name) String() string {
	return string(n)
}

// All command names (both requests and events).
const (
	ReqGetState                 Name = "get_state"
	ReqCancel                   Name = "cancel"
	ReqSendMessage              Name = "send_message"
	ReqSendEvent                Name = "send_event"
	ReqResendEvent              Name = "resend_event"
	ReqReportEvent              Name = "report_event"
	ReqRedactEvent              Name = "redact_event"
	ReqSetState                 Name = "set_state"
	ReqUpdateDelayedEvent       Name = "update_delayed_event"
	ReqSetMembership            Name = "set_membership"
	ReqSetAccountData           Name = "set_account_data"
	ReqMarkRead                 Name = "mark_read"
	ReqSetTyping                Name = "set_typing"
	ReqGetProfile               Name = "get_profile"
	ReqSetProfileField          Name = "set_profile_field"
	ReqGetMutualRooms           Name = "get_mutual_rooms"
	ReqTrackUserDevices         Name = "track_user_devices"
	ReqGetProfileEncryptionInfo Name = "get_profile_encryption_info"
	ReqGetEvent                 Name = "get_event"
	ReqGetEventContext          Name = "get_event_context"
	ReqPaginateManual           Name = "paginate_manual"
	ReqGetMentions              Name = "get_mentions"
	ReqGetRelatedEvents         Name = "get_related_events"
	ReqGetRoomState             Name = "get_room_state"
	ReqGetSpecificRoomState     Name = "get_specific_room_state"
	ReqGetReceipts              Name = "get_receipts"
	ReqPaginate                 Name = "paginate"
	ReqGetRoomSummary           Name = "get_room_summary"
	ReqGetSpaceHierarchy        Name = "get_space_hierarchy"
	ReqJoinRoom                 Name = "join_room"
	ReqKnockRoom                Name = "knock_room"
	ReqLeaveRoom                Name = "leave_room"
	ReqCreateRoom               Name = "create_room"
	ReqMuteRoom                 Name = "mute_room"
	ReqEnsureGroupSessionShared Name = "ensure_group_session_shared"
	ReqSendToDevice             Name = "send_to_device"
	ReqResolveAlias             Name = "resolve_alias"
	ReqRequestOpenIDToken       Name = "request_openid_token"
	ReqLogout                   Name = "logout"
	ReqLogin                    Name = "login"
	ReqLoginCustom              Name = "login_custom"
	ReqVerify                   Name = "verify"
	ReqDiscoverHomeserver       Name = "discover_homeserver"
	ReqGetLoginFlows            Name = "get_login_flows"
	ReqRegisterPush             Name = "register_push"
	ReqListenToDevice           Name = "listen_to_device"
	ReqGetTurnServers           Name = "get_turn_servers"
	ReqGetMediaConfig           Name = "get_media_config"
	ReqCalculateRoomID          Name = "calculate_room_id"

	ReqGetAccountInfo Name = "get_account_info"
	ReqUploadMedia    Name = "upload_media"
	ReqExportKeys     Name = "export_keys"

	RespError   Name = "error"
	RespSuccess Name = "response"

	ReqPing  Name = "ping"
	RespPong Name = "pong"

	EventSyncComplete    Name = "sync_complete"
	EventSyncStatus      Name = "sync_status"
	EventEventsDecrypted Name = "events_decrypted"
	EventTyping          Name = "typing"
	EventSendComplete    Name = "send_complete"
	EventClientState     Name = "client_state"
	EventImageAuthToken  Name = "image_auth_token"
	EventInitComplete    Name = "init_complete"
	EventRunID           Name = "run_id"
)

// Frontend -> backend request specs
var (
	// GetState returns the current client state (login/verification/session info).
	// Note that state is also emitted as `client_state` events, so you usually don't need to request it manually.
	GetState = &CommandSpecWithoutRequest[*ClientState]{Name: ReqGetState}
	// Cancel an in-flight request. Returns true if the given request ID was found, false otherwise.
	Cancel = &CommandSpec[*CancelRequestParams, bool]{Name: ReqCancel}
	// SendMessage sends a Matrix message into a room. This is a higher-level helper around sending
	// `m.room.message` (and related) content. This will always perform an asynchronous send, which
	// means the returned event won't have an ID yet. Listen for the `send_complete` event to get
	// the final result.
	SendMessage = &CommandSpec[*SendMessageParams, *database.Event]{Name: ReqSendMessage}
	// SendEvent sends an arbitrary event into a room. This should be used for non-message events like reactions.
	// Note that state events must use `set_state` instead.
	SendEvent = &CommandSpec[*SendEventParams, *database.Event]{Name: ReqSendEvent}
	// ResendEvent retries sending a previously failed outgoing event.
	ResendEvent = &CommandSpec[*ResendEventParams, *database.Event]{Name: ReqResendEvent}
	// ReportEvent reports an event to the homeserver.
	ReportEvent = &CommandSpecWithoutResponse[*ReportEventParams]{Name: ReqReportEvent}
	// RedactEvent redacts an event in a room.
	RedactEvent = &CommandSpec[*RedactEventParams, *mautrix.RespSendEvent]{Name: ReqRedactEvent}
	// SetState sends a state event to a room.
	SetState = &CommandSpec[*SendStateEventParams, id.EventID]{Name: ReqSetState}
	// UpdateDelayedEvent updates or cancels a previously scheduled delayed event as per MSC4140.
	UpdateDelayedEvent = &CommandSpec[*UpdateDelayedEventParams, *mautrix.RespUpdateDelayedEvent]{Name: ReqUpdateDelayedEvent}
	// SetMembership is used for membership actions like inviting, kicking, banning or unbanning a user.
	// This should not be used for the user's own membership. Use `join_room`, `leave_room` or `knock_room` instead.
	SetMembership = &CommandSpecWithoutResponse[*SetMembershipParams]{Name: ReqSetMembership}
	// SetAccountData sets global or per-room account data.
	SetAccountData = &CommandSpecWithoutResponse[*SetAccountDataParams]{Name: ReqSetAccountData}
	// MarkRead sends a read receipt to a room.
	MarkRead = &CommandSpecWithoutResponse[*MarkReadParams]{Name: ReqMarkRead}
	// SetTyping starts or stops sending typing notifications in a room.
	SetTyping = &CommandSpecWithoutResponse[*SetTypingParams]{Name: ReqSetTyping}
	// GetProfile returns a Matrix user profile from the homeserver.
	GetProfile = &CommandSpec[*GetProfileParams, *mautrix.RespUserProfile]{Name: ReqGetProfile}
	// SetProfileField sets a field in the current user's Matrix profile.
	SetProfileField = &CommandSpecWithoutResponse[*SetProfileFieldParams]{Name: ReqSetProfileField}
	// GetMutualRooms returns the list of rooms shared between the current user and another user
	// from the homeserver.
	GetMutualRooms = &CommandSpec[*GetProfileParams, []id.RoomID]{Name: ReqGetMutualRooms}
	// TrackUserDevices start tracking a user’s e2ee device list if it's not already tracked, then returns
	// encryption info (same result as `get_profile_encryption_info`).
	TrackUserDevices = &CommandSpec[*GetProfileParams, *ProfileEncryptionInfo]{Name: ReqTrackUserDevices}
	// GetProfileEncryptionInfo returns the device list and trust state information for a user.
	GetProfileEncryptionInfo = &CommandSpec[*GetProfileParams, *ProfileEncryptionInfo]{Name: ReqGetProfileEncryptionInfo}
	// GetEvent returns a single event in a room. This uses the database if possible,
	// but will fetch from the homeserver if the event isn't found locally.
	GetEvent = &CommandSpec[*GetEventParams, *database.Event]{Name: ReqGetEvent}
	// GetEventContext returns context around an event (before/after timeline slices) from the
	// homeserver. This is used for jumping to a specific point on the timeline. Note that there is
	// currently no safe way to merge back into the main timeline, so jumping has to be implemented
	// as a separate view.
	GetEventContext = &CommandSpec[*GetEventContextParams, *EventContextResponse]{Name: ReqGetEventContext}
	// PaginateManual returns a page of messages from the homeserver using a pagination token.
	// This is used to paginate after jumping to a specific event using `get_event_context` and
	// for normal pagination in the thread view.
	PaginateManual = &CommandSpec[*PaginateManualParams, *ManualPaginationResponse]{Name: ReqPaginateManual}
	// GetMentions returns recent events that mention the current user. This will not call the homeserver.
	// The result is sorted by timestamp in descending order. Sorting by timestamp means the sender could
	// have faked it, but there's no other cross-room event ordering in Matrix.
	GetMentions = &CommandSpec[*GetMentionsParams, []*database.Event]{Name: ReqGetMentions}
	// GetRelatedEvents returns events related to a given event from the database (e.g. reactions,
	// edits, replies depending on relation type). This will not call the homeserver.
	GetRelatedEvents = &CommandSpec[*GetRelatedEventsParams, []*database.Event]{Name: ReqGetRelatedEvents}
	// GetRoomState returns full room state, optionally after fetching it from the homeserver.
	GetRoomState = &CommandSpec[*GetRoomStateParams, []*database.Event]{Name: ReqGetRoomState}
	// GetSpecificRoomState returns the requested individual state events.
	// The events are only fetched from the database, this will not call the homeserver.
	GetSpecificRoomState = &CommandSpec[*GetSpecificRoomStateParams, []*database.Event]{Name: ReqGetSpecificRoomState}
	// GetReceipts returns read receipts for a set of event IDs. This will not call the homeserver.
	GetReceipts = &CommandSpec[*GetReceiptsParams, map[id.EventID][]*database.Receipt]{Name: ReqGetReceipts}
	// Paginate returns older messages in the timeline. This will return locally cached timelines
	// if available and fetch more from the homeserver if needed.
	Paginate = &CommandSpec[*PaginateParams, *PaginationResponse]{Name: ReqPaginate}
	// GetRoomSummary returns the basic metadata of a room from the homeserver, such as name,
	// topic, avatar and member count. This should be used for previewing rooms before joining.
	// For joined rooms, metadata is automatically pushed in the sync payloads.
	GetRoomSummary = &CommandSpec[*GetRoomSummaryParams, *mautrix.RespRoomSummary]{Name: ReqGetRoomSummary}
	// GetSpaceHierarchy returns a space hierarchy, which may include rooms the user isn't in yet.
	// This should only be used for rendering the space index page. For the room list, space edge
	// information is automatically pushed in syncs.
	GetSpaceHierarchy = &CommandSpec[*GetHierarchyParams, *mautrix.RespHierarchy]{Name: ReqGetSpaceHierarchy}
	// JoinRoom joins the given room ID or alias.
	JoinRoom = &CommandSpec[*JoinRoomParams, *mautrix.RespJoinRoom]{Name: ReqJoinRoom}
	// KnockRoom knocks on the given room ID or alias.
	KnockRoom = &CommandSpec[*JoinRoomParams, *mautrix.RespKnockRoom]{Name: ReqKnockRoom}
	// LeaveRoom leaves or rejects the invite to the given room.
	LeaveRoom = &CommandSpec[*LeaveRoomParams, *mautrix.RespLeaveRoom]{Name: ReqLeaveRoom}
	// CreateRoom creates a new room.
	CreateRoom = &CommandSpec[*mautrix.ReqCreateRoom, *mautrix.RespCreateRoom]{Name: ReqCreateRoom}
	// MuteRoom mutes or unmutes a room by manipulating push rules. It returns the previous mute state.
	MuteRoom = &CommandSpec[*MuteRoomParams, bool]{Name: ReqMuteRoom}
	// EnsureGroupSessionShared ensures that the Megolm session for a room has been shared to all
	// recipient devices. Calling this is not required, but it should be called when the user first
	// starts typing to make sending faster.
	EnsureGroupSessionShared = &CommandSpecWithoutResponse[*EnsureGroupSessionSharedParams]{Name: ReqEnsureGroupSessionShared}
	// SendToDevice sends an arbitrary to-device event. Meant for widgets, not needed otherwise.
	SendToDevice = &CommandSpec[*SendToDeviceParams, *mautrix.RespSendToDevice]{Name: ReqSendToDevice}
	// ResolveAlias resolves a room alias to the ID and list of participating servers.
	ResolveAlias = &CommandSpec[*ResolveAliasParams, *mautrix.RespAliasResolve]{Name: ReqResolveAlias}
	// RequestOpenIDToken returns an OpenID token from the homeserver. OpenID tokens are used to
	// authenticate with various external services. Widgets also need this method.
	//
	// To log into css.gomuks.app, use the response data to form the following URL and open it in
	// a browser: `https://css.gomuks.app/login?token=${access_token}&server_name=${matrix_server_name}`
	RequestOpenIDToken = &CommandSpecWithoutRequest[*mautrix.RespOpenIDToken]{Name: ReqRequestOpenIDToken}
	// Logout logs out the current session. Note that this may break the process until it's restarted.
	Logout = &CommandSpecWithoutData{Name: ReqLogout}
	// Login logs into a homeserver using a username and password. After a successful login,
	// the `client_state` event will be dispatched. The frontend should use the event rather than
	// the response to this method to update its state.
	Login = &CommandSpecWithoutResponse[*LoginParams]{Name: ReqLogin}
	// LoginCustom sends a custom login request. Like the `login` request, this will also dispatch
	// a `client_state` event after a successful login.
	LoginCustom = &CommandSpecWithoutResponse[*LoginCustomParams]{Name: ReqLoginCustom}
	// Verify verifies the session using a recovery key or recovery phrase. Like the `login`
	// request, this will also dispatch a `client_state` event after successfully verifying.
	Verify = &CommandSpecWithoutResponse[*VerifyParams]{Name: ReqVerify}
	// DiscoverHomeserver performs `.well-known` lookup on the server name of the given user ID and
	// returns the results.
	DiscoverHomeserver = &CommandSpec[*DiscoverHomeserverParams, *mautrix.ClientWellKnown]{Name: ReqDiscoverHomeserver}
	// GetLoginFlows returns the available login flows on the given homeserver.
	GetLoginFlows = &CommandSpec[*GetLoginFlowsParams, *mautrix.RespLoginFlows]{Name: ReqGetLoginFlows}
	// RegisterPush stores a gomuks-specific pusher in the database. This will not register a
	// pusher on the homeserver. Push notifications will not work without the gomuks backend
	// being online.
	RegisterPush = &CommandSpecWithoutResponse[*database.PushRegistration]{Name: ReqRegisterPush}
	// ListenToDevice toggles including to-device messages in `sync_complete` events. Only relevant for widgets.
	// Returns the previous value of the setting.
	ListenToDevice = &CommandSpec[bool, bool]{Name: ReqListenToDevice}
	// GetTurnServers returns TURN server credentials from the homeserver.
	GetTurnServers = &CommandSpecWithoutRequest[*mautrix.RespTurnServer]{Name: ReqGetTurnServers}
	// GetMediaConfig returns the homeserver's media repository configuration (e.g. upload size limit)
	GetMediaConfig = &CommandSpecWithoutRequest[*mautrix.RespMediaConfig]{Name: ReqGetMediaConfig}
	// CalculateRoomID calculates a room ID locally from a timestamp and creation content. This is
	// only relevant when creating v12+ rooms with the `fi.mau.origin_server_ts` extension that
	// allows the client to pre-calculate the room ID.
	CalculateRoomID = &CommandSpec[*CalculateRoomIDParams, id.RoomID]{Name: ReqCalculateRoomID}
)

// FFI-specific command specs
var (
	SpecGetAccountInfo = &CommandSpecWithoutRequest[*database.Account]{Name: ReqGetAccountInfo}
	SpecUploadMedia    = &CommandSpec[*UploadMediaParams, *event.MessageEventContent]{Name: ReqUploadMedia}
	SpecExportKeys     = &CommandSpec[*ExportKeysParams, string]{Name: ReqExportKeys}
)

// Backend -> frontend event specs
var (
	SpecSyncComplete    = &EventSpec[*SyncComplete]{Name: EventSyncComplete}
	SpecSyncStatus      = &EventSpec[*SyncStatus]{Name: EventSyncStatus}
	SpecEventsDecrypted = &EventSpec[*EventsDecrypted]{Name: EventEventsDecrypted}
	SpecTyping          = &EventSpec[*Typing]{Name: EventTyping}
	SpecSendComplete    = &EventSpec[*SendComplete]{Name: EventSendComplete}
	SpecClientState     = &EventSpec[*ClientState]{Name: EventClientState}
)

// Websocket-specific backend -> frontend event specs
var (
	SpecImageAuthToken = &EventSpec[ImageAuthToken]{Name: EventImageAuthToken}
	SpecInitComplete   = &EventSpec[Empty]{Name: EventInitComplete}
	SpecRunID          = &EventSpec[*RunData]{Name: EventRunID}
)

var AllNames = []Name{
	ReqGetState,
	ReqCancel,
	ReqSendMessage,
	ReqSendEvent,
	ReqResendEvent,
	ReqReportEvent,
	ReqRedactEvent,
	ReqSetState,
	ReqUpdateDelayedEvent,
	ReqSetMembership,
	ReqSetAccountData,
	ReqMarkRead,
	ReqSetTyping,
	ReqGetProfile,
	ReqSetProfileField,
	ReqGetMutualRooms,
	ReqTrackUserDevices,
	ReqGetProfileEncryptionInfo,
	ReqGetEvent,
	ReqGetEventContext,
	ReqPaginateManual,
	ReqGetMentions,
	ReqGetRelatedEvents,
	ReqGetRoomState,
	ReqGetSpecificRoomState,
	ReqGetReceipts,
	ReqPaginate,
	ReqGetRoomSummary,
	ReqGetSpaceHierarchy,
	ReqJoinRoom,
	ReqKnockRoom,
	ReqLeaveRoom,
	ReqCreateRoom,
	ReqMuteRoom,
	ReqEnsureGroupSessionShared,
	ReqSendToDevice,
	ReqResolveAlias,
	ReqRequestOpenIDToken,
	ReqLogout,
	ReqLogin,
	ReqLoginCustom,
	ReqVerify,
	ReqDiscoverHomeserver,
	ReqGetLoginFlows,
	ReqRegisterPush,
	ReqListenToDevice,
	ReqGetTurnServers,
	ReqGetMediaConfig,
	ReqCalculateRoomID,
	ReqGetAccountInfo,
	ReqUploadMedia,
	RespError,
	RespSuccess,
	ReqPing,
	RespPong,
	EventSyncComplete,
	EventSyncStatus,
	EventEventsDecrypted,
	EventTyping,
	EventSendComplete,
	EventClientState,
	EventImageAuthToken,
	EventInitComplete,
	EventRunID,
}
