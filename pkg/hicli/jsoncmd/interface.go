package jsoncmd

import (
	"context"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
)

type GomuksAPI interface {
	GetState(ctx context.Context) (*ClientState, error)
	SendMessage(ctx context.Context, params *SendMessageParams) (*database.Event, error)
	SendEvent(ctx context.Context, params *SendEventParams) (*database.Event, error)
	ResendEvent(ctx context.Context, params *ResendEventParams) (*database.Event, error)
	ReportEvent(ctx context.Context, params *ReportEventParams) error
	RedactEvent(ctx context.Context, params *RedactEventParams) (*mautrix.RespSendEvent, error)
	SetState(ctx context.Context, params *SendStateEventParams) (id.EventID, error)
	UpdateDelayedEvent(ctx context.Context, params *UpdateDelayedEventParams) (*mautrix.RespUpdateDelayedEvent, error)
	SetMembership(ctx context.Context, params *SetMembershipParams) error
	SetAccountData(ctx context.Context, params *SetAccountDataParams) error
	MarkRead(ctx context.Context, params *MarkReadParams) error
	SetTyping(ctx context.Context, params *SetTypingParams) error
	GetProfile(ctx context.Context, params *GetProfileParams) (*mautrix.RespUserProfile, error)
	SetProfileField(ctx context.Context, params *SetProfileFieldParams) error
	GetMutualRooms(ctx context.Context, params *GetProfileParams) ([]id.RoomID, error)
	TrackUserDevices(ctx context.Context, params *GetProfileParams) (*ProfileEncryptionInfo, error)
	GetProfileEncryptionInfo(ctx context.Context, params *GetProfileParams) (*ProfileEncryptionInfo, error)
	GetEvent(ctx context.Context, params *GetEventParams) (*database.Event, error)
	GetRelatedEvents(ctx context.Context, params *GetRelatedEventsParams) ([]*database.Event, error)
	GetEventContext(ctx context.Context, params *GetEventContextParams) (*EventContextResponse, error)
	GetRoomState(ctx context.Context, params *GetRoomStateParams) ([]*database.Event, error)
	GetSpecificRoomState(ctx context.Context, params *GetSpecificRoomStateParams) ([]*database.Event, error)
	GetReceipts(ctx context.Context, params *GetReceiptsParams) (map[id.EventID][]*database.Receipt, error)
	Paginate(ctx context.Context, params *PaginateParams) (*PaginationResponse, error)
	PaginateManual(ctx context.Context, params *PaginateManualParams) (*ManualPaginationResponse, error)
	GetMentions(ctx context.Context, params *GetMentionsParams) ([]*database.Event, error)
	GetRoomSummary(ctx context.Context, params *GetRoomSummaryParams) (*mautrix.RespRoomSummary, error)
	GetSpaceHierarchy(ctx context.Context, params *GetHierarchyParams) (*mautrix.RespHierarchy, error)
	JoinRoom(ctx context.Context, params *JoinRoomParams) (*mautrix.RespJoinRoom, error)
	KnockRoom(ctx context.Context, params *JoinRoomParams) (*mautrix.RespKnockRoom, error)
	LeaveRoom(ctx context.Context, params *LeaveRoomParams) (*mautrix.RespLeaveRoom, error)
	CreateRoom(ctx context.Context, params *mautrix.ReqCreateRoom) (*mautrix.RespCreateRoom, error)
	MuteRoom(ctx context.Context, params *MuteRoomParams) (bool, error)
	EnsureGroupSessionShared(ctx context.Context, params *EnsureGroupSessionSharedParams) error
	SendToDevice(ctx context.Context, params *SendToDeviceParams) (*mautrix.RespSendToDevice, error)
	ResolveAlias(ctx context.Context, params *ResolveAliasParams) (*mautrix.RespAliasResolve, error)
	RequestOpenIDToken(ctx context.Context) (*mautrix.RespOpenIDToken, error)
	Logout(ctx context.Context) error
	Login(ctx context.Context, params *LoginParams) error
	LoginCustom(ctx context.Context, params *LoginCustomParams) error
	Verify(ctx context.Context, params *VerifyParams) error
	DiscoverHomeserver(ctx context.Context, params *DiscoverHomeserverParams) (*mautrix.ClientWellKnown, error)
	GetLoginFlows(ctx context.Context, params *GetLoginFlowsParams) (*mautrix.RespLoginFlows, error)
	RegisterPush(ctx context.Context, params *database.PushRegistration) error
	ListenToDevice(ctx context.Context, listen bool) (bool, error)
	GetTurnServers(ctx context.Context) (*mautrix.RespTurnServer, error)
	GetMediaConfig(ctx context.Context) (*mautrix.RespMediaConfig, error)
	CalculateRoomID(ctx context.Context, params *CalculateRoomIDParams) (id.RoomID, error)
}
