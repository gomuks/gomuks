// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/pushrules"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

func (h *HiClient) handleJSONCommand(ctx context.Context, req *JSONCommand) (any, error) {
	switch req.Command {
	case jsoncmd.ReqGetState:
		return jsoncmd.GetState.RunCtx(ctx, req.Data, h.API.GetState)
	case jsoncmd.ReqCancel:
		return jsoncmd.Cancel.Run(req.Data, func(params *jsoncmd.CancelRequestParams) (bool, error) {
			h.jsonRequestsLock.Lock()
			cancelTarget, ok := h.jsonRequests[params.RequestID]
			h.jsonRequestsLock.Unlock()
			if !ok {
				return false, nil
			}
			if params.Reason == "" {
				cancelTarget(nil)
			} else {
				cancelTarget(errors.New(params.Reason))
			}
			return true, nil
		})
	case jsoncmd.ReqSendMessage:
		return jsoncmd.SendMessage.RunCtx(ctx, req.Data, h.API.SendMessage)
	case jsoncmd.ReqSendEvent:
		return jsoncmd.SendEvent.RunCtx(ctx, req.Data, h.API.SendEvent)
	case jsoncmd.ReqResendEvent:
		return jsoncmd.ResendEvent.RunCtx(ctx, req.Data, h.API.ResendEvent)
	case jsoncmd.ReqReportEvent:
		return jsoncmd.ReportEvent.RunCtx(ctx, req.Data, h.API.ReportEvent)
	case jsoncmd.ReqRedactEvent:
		return jsoncmd.RedactEvent.RunCtx(ctx, req.Data, h.API.RedactEvent)
	case jsoncmd.ReqSetState:
		return jsoncmd.SetState.RunCtx(ctx, req.Data, h.API.SetState)
	case jsoncmd.ReqUpdateDelayedEvent:
		return jsoncmd.UpdateDelayedEvent.RunCtx(ctx, req.Data, h.API.UpdateDelayedEvent)
	case jsoncmd.ReqSetMembership:
		return jsoncmd.SetMembership.RunCtx(ctx, req.Data, h.API.SetMembership)
	case jsoncmd.ReqSetAccountData:
		return jsoncmd.SetAccountData.RunCtx(ctx, req.Data, h.API.SetAccountData)
	case jsoncmd.ReqMarkRead:
		return jsoncmd.MarkRead.RunCtx(ctx, req.Data, h.API.MarkRead)
	case jsoncmd.ReqSetTyping:
		return jsoncmd.SetTyping.RunCtx(ctx, req.Data, h.API.SetTyping)
	case jsoncmd.ReqGetProfile:
		return jsoncmd.GetProfile.RunCtx(ctx, req.Data, h.API.GetProfile)
	case jsoncmd.ReqSetProfileField:
		return jsoncmd.SetProfileField.RunCtx(ctx, req.Data, h.API.SetProfileField)
	case jsoncmd.ReqGetMutualRooms:
		return jsoncmd.GetMutualRooms.RunCtx(ctx, req.Data, h.API.GetMutualRooms)
	case jsoncmd.ReqTrackUserDevices:
		return jsoncmd.TrackUserDevices.RunCtx(ctx, req.Data, h.API.TrackUserDevices)
	case jsoncmd.ReqGetProfileEncryptionInfo:
		return jsoncmd.GetProfileEncryptionInfo.RunCtx(ctx, req.Data, h.API.GetProfileEncryptionInfo)
	case jsoncmd.ReqGetEvent:
		return jsoncmd.GetEvent.RunCtx(ctx, req.Data, h.API.GetEvent)
	case jsoncmd.ReqGetRelatedEvents:
		return jsoncmd.GetRelatedEvents.RunCtx(ctx, req.Data, h.API.GetRelatedEvents)
	case jsoncmd.ReqGetEventContext:
		return jsoncmd.GetEventContext.RunCtx(ctx, req.Data, h.API.GetEventContext)
	case jsoncmd.ReqPaginateManual:
		return jsoncmd.PaginateManual.RunCtx(ctx, req.Data, h.API.PaginateManual)
	case jsoncmd.ReqGetMentions:
		return jsoncmd.GetMentions.RunCtx(ctx, req.Data, h.API.GetMentions)
	case jsoncmd.ReqGetRoomState:
		return jsoncmd.GetRoomState.RunCtx(ctx, req.Data, h.API.GetRoomState)
	case jsoncmd.ReqGetSpecificRoomState:
		return jsoncmd.GetSpecificRoomState.RunCtx(ctx, req.Data, h.API.GetSpecificRoomState)
	case jsoncmd.ReqGetReceipts:
		return jsoncmd.GetReceipts.RunCtx(ctx, req.Data, h.API.GetReceipts)
	case jsoncmd.ReqPaginate:
		return jsoncmd.Paginate.RunCtx(ctx, req.Data, h.API.Paginate)
	case jsoncmd.ReqGetRoomSummary:
		return jsoncmd.GetRoomSummary.RunCtx(ctx, req.Data, h.API.GetRoomSummary)
	case jsoncmd.ReqGetSpaceHierarchy:
		return jsoncmd.GetSpaceHierarchy.RunCtx(ctx, req.Data, h.API.GetSpaceHierarchy)
	case jsoncmd.ReqJoinRoom:
		return jsoncmd.JoinRoom.RunCtx(ctx, req.Data, h.API.JoinRoom)
	case jsoncmd.ReqKnockRoom:
		return jsoncmd.KnockRoom.RunCtx(ctx, req.Data, h.API.KnockRoom)
	case jsoncmd.ReqLeaveRoom:
		return jsoncmd.LeaveRoom.RunCtx(ctx, req.Data, h.API.LeaveRoom)
	case jsoncmd.ReqCreateRoom:
		return jsoncmd.CreateRoom.RunCtx(ctx, req.Data, h.API.CreateRoom)
	case jsoncmd.ReqMuteRoom:
		return jsoncmd.MuteRoom.RunCtx(ctx, req.Data, h.API.MuteRoom)
	case jsoncmd.ReqEnsureGroupSessionShared:
		return jsoncmd.EnsureGroupSessionShared.RunCtx(ctx, req.Data, h.API.EnsureGroupSessionShared)
	case jsoncmd.ReqSendToDevice:
		return jsoncmd.SendToDevice.RunCtx(ctx, req.Data, h.API.SendToDevice)
	case jsoncmd.ReqResolveAlias:
		return jsoncmd.ResolveAlias.RunCtx(ctx, req.Data, h.API.ResolveAlias)
	case jsoncmd.ReqRequestOpenIDToken:
		return jsoncmd.RequestOpenIDToken.RunCtx(ctx, req.Data, h.API.RequestOpenIDToken)
	case jsoncmd.ReqLogout:
		return jsoncmd.Logout.RunCtx(ctx, req.Data, h.API.Logout)
	case jsoncmd.ReqLogin:
		return jsoncmd.Login.RunCtx(ctx, req.Data, h.API.Login)
	case jsoncmd.ReqLoginCustom:
		return jsoncmd.LoginCustom.RunCtx(ctx, req.Data, h.API.LoginCustom)
	case jsoncmd.ReqVerify:
		return jsoncmd.Verify.RunCtx(ctx, req.Data, h.API.Verify)
	case jsoncmd.ReqDiscoverHomeserver:
		return jsoncmd.DiscoverHomeserver.RunCtx(ctx, req.Data, h.API.DiscoverHomeserver)
	case jsoncmd.ReqGetLoginFlows:
		return jsoncmd.GetLoginFlows.RunCtx(ctx, req.Data, h.API.GetLoginFlows)
	case jsoncmd.ReqRegisterPush:
		return jsoncmd.RegisterPush.RunCtx(ctx, req.Data, h.API.RegisterPush)
	case jsoncmd.ReqListenToDevice:
		return jsoncmd.ListenToDevice.RunCtx(ctx, req.Data, h.API.ListenToDevice)
	case jsoncmd.ReqGetTurnServers:
		return jsoncmd.GetTurnServers.RunCtx(ctx, req.Data, h.API.GetTurnServers)
	case jsoncmd.ReqGetMediaConfig:
		return jsoncmd.GetMediaConfig.RunCtx(ctx, req.Data, h.API.GetMediaConfig)
	case jsoncmd.ReqCalculateRoomID:
		return jsoncmd.CalculateRoomID.RunCtx(ctx, req.Data, h.API.CalculateRoomID)
	case jsoncmd.ReqRerequestSession:
		return jsoncmd.RerequestSession.RunCtx(ctx, req.Data, h.API.RerequestSession)
	default:
		return nil, fmt.Errorf("unknown command %q", req.Command)
	}
}

