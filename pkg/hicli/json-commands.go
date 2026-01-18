// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"context"
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
		return jsoncmd.GetState.Run(req.Data, func() (*jsoncmd.ClientState, error) {
			return h.State(), nil
		})
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
		return jsoncmd.SendMessage.Run(req.Data, func(params *jsoncmd.SendMessageParams) (*database.Event, error) {
			return h.SendMessage(ctx, params.RoomID, params.BaseContent, params.Extra, params.Text, params.RelatesTo, params.Mentions, params.URLPreviews)
		})
	case jsoncmd.ReqSendEvent:
		return jsoncmd.SendEvent.Run(req.Data, func(params *jsoncmd.SendEventParams) (*database.Event, error) {
			return h.Send(ctx, params.RoomID, params.EventType, params.Content, params.DisableEncryption, params.Synchronous, time.Duration(params.StickyDurationMS)*time.Millisecond)
		})
	case jsoncmd.ReqSendDelayedEvent:
		return jsoncmd.SendDelayedEvent.Run(req.Data, func(params *jsoncmd.SendDelayedEventParams) (id.DelayID, error) {
			extra := mautrix.ReqSendEvent{
				UnstableDelay:          time.Duration(params.DelayMS) * time.Millisecond,
				UnstableStickyDuration: time.Duration(params.StickyDurationMS) * time.Millisecond,
			}
			var resp *mautrix.RespSendEvent
			var err error
			if params.StateKey != nil {
				resp, err = h.Client.SendStateEvent(ctx, params.RoomID, params.EventType, *params.StateKey, params.Content, extra)
			} else {
				resp, err = h.Client.SendMessageEvent(ctx, params.RoomID, params.EventType, params.Content, extra)
			}
			if err != nil {
				return "", err
			}
			return resp.UnstableDelayID, nil
		})
	case jsoncmd.ReqResendEvent:
		return jsoncmd.ResendEvent.Run(req.Data, func(params *jsoncmd.ResendEventParams) (*database.Event, error) {
			return h.Resend(ctx, params.TransactionID)
		})
	case jsoncmd.ReqReportEvent:
		return jsoncmd.ReportEvent.Run(req.Data, func(params *jsoncmd.ReportEventParams) error {
			return h.Client.ReportEvent(ctx, params.RoomID, params.EventID, params.Reason)
		})
	case jsoncmd.ReqRedactEvent:
		return jsoncmd.RedactEvent.Run(req.Data, func(params *jsoncmd.RedactEventParams) (*mautrix.RespSendEvent, error) {
			return h.Client.RedactEvent(ctx, params.RoomID, params.EventID, mautrix.ReqRedact{
				Reason: params.Reason,
			})
		})
	case jsoncmd.ReqSetState:
		return jsoncmd.SetState.Run(req.Data, func(params *jsoncmd.SendStateEventParams) (id.EventID, error) {
			return h.SetState(ctx, params.RoomID, params.EventType, params.StateKey, params.Content)
		})
	case jsoncmd.ReqUpdateDelayedEvent:
		return jsoncmd.UpdateDelayedEvent.Run(req.Data, func(params *jsoncmd.UpdateDelayedEventParams) (*mautrix.RespUpdateDelayedEvent, error) {
			return h.Client.UpdateDelayedEvent(ctx, &mautrix.ReqUpdateDelayedEvent{
				DelayID: params.DelayID,
				Action:  params.Action,
			})
		})
	case jsoncmd.ReqSetMembership:
		return jsoncmd.SetMembership.Run(req.Data, func(params *jsoncmd.SetMembershipParams) (err error) {
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
		})
	case jsoncmd.ReqSetAccountData:
		return jsoncmd.SetAccountData.Run(req.Data, func(params *jsoncmd.SetAccountDataParams) error {
			if params.RoomID != "" {
				return h.Client.SetRoomAccountData(ctx, params.RoomID, params.Type, params.Content)
			}
			return h.Client.SetAccountData(ctx, params.Type, params.Content)
		})
	case jsoncmd.ReqMarkRead:
		return jsoncmd.MarkRead.Run(req.Data, func(params *jsoncmd.MarkReadParams) error {
			return h.MarkRead(ctx, params.RoomID, params.EventID, params.ReceiptType)
		})
	case jsoncmd.ReqSetTyping:
		return jsoncmd.SetTyping.Run(req.Data, func(params *jsoncmd.SetTypingParams) error {
			return h.SetTyping(ctx, params.RoomID, time.Duration(params.Timeout)*time.Millisecond)
		})
	case jsoncmd.ReqGetProfile:
		return jsoncmd.GetProfile.Run(req.Data, func(params *jsoncmd.GetProfileParams) (*mautrix.RespUserProfile, error) {
			return h.Client.GetProfile(mautrix.WithMaxRetries(ctx, 0), params.UserID)
		})
	case jsoncmd.ReqSetProfileField:
		return jsoncmd.SetProfileField.Run(req.Data, func(params *jsoncmd.SetProfileFieldParams) error {
			return h.Client.SetProfileField(ctx, params.Field, params.Value)
		})
	case jsoncmd.ReqGetMutualRooms:
		return jsoncmd.GetMutualRooms.Run(req.Data, func(params *jsoncmd.GetProfileParams) ([]id.RoomID, error) {
			return h.GetMutualRooms(mautrix.WithMaxRetries(ctx, 0), params.UserID)
		})
	case jsoncmd.ReqTrackUserDevices:
		return jsoncmd.TrackUserDevices.Run(req.Data, func(params *jsoncmd.GetProfileParams) (*jsoncmd.ProfileEncryptionInfo, error) {
			err := h.TrackUserDevices(ctx, params.UserID)
			if err != nil {
				return nil, err
			}
			return h.GetProfileEncryptionInfo(ctx, params.UserID)
		})
	case jsoncmd.ReqGetProfileEncryptionInfo:
		return jsoncmd.GetProfileEncryptionInfo.Run(req.Data, func(params *jsoncmd.GetProfileParams) (*jsoncmd.ProfileEncryptionInfo, error) {
			return h.GetProfileEncryptionInfo(ctx, params.UserID)
		})
	case jsoncmd.ReqGetEvent:
		return jsoncmd.GetEvent.Run(req.Data, func(params *jsoncmd.GetEventParams) (*database.Event, error) {
			if params.Unredact {
				return h.GetUnredactedEvent(mautrix.WithMaxRetries(ctx, 2), params.RoomID, params.EventID)
			}
			return h.GetEvent(mautrix.WithMaxRetries(ctx, 2), params.RoomID, params.EventID)
		})
	case jsoncmd.ReqGetRelatedEvents:
		return jsoncmd.GetRelatedEvents.Run(req.Data, func(params *jsoncmd.GetRelatedEventsParams) ([]*database.Event, error) {
			return nonNilArray(h.DB.Event.GetRelatedEvents(ctx, params.RoomID, params.EventID, params.RelationType))
		})
	case jsoncmd.ReqGetEventContext:
		return jsoncmd.GetEventContext.Run(req.Data, func(params *jsoncmd.GetEventContextParams) (*jsoncmd.EventContextResponse, error) {
			return h.GetEventContext(mautrix.WithMaxRetries(ctx, 0), params.RoomID, params.EventID, params.Limit)
		})
	case jsoncmd.ReqPaginateManual:
		return jsoncmd.PaginateManual.Run(req.Data, func(params *jsoncmd.PaginateManualParams) (*jsoncmd.ManualPaginationResponse, error) {
			return h.PaginateManual(mautrix.WithMaxRetries(ctx, 0), params.RoomID, params.ThreadRoot, params.Since, params.Direction, params.Limit)
		})
	case jsoncmd.ReqGetMentions:
		return jsoncmd.GetMentions.Run(req.Data, func(params *jsoncmd.GetMentionsParams) ([]*database.Event, error) {
			return nonNilArray(h.GetMentions(ctx, params.MaxTimestamp.Time, params.Type, params.Limit, params.RoomID))
		})
	case jsoncmd.ReqGetRoomState:
		return jsoncmd.GetRoomState.Run(req.Data, func(params *jsoncmd.GetRoomStateParams) ([]*database.Event, error) {
			return h.GetRoomState(ctx, params.RoomID, params.IncludeMembers, params.FetchMembers, params.Refetch)
		})
	case jsoncmd.ReqGetSpecificRoomState:
		return jsoncmd.GetSpecificRoomState.Run(req.Data, func(params *jsoncmd.GetSpecificRoomStateParams) ([]*database.Event, error) {
			return nonNilArray(h.DB.CurrentState.GetMany(ctx, params.Keys))
		})
	case jsoncmd.ReqGetReceipts:
		return jsoncmd.GetReceipts.Run(req.Data, func(params *jsoncmd.GetReceiptsParams) (map[id.EventID][]*database.Receipt, error) {
			return h.GetReceipts(ctx, params.RoomID, params.EventIDs)
		})
	case jsoncmd.ReqPaginate:
		return jsoncmd.Paginate.Run(req.Data, func(params *jsoncmd.PaginateParams) (*jsoncmd.PaginationResponse, error) {
			return h.Paginate(ctx, params.RoomID, params.MaxTimelineID, params.Limit, params.Reset)
		})
	case jsoncmd.ReqGetRoomSummary:
		return jsoncmd.GetRoomSummary.Run(req.Data, func(params *jsoncmd.GetRoomSummaryParams) (*mautrix.RespRoomSummary, error) {
			return h.Client.GetRoomSummary(mautrix.WithMaxRetries(ctx, 2), params.RoomIDOrAlias, params.Via...)
		})
	case jsoncmd.ReqGetSpaceHierarchy:
		return jsoncmd.GetSpaceHierarchy.Run(req.Data, func(params *jsoncmd.GetHierarchyParams) (*mautrix.RespHierarchy, error) {
			return h.Client.Hierarchy(mautrix.WithMaxRetries(ctx, 0), params.RoomID, &mautrix.ReqHierarchy{
				From:          params.From,
				Limit:         params.Limit,
				MaxDepth:      params.MaxDepth,
				SuggestedOnly: params.SuggestedOnly,
			})
		})
	case jsoncmd.ReqJoinRoom:
		return jsoncmd.JoinRoom.Run(req.Data, func(params *jsoncmd.JoinRoomParams) (*mautrix.RespJoinRoom, error) {
			return h.Client.JoinRoom(mautrix.WithMaxRetries(ctx, 2), params.RoomIDOrAlias, &mautrix.ReqJoinRoom{
				Via:    params.Via,
				Reason: params.Reason,
			})
		})
	case jsoncmd.ReqKnockRoom:
		return jsoncmd.KnockRoom.Run(req.Data, func(params *jsoncmd.JoinRoomParams) (*mautrix.RespKnockRoom, error) {
			return h.Client.KnockRoom(mautrix.WithMaxRetries(ctx, 2), params.RoomIDOrAlias, &mautrix.ReqKnockRoom{
				Via:    params.Via,
				Reason: params.Reason,
			})
		})
	case jsoncmd.ReqLeaveRoom:
		return jsoncmd.LeaveRoom.Run(req.Data, func(params *jsoncmd.LeaveRoomParams) (*mautrix.RespLeaveRoom, error) {
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
		})
	case jsoncmd.ReqCreateRoom:
		return jsoncmd.CreateRoom.RunCtx(mautrix.WithMaxRetries(ctx, 0), req.Data, h.Client.CreateRoom)
	case jsoncmd.ReqMuteRoom:
		return jsoncmd.MuteRoom.Run(req.Data, func(params *jsoncmd.MuteRoomParams) (bool, error) {
			if params.Muted {
				return true, h.Client.PutPushRule(ctx, "global", pushrules.RoomRule, string(params.RoomID), &mautrix.ReqPutPushRule{
					Actions: []pushrules.PushActionType{},
				})
			}
			return false, h.Client.DeletePushRule(ctx, "global", pushrules.RoomRule, string(params.RoomID))
		})
	case jsoncmd.ReqEnsureGroupSessionShared:
		return jsoncmd.EnsureGroupSessionShared.Run(req.Data, func(params *jsoncmd.EnsureGroupSessionSharedParams) error {
			return h.EnsureGroupSessionShared(ctx, params.RoomID)
		})
	case jsoncmd.ReqSendToDevice:
		return jsoncmd.SendToDevice.Run(req.Data, func(params *jsoncmd.SendToDeviceParams) (*mautrix.RespSendToDevice, error) {
			params.EventType.Class = event.ToDeviceEventType
			return h.SendToDevice(ctx, params.EventType, params.ReqSendToDevice, params.Encrypted)
		})
	case jsoncmd.ReqResolveAlias:
		return jsoncmd.ResolveAlias.Run(req.Data, func(params *jsoncmd.ResolveAliasParams) (*mautrix.RespAliasResolve, error) {
			return h.Client.ResolveAlias(mautrix.WithMaxRetries(ctx, 0), params.Alias)
		})
	case jsoncmd.ReqRequestOpenIDToken:
		return jsoncmd.RequestOpenIDToken.RunCtx(ctx, req.Data, h.Client.RequestOpenIDToken)
	case jsoncmd.ReqLogout:
		return jsoncmd.Logout.Run(req.Data, func() error {
			if h.LogoutFunc == nil {
				return errors.New("logout not supported")
			}
			return h.LogoutFunc(ctx)
		})
	case jsoncmd.ReqLogin:
		return jsoncmd.Login.Run(req.Data, func(params *jsoncmd.LoginParams) error {
			err := h.LoginPassword(ctx, params.HomeserverURL, params.Username, params.Password)
			if err != nil {
				h.Log.Err(err).Msg("Failed to login")
			}
			return err
		})
	case jsoncmd.ReqLoginCustom:
		return jsoncmd.LoginCustom.Run(req.Data, func(params *jsoncmd.LoginCustomParams) error {
			var err error
			h.Client.HomeserverURL, err = url.Parse(params.HomeserverURL)
			if err != nil {
				return err
			}
			err = h.Login(ctx, params.Request)
			if err != nil {
				h.Log.Err(err).Msg("Failed to login")
			}
			return err
		})
	case jsoncmd.ReqVerify:
		return jsoncmd.Verify.Run(req.Data, func(params *jsoncmd.VerifyParams) error {
			return h.Verify(ctx, params.RecoveryKey)
		})
	case jsoncmd.ReqDiscoverHomeserver:
		return jsoncmd.DiscoverHomeserver.Run(req.Data, func(params *jsoncmd.DiscoverHomeserverParams) (*mautrix.ClientWellKnown, error) {
			_, homeserver, err := params.UserID.Parse()
			if err != nil {
				return nil, err
			}
			return mautrix.DiscoverClientAPI(ctx, homeserver)
		})
	case jsoncmd.ReqGetLoginFlows:
		return jsoncmd.GetLoginFlows.Run(req.Data, func(params *jsoncmd.GetLoginFlowsParams) (*mautrix.RespLoginFlows, error) {
			cli, err := h.tempClient(params.HomeserverURL)
			if err != nil {
				return nil, err
			}
			err = h.checkServerVersions(ctx, cli)
			if err != nil {
				return nil, err
			}
			return cli.GetLoginFlows(ctx)
		})
	case jsoncmd.ReqRegisterPush:
		return jsoncmd.RegisterPush.Run(req.Data, func(params *database.PushRegistration) error {
			return h.DB.PushRegistration.Put(ctx, params)
		})
	case jsoncmd.ReqListenToDevice:
		return jsoncmd.ListenToDevice.Run(req.Data, func(listen bool) (bool, error) {
			return h.ToDeviceInSync.Swap(listen), nil
		})
	case jsoncmd.ReqGetTurnServers:
		return jsoncmd.GetTurnServers.RunCtx(ctx, req.Data, h.Client.TurnServer)
	case jsoncmd.ReqGetMediaConfig:
		return jsoncmd.GetMediaConfig.RunCtx(ctx, req.Data, h.Client.GetMediaConfig)
	case jsoncmd.ReqCalculateRoomID:
		return jsoncmd.CalculateRoomID.Run(req.Data, func(params *jsoncmd.CalculateRoomIDParams) (id.RoomID, error) {
			return h.CalculateRoomID(params.Timestamp, params.CreationContent)
		})
	default:
		return nil, fmt.Errorf("unknown command %q", req.Command)
	}
}

func nonNilArray[T any](arr []T, err error) ([]T, error) {
	if arr == nil && err == nil {
		return []T{}, nil
	}
	return arr, err
}
