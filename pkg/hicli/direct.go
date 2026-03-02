// Copyright (c) 2026 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

var (
	ErrMDirectMalformed = errors.New("m.direct data is malformed")
	ErrMDirectNoOp      = errors.New("operation is a no-op")
)

func (h *HiClient) ConvertToDM(ctx context.Context, roomID id.RoomID, recipient id.UserID) error {
	h.directChatLock.Lock()
	defer h.directChatLock.Unlock()
	rooms, err := h.unlockedGetDirectChatRooms(ctx)
	if err != nil {
		if errors.Is(err, ErrMDirectMalformed) {
			zerolog.Ctx(ctx).Warn().Err(err).
				Stringer("room_id", roomID).
				Stringer("recipient", recipient).
				Msg("Not converting room to DM because m.direct data is malformed")
			return nil
		}
		return err
	}
	var newMap event.DirectChatsEventContent
	existingVal, ok := rooms[roomID]
	if !ok {
		if recipient == "" {
			// No-op: want to unmark, but it's already not a DM
			return ErrMDirectNoOp
		}
		newMap = maps.Clone(h.directChatUsers)
		//h.dirtyDirectChatRooms[roomID] = recipient
		newMap[recipient] = append(newMap[recipient], roomID)
	} else if recipient == "" {
		newMap = maps.Clone(h.directChatUsers)
		h.unlockedUnmarkDM(roomID, existingVal, newMap)
		//delete(h.directChatRooms, roomID)
	} else if recipient != existingVal {
		newMap = maps.Clone(h.directChatUsers)
		h.unlockedUnmarkDM(roomID, existingVal, newMap)
		//h.dirtyDirectChatRooms[roomID] = recipient
		newMap[recipient] = append(newMap[recipient], roomID)
	} else {
		// No-op: already marked as DM with the same recipient
		return ErrMDirectNoOp
	}
	return h.Client.SetAccountData(ctx, event.AccountDataDirectChats.Type, newMap)
}

func (h *HiClient) GetDMUserID(ctx context.Context, roomID id.RoomID) (id.UserID, error) {
	rooms, err := h.unlockedGetDirectChatRooms(ctx)
	if err != nil {
		return "", err
	}
	return rooms[roomID], nil
}

func (h *HiClient) unlockedUnmarkDM(roomID id.RoomID, currentValue id.UserID, users event.DirectChatsEventContent) {
	if currentValue != "" {
		userRooms := slices.DeleteFunc(users[currentValue], func(r id.RoomID) bool {
			return r == roomID
		})
		if len(userRooms) == 0 {
			delete(users, currentValue)
		} else {
			users[currentValue] = userRooms
		}
	} else {
		for userID, userRooms := range users {
			userRooms = slices.DeleteFunc(userRooms, func(r id.RoomID) bool {
				return r == roomID
			})
			if len(userRooms) == 0 {
				delete(users, userID)
			} else {
				users[userID] = userRooms
			}
		}
	}
}

func directRoomsDiff(old, new map[id.RoomID]id.UserID) (changes map[id.RoomID]id.UserID) {
	changes = make(map[id.RoomID]id.UserID)
	for roomID, oldVal := range old {
		if newVal, ok := new[roomID]; !ok || newVal != oldVal {
			changes[roomID] = newVal
		}
	}
	for roomID, newVal := range new {
		if _, ok := old[roomID]; !ok {
			changes[roomID] = newVal
		}
	}
	return
}

func (h *HiClient) handleSyncDirectChats(ctx context.Context, evt *event.Event, calculateChanges bool) map[id.RoomID]id.UserID {
	err := evt.Content.ParseRaw(evt.Type)
	if err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to parse m.direct event in sync")
		return nil
	}
	val, ok := evt.Content.Parsed.(*event.DirectChatsEventContent)
	if !ok || val == nil {
		zerolog.Ctx(ctx).Warn().
			Type("parsed_type", evt.Content.Parsed).
			Msg("Parsed m.direct event content is not of expected type")
		return nil
	}
	h.directChatLock.Lock()
	defer h.directChatLock.Unlock()
	if !calculateChanges {
		h.directChatMalformed = false
		h.directChatUsers = *val
		return nil
	}
	oldRooms, _ := h.unlockedGetDirectChatRooms(ctx)
	h.directChatMalformed = false
	h.directChatUsers = *val
	h.directChatRooms = nil
	newRooms, err := h.unlockedGetDirectChatRooms(ctx)
	if err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to parse new m.direct event to diff changes")
		return nil
	}
	return directRoomsDiff(oldRooms, newRooms)
}

func (h *HiClient) unlockedGetDirectChatUsers(ctx context.Context) (event.DirectChatsEventContent, error) {
	if h.directChatUsers == nil {
		if h.directChatMalformed {
			return nil, fmt.Errorf("%w (cached error)", ErrMDirectMalformed)
		}
		data, err := h.DB.AccountData.GetGlobal(ctx, h.Account.UserID, event.AccountDataDirectChats)
		if err != nil {
			return nil, err
		}
		h.directChatUsers = make(event.DirectChatsEventContent)
		if data != nil {
			err = json.Unmarshal(data.Content, &h.directChatUsers)
			if err != nil {
				h.directChatMalformed = true
				return nil, fmt.Errorf("%w: %w", ErrMDirectMalformed, err)
			}
		}
	}
	return h.directChatUsers, nil
}

func (h *HiClient) unlockedGetDirectChatRooms(ctx context.Context) (map[id.RoomID]id.UserID, error) {
	if h.directChatRooms == nil {
		users, err := h.unlockedGetDirectChatUsers(ctx)
		if err != nil {
			return nil, err
		}
		out := make(map[id.RoomID]id.UserID, len(users))
		for userID, rooms := range users {
			for _, roomID := range rooms {
				val, exists := out[roomID]
				if exists {
					if val != "" && val != userID {
						out[roomID] = ""
					}
				} else {
					out[roomID] = userID
				}
			}
		}
		h.directChatRooms = out
	}
	return h.directChatRooms, nil
}