type JSONAPI struct {
	*HiClient
}

var _ jsoncmd.GomuksAPI = (*JSONAPI)(nil)

func (h *JSONAPI) GetState(ctx context.Context) (*jsoncmd.ClientState, error) {
	return h.State(), nil
}

func (h *JSONAPI) SendMessage(ctx context.Context, params *jsoncmd.SendMessageParams) (*database.Event, error) {
	return h.HiClient.SendMessage(ctx, params.RoomID, params.BaseContent, params.Extra, params.Text, params.RelatesTo, params.Mentions, params.URLPreviews)
}

func (h *JSONAPI) SendEvent(ctx context.Context, params *jsoncmd.SendEventParams) (*database.Event, error) {
	return h.Send(ctx, params.RoomID, params.EventType, params.Content, params.DisableEncryption, params.Synchronous)
}

func (h *JSONAPI) ResendEvent(ctx context.Context, params *jsoncmd.ResendEventParams) (*database.Event, error) {
	return h.Resend(ctx, params.TransactionID)
}

func (h *JSONAPI) ReportEvent(ctx context.Context, params *jsoncmd.ReportEventParams) error {
	return h.Client.ReportEvent(ctx, params.RoomID, params.EventID, params.Reason)
}

func (h *JSONAPI) RedactEvent(ctx context.Context, params *jsoncmd.RedactEventParams) (*mautrix.RespSendEvent, error) {
	return h.Client.RedactEvent(ctx, params.RoomID, params.EventID, mautrix.ReqRedact{
		Reason: params.Reason,
	})
}

