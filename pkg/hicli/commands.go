// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.mau.fi/util/exstrings"
	"go.mau.fi/util/jsontime"
	"go.mau.fi/util/random"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/cmdspec"
	"go.mau.fi/gomuks/pkg/hicli/database"
)

const FakeGomuksSender id.UserID = "@gomuks"

func makeFakeEvent(roomID id.RoomID, html string) *database.Event {
	return &database.Event{
		RowID:         -database.EventRowID(time.Now().UnixMilli()),
		TimelineRowID: 0,
		RoomID:        roomID,
		ID:            id.EventID("$gomuks-internal-" + random.String(10)),
		Sender:        FakeGomuksSender,
		Type:          event.EventMessage.Type,
		Timestamp:     jsontime.UnixMilliNow(),
		Content:       json.RawMessage(`{"msgtype":"m.text"}`),
		Unsigned:      json.RawMessage("{}"),
		LocalContent: &database.LocalContent{
			SanitizedHTML: html,
		},
	}
}

func (h *HiClient) ProcessCommand(
	ctx context.Context,
	roomID id.RoomID,
	cmd *event.BotCommandInput,
	relatesTo *event.RelatesTo,
) (*database.Event, error) {
	ctx = mautrix.WithMaxRetries(ctx, 0)
	var responseHTML, responseText string
	var retErr error
	switch cmd.Syntax {
	case cmdspec.DiscardSession:
		responseText = h.handleCmdDiscardSession(ctx, roomID)
	case cmdspec.Meow:
		responseText = "Meow " + gjson.GetBytes(cmd.Arguments, "meow").Str
	case cmdspec.Invite:
		responseText, retErr = callWithParsedArgs(ctx, roomID, cmd.Arguments, relatesTo, h.handleCmdInvite)
	case cmdspec.Kick:
		responseText, retErr = callWithParsedArgs(ctx, roomID, cmd.Arguments, relatesTo, h.handleCmdKick)
	case cmdspec.Ban:
		responseText, retErr = callWithParsedArgs(ctx, roomID, cmd.Arguments, relatesTo, h.handleCmdBan)
	case cmdspec.Join:
		responseText, retErr = callWithParsedArgs(ctx, roomID, cmd.Arguments, relatesTo, h.handleCmdJoin)
	case cmdspec.Leave:
		responseText = h.handleCmdLeave(ctx, roomID)
	case cmdspec.MyRoomNick:
		responseText, retErr = callWithParsedArgs(ctx, roomID, cmd.Arguments, relatesTo, h.handleCmdMyRoomNick)
	case cmdspec.GlobalNick:
		responseText, retErr = callWithParsedArgs(ctx, roomID, cmd.Arguments, relatesTo, h.handleCmdGlobalNick)
	case cmdspec.Redact:
		responseText, retErr = callWithParsedArgs(ctx, roomID, cmd.Arguments, relatesTo, h.handleCmdRedact)
	case cmdspec.Raw:
		return callWithParsedArgs(ctx, roomID, cmd.Arguments, relatesTo, h.handleCmdRaw)
	case cmdspec.UnencryptedRaw:
		return callWithParsedArgs(ctx, roomID, cmd.Arguments, relatesTo, h.handleCmdUnencryptedRaw)
	case cmdspec.RawState:
		return callWithParsedArgs(ctx, roomID, cmd.Arguments, relatesTo, h.handleCmdRaw)
	default:
		responseHTML = fmt.Sprintf("Unknown command <code>%s</code>", html.EscapeString(cmd.Syntax))
	}
	if responseText != "" {
		responseHTML = html.EscapeString(responseText)
	}
	if retErr != nil {
		return nil, retErr
	} else if responseHTML == "" {
		return nil, nil
	}
	return makeFakeEvent(roomID, responseHTML), nil
}

func callWithParsedArgs[T, R any](
	ctx context.Context,
	roomID id.RoomID,
	args json.RawMessage,
	relatesTo *event.RelatesTo,
	fn func(context.Context, id.RoomID, T, *event.RelatesTo) R,
) (R, error) {
	var parsedArgs T
	err := json.Unmarshal(args, &parsedArgs)
	if err != nil {
		var zero R
		return zero, err
	}
	return fn(ctx, roomID, parsedArgs, relatesTo), nil
}

func (h *HiClient) handleCmdDiscardSession(ctx context.Context, roomID id.RoomID) string {
	err := h.CryptoStore.RemoveOutboundGroupSession(ctx, roomID)
	if err != nil {
		return fmt.Sprintf("Failed to remove outbound megolm session: %s", err)
	}
	return "Successfully discarded the outbound megolm session for this room"
}

type inviteArgs struct {
	UserID id.UserID `json:"user_id"`
	Reason string    `json:"reason"`
}

func (h *HiClient) handleCmdInvite(ctx context.Context, roomID id.RoomID, args inviteArgs, _ *event.RelatesTo) string {
	_, err := h.Client.InviteUser(ctx, roomID, &mautrix.ReqInviteUser{
		Reason: args.Reason,
		UserID: args.UserID,
	})
	if err != nil {
		return fmt.Sprintf("Failed to send invite: %v", err)
	}
	return ""
}

func (h *HiClient) handleCmdKick(ctx context.Context, roomID id.RoomID, args inviteArgs, _ *event.RelatesTo) string {
	_, err := h.Client.KickUser(ctx, roomID, &mautrix.ReqKickUser{
		Reason: args.Reason,
		UserID: args.UserID,
	})
	if err != nil {
		return fmt.Sprintf("Failed to kick user: %v", err)
	}
	return ""
}

