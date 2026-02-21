// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package rpc

import (
	"context"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

func (gr *GomuksRPC) GetState(ctx context.Context) (*jsoncmd.ClientState, error) {
	return executeRequest(gr, ctx, jsoncmd.GetState, nil)
}

func (gr *GomuksRPC) SendMessage(ctx context.Context, params *jsoncmd.SendMessageParams) (*database.Event, error) {
	return executeRequest(gr, ctx, jsoncmd.SendMessage, params)
}

func (gr *GomuksRPC) SendEvent(ctx context.Context, params *jsoncmd.SendEventParams) (*database.Event, error) {
	return executeRequest(gr, ctx, jsoncmd.SendEvent, params)
}

func (gr *GomuksRPC) ResendEvent(ctx context.Context, params *jsoncmd.ResendEventParams) (*database.Event, error) {
	return executeRequest(gr, ctx, jsoncmd.ResendEvent, params)
}

func (gr *GomuksRPC) ReportEvent(ctx context.Context, params *jsoncmd.ReportEventParams) error {
	return executeRequestNoResponse(gr, ctx, jsoncmd.ReportEvent, params)
}

func (gr *GomuksRPC) RedactEvent(ctx context.Context, params *jsoncmd.RedactEventParams) (*mautrix.RespSendEvent, error) {
	return executeRequest(gr, ctx, jsoncmd.RedactEvent, params)
}

func (gr *GomuksRPC) SetState(ctx context.Context, params *jsoncmd.SendStateEventParams) (id.EventID, error) {
	return executeRequest(gr, ctx, jsoncmd.SetState, params)
}

func (gr *GomuksRPC) UpdateDelayedEvent(ctx context.Context, params *jsoncmd.UpdateDelayedEventParams) (*mautrix.RespUpdateDelayedEvent, error) {
	return executeRequest(gr, ctx, jsoncmd.UpdateDelayedEvent, params)
}

func (gr *GomuksRPC) SetMembership(ctx context.Context, params *jsoncmd.SetMembershipParams) error {
	return executeRequestNoResponse(gr, ctx, jsoncmd.SetMembership, params)
}

func (gr *GomuksRPC) SetAccountData(ctx context.Context, params *jsoncmd.SetAccountDataParams) error {
	return executeRequestNoResponse(gr, ctx, jsoncmd.SetAccountData, params)
}

func (gr *GomuksRPC) MarkRead(ctx context.Context, params *jsoncmd.MarkReadParams) error {
	return executeRequestNoResponse(gr, ctx, jsoncmd.MarkRead, params)
}

func (gr *GomuksRPC) SetTyping(ctx context.Context, params *jsoncmd.SetTypingParams) error {
	return executeRequestNoResponse(gr, ctx, jsoncmd.SetTyping, params)
}

func (gr *GomuksRPC) GetProfile(ctx context.Context, params *jsoncmd.GetProfileParams) (*mautrix.RespUserProfile, error) {
	return executeRequest(gr, ctx, jsoncmd.GetProfile, params)
}

func (gr *GomuksRPC) SetProfileField(ctx context.Context, params *jsoncmd.SetProfileFieldParams) error {
	return executeRequestNoResponse(gr, ctx, jsoncmd.SetProfileField, params)
}

func (gr *GomuksRPC) GetMutualRooms(ctx context.Context, params *jsoncmd.GetProfileParams) ([]id.RoomID, error) {
	return executeRequest(gr, ctx, jsoncmd.GetMutualRooms, params)
}

func (gr *GomuksRPC) TrackUserDevices(ctx context.Context, params *jsoncmd.GetProfileParams) (*jsoncmd.ProfileEncryptionInfo, error) {
	return executeRequest(gr, ctx, jsoncmd.TrackUserDevices, params)
}

func (gr *GomuksRPC) GetProfileEncryptionInfo(ctx context.Context, params *jsoncmd.GetProfileParams) (*jsoncmd.ProfileEncryptionInfo, error) {
	return executeRequest(gr, ctx, jsoncmd.GetProfileEncryptionInfo, params)
}

func (gr *GomuksRPC) GetEvent(ctx context.Context, params *jsoncmd.GetEventParams) (*database.Event, error) {
	return executeRequest(gr, ctx, jsoncmd.GetEvent, params)
}

func (gr *GomuksRPC) GetRelatedEvents(ctx context.Context, params *jsoncmd.GetRelatedEventsParams) ([]*database.Event, error) {
	return executeRequest(gr, ctx, jsoncmd.GetRelatedEvents, params)
}

func (gr *GomuksRPC) GetEventContext(ctx context.Context, params *jsoncmd.GetEventContextParams) (*jsoncmd.EventContextResponse, error) {
	return executeRequest(gr, ctx, jsoncmd.GetEventContext, params)
}

func (gr *GomuksRPC) GetRoomState(ctx context.Context, params *jsoncmd.GetRoomStateParams) ([]*database.Event, error) {
	return executeRequest(gr, ctx, jsoncmd.GetRoomState, params)
}

func (gr *GomuksRPC) GetSpecificRoomState(ctx context.Context, params *jsoncmd.GetSpecificRoomStateParams) ([]*database.Event, error) {
	return executeRequest(gr, ctx, jsoncmd.GetSpecificRoomState, params)
}

func (gr *GomuksRPC) GetReceipts(ctx context.Context, params *jsoncmd.GetReceiptsParams) (map[id.EventID][]*database.Receipt, error) {
	return executeRequest(gr, ctx, jsoncmd.GetReceipts, params)
}

func (gr *GomuksRPC) Paginate(ctx context.Context, params *jsoncmd.PaginateParams) (*jsoncmd.PaginationResponse, error) {
	return executeRequest(gr, ctx, jsoncmd.Paginate, params)
}

func (gr *GomuksRPC) PaginateManual(ctx context.Context, params *jsoncmd.PaginateManualParams) (*jsoncmd.ManualPaginationResponse, error) {
	return executeRequest(gr, ctx, jsoncmd.PaginateManual, params)
}

func (gr *GomuksRPC) GetMentions(ctx context.Context, params *jsoncmd.GetMentionsParams) ([]*database.Event, error) {
	return executeRequest(gr, ctx, jsoncmd.GetMentions, params)
}

func (gr *GomuksRPC) GetRoomSummary(ctx context.Context, params *jsoncmd.GetRoomSummaryParams) (*mautrix.RespRoomSummary, error) {
	return executeRequest(gr, ctx, jsoncmd.GetRoomSummary, params)
}

func (gr *GomuksRPC) GetSpaceHierarchy(ctx context.Context, params *jsoncmd.GetHierarchyParams) (*mautrix.RespHierarchy, error) {
	return executeRequest(gr, ctx, jsoncmd.GetSpaceHierarchy, params)
}

func (gr *GomuksRPC) JoinRoom(ctx context.Context, params *jsoncmd.JoinRoomParams) (*mautrix.RespJoinRoom, error) {
	return executeRequest(gr, ctx, jsoncmd.JoinRoom, params)
}

func (gr *GomuksRPC) KnockRoom(ctx context.Context, params *jsoncmd.JoinRoomParams) (*mautrix.RespKnockRoom, error) {
	return executeRequest(gr, ctx, jsoncmd.KnockRoom, params)
}

func (gr *GomuksRPC) LeaveRoom(ctx context.Context, params *jsoncmd.LeaveRoomParams) (*mautrix.RespLeaveRoom, error) {
	return executeRequest(gr, ctx, jsoncmd.LeaveRoom, params)
}

func (gr *GomuksRPC) CreateRoom(ctx context.Context, params *mautrix.ReqCreateRoom) (*mautrix.RespCreateRoom, error) {
	return executeRequest(gr, ctx, jsoncmd.CreateRoom, params)
}

func (gr *GomuksRPC) MuteRoom(ctx context.Context, params *jsoncmd.MuteRoomParams) (bool, error) {
	return executeRequest(gr, ctx, jsoncmd.MuteRoom, params)
}

func (gr *GomuksRPC) EnsureGroupSessionShared(ctx context.Context, params *jsoncmd.EnsureGroupSessionSharedParams) error {
	return executeRequestNoResponse(gr, ctx, jsoncmd.EnsureGroupSessionShared, params)
}

func (gr *GomuksRPC) SendToDevice(ctx context.Context, params *jsoncmd.SendToDeviceParams) (*mautrix.RespSendToDevice, error) {
	return executeRequest(gr, ctx, jsoncmd.SendToDevice, params)
}

func (gr *GomuksRPC) ResolveAlias(ctx context.Context, params *jsoncmd.ResolveAliasParams) (*mautrix.RespAliasResolve, error) {
	return executeRequest(gr, ctx, jsoncmd.ResolveAlias, params)
}

func (gr *GomuksRPC) RequestOpenIDToken(ctx context.Context) (*mautrix.RespOpenIDToken, error) {
	return executeRequest(gr, ctx, jsoncmd.RequestOpenIDToken, nil)
}

func (gr *GomuksRPC) Logout(ctx context.Context) error {
	return executeRequestNoResponse(gr, ctx, jsoncmd.Logout, nil)
}

func (gr *GomuksRPC) Login(ctx context.Context, params *jsoncmd.LoginParams) error {
	return executeRequestNoResponse(gr, ctx, jsoncmd.Login, params)
}

func (gr *GomuksRPC) LoginCustom(ctx context.Context, params *jsoncmd.LoginCustomParams) error {
	return executeRequestNoResponse(gr, ctx, jsoncmd.LoginCustom, params)
}

func (gr *GomuksRPC) Verify(ctx context.Context, params *jsoncmd.VerifyParams) error {
	return executeRequestNoResponse(gr, ctx, jsoncmd.Verify, params)
}

func (gr *GomuksRPC) DiscoverHomeserver(ctx context.Context, params *jsoncmd.DiscoverHomeserverParams) (*mautrix.ClientWellKnown, error) {
	return executeRequest(gr, ctx, jsoncmd.DiscoverHomeserver, params)
}

func (gr *GomuksRPC) GetLoginFlows(ctx context.Context, params *jsoncmd.GetLoginFlowsParams) (*mautrix.RespLoginFlows, error) {
	return executeRequest(gr, ctx, jsoncmd.GetLoginFlows, params)
}

func (gr *GomuksRPC) RegisterPush(ctx context.Context, params *database.PushRegistration) error {
	return executeRequestNoResponse(gr, ctx, jsoncmd.RegisterPush, params)
}

func (gr *GomuksRPC) ListenToDevice(ctx context.Context, listen bool) (bool, error) {
	return executeRequest(gr, ctx, jsoncmd.ListenToDevice, listen)
}

func (gr *GomuksRPC) GetTurnServers(ctx context.Context) (*mautrix.RespTurnServer, error) {
	return executeRequest(gr, ctx, jsoncmd.GetTurnServers, nil)
}

func (gr *GomuksRPC) GetMediaConfig(ctx context.Context) (*mautrix.RespMediaConfig, error) {
	return executeRequest(gr, ctx, jsoncmd.GetMediaConfig, nil)
}

func (gr *GomuksRPC) CalculateRoomID(ctx context.Context, params *jsoncmd.CalculateRoomIDParams) (id.RoomID, error) {
	return executeRequest(gr, ctx, jsoncmd.CalculateRoomID, params)
}

func (gr *GomuksRPC) RerequestSession(ctx context.Context, params *jsoncmd.RerequestSessionParams) error {
	return executeRequestNoResponse(gr, ctx, jsoncmd.RerequestSession, params)
}