func (h *JSONAPI) SetState(ctx context.Context, params *jsoncmd.SendStateEventParams) (id.EventID, error) {
	return h.HiClient.SetState(ctx, params.RoomID, params.EventType, params.StateKey, params.Content, mautrix.ReqSendEvent{
		UnstableDelay: time.Duration(params.DelayMS) * time.Millisecond,
	})
}

func (h *JSONAPI) UpdateDelayedEvent(ctx context.Context, params *jsoncmd.UpdateDelayedEventParams) (*mautrix.RespUpdateDelayedEvent, error) {
	return h.Client.UpdateDelayedEvent(ctx, &mautrix.ReqUpdateDelayedEvent{
		DelayID: params.DelayID,
		Action:  params.Action,
	})
}

func (h *JSONAPI) SetMembership(ctx context.Context, params *jsoncmd.SetMembershipParams) (err error) {
	switch params.Action {
	case "invite":
		_, err = h.Client.InviteUser(ctx, params.RoomID, &mautrix.ReqInviteUser{UserID: params.UserID, Reason: params.Reason})
	case "kick":
		_, err = h.Client.KickUser(ctx, params.RoomID, &mautrix.ReqKickUser{UserID: params.UserID, Reason: params.Reason})
	case "ban":
		_, err = h.Client.BanUser(ctx, params.RoomID, &mautrix.ReqBanUser{UserID: params.UserID, Reason: params.Reason, MSC4293RedactEvents: params.MSC4293RedactEvents})
	case "unban":
		_, err = h.Client.UnbanUser(ctx, params.RoomID, &mautrix.ReqUnbanUser{UserID: params.UserID, Reason: params.Reason})
	default:
		err = fmt.Errorf("unknown action %q", params.Action)
	}
	return
}

func (h *JSONAPI) SetAccountData(ctx context.Context, params *jsoncmd.SetAccountDataParams) error {
	if params.RoomID != "" {
		return h.Client.SetRoomAccountData(ctx, params.RoomID, params.Type, params.Content)
	}
	return h.Client.SetAccountData(ctx, params.Type, params.Content)
}

func (h *JSONAPI) MarkRead(ctx context.Context, params *jsoncmd.MarkReadParams) error {
	return h.HiClient.MarkRead(ctx, params.RoomID, params.EventID, params.ReceiptType)
}