func (h *HiClient) handleCmdBan(ctx context.Context, roomID id.RoomID, args inviteArgs, _ *event.RelatesTo) string {
	_, err := h.Client.BanUser(ctx, roomID, &mautrix.ReqBanUser{
		Reason: args.Reason,
		UserID: args.UserID,
	})
	if err != nil {
		return fmt.Sprintf("Failed to ban user: %v", err)
	}
	return ""
}

type joinArgs struct {
	RoomReference string `json:"room_reference"`
	Reason        string `json:"reason"`
}

func (h *HiClient) handleCmdJoin(ctx context.Context, _ id.RoomID, args joinArgs, _ *event.RelatesTo) string {
	roomRef := args.RoomReference
	req := &mautrix.ReqJoinRoom{
		Reason: args.Reason,
	}
	if url, _ := id.ParseMatrixURIOrMatrixToURL(roomRef); url != nil {
		roomRef = url.PrimaryIdentifier()
		req.Via = url.Via
	}
	if len(roomRef) == 0 || (roomRef[0] != '!' && roomRef[0] != '#') {
		return "Input is not a room ID or alias"
	}
	_, err := h.Client.JoinRoom(ctx, args.RoomReference, req)
	if err != nil {
		return fmt.Sprintf("Failed to join room: %v", err)
	}
	return ""
}

func (h *HiClient) handleCmdLeave(ctx context.Context, roomID id.RoomID) string {
	_, err := h.Client.LeaveRoom(ctx, roomID)
	if err != nil {
		return fmt.Sprintf("Failed to leave room: %v", err)
	}
	return ""
}

type myRoomNickParams struct {
	Name string `json:"name"`
}

func (h *HiClient) handleCmdMyRoomNick(ctx context.Context, roomID id.RoomID, params myRoomNickParams, _ *event.RelatesTo) string {
	if evt, err := h.DB.CurrentState.Get(ctx, roomID, event.StateMember, h.Account.UserID.String()); err != nil {
		return fmt.Sprintf("Failed to get current member event: %v", err)
	} else if evt == nil {
		return "No member event found for self in this room"
	} else if content, err := sjson.SetBytes(evt.Content, "displayname", params.Name); err != nil {
		return fmt.Sprintf("Failed to mutate member event content: %v", err)
	} else if _, err = h.Client.SendStateEvent(ctx, roomID, event.StateMember, h.Account.UserID.String(), json.RawMessage(content)); err != nil {
		return fmt.Sprintf("Failed to update member event: %v", err)
	}
	return ""
}

func (h *HiClient) handleCmdGlobalNick(ctx context.Context, _ id.RoomID, params myRoomNickParams, _ *event.RelatesTo) string {
	if err := h.Client.SetProfileField(ctx, "displayname", params.Name); err != nil {
		return fmt.Sprintf("Failed to set display name: %v", err)
	}
	return ""
}

type redactParams struct {
	EventID id.EventID `json:"event_id"`
	Reason  string     `json:"reason"`
}

func (h *HiClient) handleCmdRedact(ctx context.Context, roomID id.RoomID, params redactParams, _ *event.RelatesTo) string {
	if !strings.HasPrefix(string(params.EventID), "$") {
		url, err := id.ParseMatrixURIOrMatrixToURL(string(params.EventID))
		if err != nil {
			return "Input is not a valid event ID or URL"
		}
		roomID = url.RoomID()
		params.EventID = url.EventID()
		if params.EventID == "" {
			return "Input is not a valid event ID or event URL"
		}
	}
	_, err := h.Client.RedactEvent(ctx, roomID, params.EventID, mautrix.ReqRedact{
		Reason: params.Reason,
	})
	if err != nil {
		return fmt.Sprintf("Failed to redact event: %v", err)
	}
	return ""
}

type rawArguments struct {
	EventType string  `json:"event_type"`
	StateKey  *string `json:"state_key"`
	JSON      string  `json:"json"`
}

func (h *HiClient) handleCmdRaw(ctx context.Context, roomID id.RoomID, args rawArguments, _ *event.RelatesTo) *database.Event {
	return h.handleCmdRawInternal(ctx, roomID, args, false)
}

func (h *HiClient) handleCmdUnencryptedRaw(ctx context.Context, roomID id.RoomID, args rawArguments, _ *event.RelatesTo) *database.Event {
	return h.handleCmdRawInternal(ctx, roomID, args, true)
}

func (h *HiClient) handleCmdRawInternal(ctx context.Context, roomID id.RoomID, args rawArguments, unencrypted bool) *database.Event {
	jsonData := json.RawMessage(exstrings.UnsafeBytes(args.JSON))
	if !json.Valid(jsonData) {
		return makeFakeEvent(roomID, "Invalid JSON entered")
	}
	if args.StateKey != nil {
		_, err := h.SetState(ctx, roomID, event.Type{Type: args.EventType, Class: event.StateEventType}, *args.StateKey, jsonData)
		if err != nil {
			return makeFakeEvent(roomID, fmt.Sprintf("Failed to send state event: %s", html.EscapeString(err.Error())))
		}
		return nil
	} else {
		evt, err := h.send(ctx, roomID, event.Type{Type: args.EventType}, jsonData, "", unencrypted, false, 0)
		if err != nil {
			return makeFakeEvent(roomID, fmt.Sprintf("Failed to send event: %s", html.EscapeString(err.Error())))
		}
		return evt
	}
}
