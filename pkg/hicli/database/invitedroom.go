// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package database

import (
	"context"

	"go.mau.fi/util/dbutil"
	"go.mau.fi/util/jsontime"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

const (
	getInvitedRoomsQuery = `
		SELECT room_id, received_at, invite_state
		FROM invited_room
		ORDER BY received_at DESC
	`
	getInvitedRoomQuery = `
		SELECT room_id, received_at, invite_state
		FROM invited_room
		WHERE room_id = $1
	`
	deleteInvitedRoomQuery = `
		DELETE FROM invited_room WHERE room_id = $1
	`
	upsertInvitedRoomQuery = `
		INSERT INTO invited_room (room_id, received_at, invite_state)
		VALUES ($1, $2, $3)
		ON CONFLICT (room_id) DO UPDATE
			SET received_at = $2, invite_state = $3
	`
)

type InvitedRoomQuery struct {
	*dbutil.QueryHelper[*InvitedRoom]
}

func (irq *InvitedRoomQuery) GetAll(ctx context.Context) ([]*InvitedRoom, error) {
	return irq.QueryMany(ctx, getInvitedRoomsQuery)
}

func (irq *InvitedRoomQuery) Get(ctx context.Context, roomID id.RoomID) (*InvitedRoom, error) {
	return irq.QueryOne(ctx, getInvitedRoomQuery, roomID)
}

func (irq *InvitedRoomQuery) Upsert(ctx context.Context, room *InvitedRoom) error {
	return irq.Exec(ctx, upsertInvitedRoomQuery, room.sqlVariables()...)
}

func (irq *InvitedRoomQuery) Delete(ctx context.Context, roomID id.RoomID) error {
	return irq.Exec(ctx, deleteInvitedRoomQuery, roomID)
}

type InvitedRoom struct {
	// The ID of the invited room.
	ID id.RoomID `json:"room_id"`
	// Either the timestamp of the invite event, or the time when the invite was received.
	CreatedAt jsontime.UnixMilli `json:"created_at"`
	// The (untrusted) room metadata state events for the room.
	InviteState []*event.Event `json:"invite_state"`
}

func (r *InvitedRoom) sqlVariables() []any {
	return []any{
		r.ID,
		dbutil.UnixMilliPtr(r.CreatedAt.Time),
		dbutil.JSON{Data: &r.InviteState},
	}
}

func (r *InvitedRoom) Scan(row dbutil.Scannable) (*InvitedRoom, error) {
	var createdAt int64
	err := row.Scan(&r.ID, &createdAt, dbutil.JSON{Data: &r.InviteState})
	if err != nil {
		return nil, err
	}
	r.CreatedAt = jsontime.UMInt(createdAt)
	return r, nil
}

func (r *InvitedRoom) GetDMUserID(self id.UserID) (target id.UserID) {
	if r == nil {
		return
	}
	for _, ev := range r.InviteState {
		switch ev.Type {
		case event.StateMember:
			userID := id.UserID(ev.GetStateKey())
			if userID == self {
				isDirect, _ := ev.Content.Raw["is_direct"].(bool)
				if !isDirect {
					return ""
				}
			} else if target != "" {
				return ""
			} else {
				target = userID
			}
		case event.StateJoinRules:
			if ev.Content.Raw["join_rule"] != string(event.JoinRuleInvite) {
				return ""
			}
		case event.StateRoomName, event.StateTopic, event.StateRoomAvatar, event.StateCanonicalAlias:
			return ""
		}
	}
	return
}