func (h *JSONAPI) SetTyping(ctx context.Context, params *jsoncmd.SetTypingParams) error {
	return h.HiClient.SetTyping(ctx, params.RoomID, time.Duration(params.Timeout)*time.Millisecond)
}

func (h *JSONAPI) GetProfile(ctx context.Context, params *jsoncmd.GetProfileParams) (*mautrix.RespUserProfile, error) {
	return h.Client.GetProfile(mautrix.WithMaxRetries(ctx, 0), params.UserID)
}

func (h *JSONAPI) SetProfileField(ctx context.Context, params *jsoncmd.SetProfileFieldParams) error {
	return h.Client.SetProfileField(ctx, params.Field, params.Value)
}

func (h *JSONAPI) GetMutualRooms(ctx context.Context, params *jsoncmd.GetProfileParams) ([]id.RoomID, error) {
	return h.HiClient.GetMutualRooms(mautrix.WithMaxRetries(ctx, 0), params.UserID)
}

func (h *JSONAPI) TrackUserDevices(ctx context.Context, params *jsoncmd.GetProfileParams) (*jsoncmd.ProfileEncryptionInfo, error) {
	err := h.HiClient.TrackUserDevices(ctx, params.UserID)
	if err != nil {
		return nil, err
	}
	return h.HiClient.GetProfileEncryptionInfo(ctx, params.UserID)
}

func (h *JSONAPI) GetProfileEncryptionInfo(ctx context.Context, params *jsoncmd.GetProfileParams) (*jsoncmd.ProfileEncryptionInfo, error) {
	return h.HiClient.GetProfileEncryptionInfo(ctx, params.UserID)
}

func (h *JSONAPI) GetEvent(ctx context.Context, params *jsoncmd.GetEventParams) (*database.Event, error) {
	if params.Unredact {
		return h.GetUnredactedEvent(mautrix.WithMaxRetries(ctx, 2), params.RoomID, params.EventID)
	}
	return h.HiClient.GetEvent(mautrix.WithMaxRetries(ctx, 2), params.RoomID, params.EventID)
}

func (h *JSONAPI) GetRelatedEvents(ctx context.Context, params *jsoncmd.GetRelatedEventsParams) ([]*database.Event, error) {
	return nonNilArray(h.DB.Event.GetRelatedEvents(ctx, params.RoomID, params.EventID, params.RelationType))
}

func (h *JSONAPI) GetEventContext(ctx context.Context, params *jsoncmd.GetEventContextParams) (*jsoncmd.EventContextResponse, error) {
	return h.HiClient.GetEventContext(mautrix.WithMaxRetries(ctx, 0), params.RoomID, params.EventID, params.Limit)
}

func (h *JSONAPI) GetRoomState(ctx context.Context, params *jsoncmd.GetRoomStateParams) ([]*database.Event, error) {
	return h.HiClient.GetRoomState(ctx, params.RoomID, params.IncludeMembers, params.FetchMembers, params.Refetch)
}

func (h *JSONAPI) GetSpecificRoomState(ctx context.Context, params *jsoncmd.GetSpecificRoomStateParams) ([]*database.Event, error) {
	return nonNilArray(h.DB.CurrentState.GetMany(ctx, params.Keys))
}

func (h *JSONAPI) GetReceipts(ctx context.Context, params *jsoncmd.GetReceiptsParams) (map[id.EventID][]*database.Receipt, error) {
	return h.HiClient.GetReceipts(ctx, params.RoomID, params.EventIDs)
}

func (h *JSONAPI) Paginate(ctx context.Context, params *jsoncmd.PaginateParams) (*jsoncmd.PaginationResponse, error) {
	return h.HiClient.Paginate(ctx, params.RoomID, params.MaxTimelineID, params.Limit, params.Reset)
}

func (h *JSONAPI) PaginateManual(ctx context.Context, params *jsoncmd.PaginateManualParams) (*jsoncmd.ManualPaginationResponse, error) {
	return h.HiClient.PaginateManual(mautrix.WithMaxRetries(ctx, 0), params.RoomID, params.ThreadRoot, params.Since, params.Direction, params.Limit)
}

