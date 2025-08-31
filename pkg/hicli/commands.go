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
	"time"

	"go.mau.fi/util/exstrings"
	"go.mau.fi/util/jsontime"
	"go.mau.fi/util/random"
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
	var responseHTML, responseText string
	var retErr error
	switch cmd.Syntax {
	case cmdspec.DiscardSession:
		responseText = h.handleCmdDiscardSession(ctx, roomID)
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

func (h *HiClient) handleCmdDiscardSession(ctx context.Context, roomID id.RoomID) string {
	err := h.CryptoStore.RemoveOutboundGroupSession(ctx, roomID)
	if err != nil {
		return fmt.Sprintf("Failed to remove outbound megolm session: %s", err)
	}
	return "Successfully discarded the outbound megolm session for this room"
}

type rawArguments struct {
	EventType string  `json:"event_type"`
	StateKey  *string `json:"state_key"`
	JSON      string  `json:"json"`
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
		evt, err := h.send(ctx, roomID, event.Type{Type: args.EventType}, jsonData, "", unencrypted, false)
		if err != nil {
			return makeFakeEvent(roomID, fmt.Sprintf("Failed to send event: %s", html.EscapeString(err.Error())))
		}
		return evt
	}
}