func (h *JSONAPI) GetMentions(ctx context.Context, params *jsoncmd.GetMentionsParams) ([]*database.Event, error) {
	return nonNilArray(h.HiClient.GetMentions(ctx, params.MaxTimestamp.Time, params.Type, params.Limit, params.RoomID))
}

func (h *JSONAPI) GetRoomSummary(ctx context.Context, params *jsoncmd.GetRoomSummaryParams) (*mautrix.RespRoomSummary, error) {
	return h.Client.GetRoomSummary(mautrix.WithMaxRetries(ctx, 2), params.RoomIDOrAlias, params.Via...)
}

func (h *JSONAPI) GetSpaceHierarchy(ctx context.Context, params *jsoncmd.GetHierarchyParams) (*mautrix.RespHierarchy, error) {
	return h.Client.Hierarchy(mautrix.WithMaxRetries(ctx, 0), params.RoomID, &mautrix.ReqHierarchy{
		From:          params.From,
		Limit:         params.Limit,
		MaxDepth:      params.MaxDepth,
		SuggestedOnly: params.SuggestedOnly,
	})
}

func (h *JSONAPI) JoinRoom(ctx context.Context, params *jsoncmd.JoinRoomParams) (*mautrix.RespJoinRoom, error) {
	return h.Client.JoinRoom(mautrix.WithMaxRetries(ctx, 2), params.RoomIDOrAlias, &mautrix.ReqJoinRoom{
		Via:    params.Via,
		Reason: params.Reason,
	})
}

func (h *JSONAPI) KnockRoom(ctx context.Context, params *jsoncmd.JoinRoomParams) (*mautrix.RespKnockRoom, error) {
	return h.Client.KnockRoom(mautrix.WithMaxRetries(ctx, 2), params.RoomIDOrAlias, &mautrix.ReqKnockRoom{
		Via:    params.Via,
		Reason: params.Reason,
	})
}

func (h *JSONAPI) LeaveRoom(ctx context.Context, params *jsoncmd.LeaveRoomParams) (*mautrix.RespLeaveRoom, error) {
	resp, err := h.Client.LeaveRoom(mautrix.WithMaxRetries(ctx, 2), params.RoomID, &mautrix.ReqLeave{Reason: params.Reason})
	if err == nil ||
		errors.Is(err, mautrix.MNotFound) ||
		errors.Is(err, mautrix.MForbidden) ||
		// Synapse-specific hack: the server incorrectly returns M_UNKNOWN in some cases
		// instead of a sensible code like M_NOT_FOUND.
		strings.Contains(err.Error(), "Not a known room") {
		deleteInviteErr := h.DB.InvitedRoom.Delete(ctx, params.RoomID)
		if deleteInviteErr != nil {
			zerolog.Ctx(ctx).Err(deleteInviteErr).
				Stringer("room_id", params.RoomID).
				Msg("Failed to delete invite from database after leaving room")
		} else {
			zerolog.Ctx(ctx).Debug().
				Stringer("room_id", params.RoomID).
				Msg("Deleted invite from database after leaving room")
		}
	}
	return resp, err
}

func (h *JSONAPI) CreateRoom(ctx context.Context, params *mautrix.ReqCreateRoom) (*mautrix.RespCreateRoom, error) {
	return h.Client.CreateRoom(mautrix.WithMaxRetries(ctx, 0), params)
}

func (h *JSONAPI) MuteRoom(ctx context.Context, params *jsoncmd.MuteRoomParams) (bool, error) {
	if params.Muted {
		return true, h.Client.PutPushRule(ctx, "global", pushrules.RoomRule, string(params.RoomID), &mautrix.ReqPutPushRule{
			Actions: []pushrules.PushActionType{},
		})
	}
	return false, h.Client.DeletePushRule(ctx, "global", pushrules.RoomRule, string(params.RoomID))
}

func (h *JSONAPI) EnsureGroupSessionShared(ctx context.Context, params *jsoncmd.EnsureGroupSessionSharedParams) error {
	return h.HiClient.EnsureGroupSessionShared(ctx, params.RoomID)
}

func (h *JSONAPI) SendToDevice(ctx context.Context, params *jsoncmd.SendToDeviceParams) (*mautrix.RespSendToDevice, error) {
	params.EventType.Class = event.ToDeviceEventType
	return h.HiClient.SendToDevice(ctx, params.EventType, params.ReqSendToDevice, params.Encrypted)
}

func (h *JSONAPI) ResolveAlias(ctx context.Context, params *jsoncmd.ResolveAliasParams) (*mautrix.RespAliasResolve, error) {
	return h.Client.ResolveAlias(mautrix.WithMaxRetries(ctx, 0), params.Alias)
}

func (h *JSONAPI) RequestOpenIDToken(ctx context.Context) (*mautrix.RespOpenIDToken, error) {
	return h.Client.RequestOpenIDToken(ctx)
}

func (h *JSONAPI) Logout(ctx context.Context) error {
	if h.LogoutFunc == nil {
		return errors.New("logout not supported")
	}
	return h.LogoutFunc(ctx)
}

func (h *JSONAPI) Login(ctx context.Context, params *jsoncmd.LoginParams) error {
	err := h.LoginPassword(ctx, params.HomeserverURL, params.Username, params.Password)
	if err != nil {
		h.Log.Err(err).Msg("Failed to login")
	}
	return err
}

func (h *JSONAPI) LoginCustom(ctx context.Context, params *jsoncmd.LoginCustomParams) error {
	var err error
	h.Client.HomeserverURL, err = url.Parse(params.HomeserverURL)
	if err != nil {
		return err
	}
	err = h.HiClient.Login(ctx, params.Request)
	if err != nil {
		h.Log.Err(err).Msg("Failed to login")
	}
	return err
}

func (h *JSONAPI) Verify(ctx context.Context, params *jsoncmd.VerifyParams) error {
	return h.HiClient.Verify(ctx, params.RecoveryKey)
}

func (h *JSONAPI) DiscoverHomeserver(ctx context.Context, params *jsoncmd.DiscoverHomeserverParams) (*mautrix.ClientWellKnown, error) {
	_, homeserver, err := params.UserID.Parse()
	if err != nil {
		return nil, err
	}
	return mautrix.DiscoverClientAPI(ctx, homeserver)
}

func (h *JSONAPI) GetLoginFlows(ctx context.Context, params *jsoncmd.GetLoginFlowsParams) (*mautrix.RespLoginFlows, error) {
	cli, err := h.tempClient(params.HomeserverURL)
	if err != nil {
		return nil, err
	}
	err = h.checkServerVersions(ctx, cli)
	if err != nil {
		return nil, err
	}
	return cli.GetLoginFlows(ctx)
}

func (h *JSONAPI) RegisterPush(ctx context.Context, params *database.PushRegistration) error {
	return h.DB.PushRegistration.Put(ctx, params)
}

func (h *JSONAPI) ListenToDevice(ctx context.Context, listen bool) (bool, error) {
	return h.ToDeviceInSync.Swap(listen), nil
}

func (h *JSONAPI) GetTurnServers(ctx context.Context) (*mautrix.RespTurnServer, error) {
	return h.Client.TurnServer(ctx)
}

func (h *JSONAPI) GetMediaConfig(ctx context.Context) (*mautrix.RespMediaConfig, error) {
	return h.Client.GetMediaConfig(ctx)
}

func (h *JSONAPI) CalculateRoomID(ctx context.Context, params *jsoncmd.CalculateRoomIDParams) (id.RoomID, error) {
	return h.HiClient.CalculateRoomID(params.Timestamp, params.CreationContent)
}

func (h *JSONAPI) RerequestSession(ctx context.Context, params *jsoncmd.RerequestSessionParams) error {
	decoded, _ := base64.RawStdEncoding.DecodeString(string(params.SessionID))
	if len(decoded) != 32 {
		return fmt.Errorf("invalid session ID")
	}
	err := h.DB.SessionRequest.Overwrite(ctx, &database.SessionRequest{
		RoomID:        params.RoomID,
		SessionID:     params.SessionID,
		Sender:        params.Sender,
		MinIndex:      10000,
		BackupChecked: false,
		RequestSent:   false,
	})
	if err != nil {
		return err
	}
	h.WakeupRequestQueue()
	return nil
}

func nonNilArray[T any](arr []T, err error) ([]T, error) {
	if arr == nil && err == nil {
		return []T{}, nil
	}
	return arr, err
}
