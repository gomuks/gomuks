// gomuks - A Matrix client written in Go.
// Copyright (C) 2026 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package gomuks

import (
	"context"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
)

type PushPayload struct {
	mautrix.ReqAckWebPusher

	EventID id.EventID `json:"event_id"`
	RoomID  id.RoomID  `json:"room_id"`
}

func (gmx *Gomuks) ReceivePushNotification(ctx context.Context, payload PushPayload) (*database.Event, error) {
	if err := gmx.initClientForNotifications(ctx); err != nil {
		return nil, err
	}
	if payload.AckToken != "" && payload.AppID != "" {
		err := gmx.Client.Client.AckWebPusher(ctx, &payload.ReqAckWebPusher)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("Failed to activate web push")
		} else {
			zerolog.Ctx(ctx).Debug().Msg("Successfully activated web push")
		}
	}
	if payload.EventID != "" && payload.RoomID != "" {
		return gmx.handlePushForEvent(ctx, payload.RoomID, payload.EventID)
	}
	return nil, nil
}

func (gmx *Gomuks) handlePushForEvent(ctx context.Context, roomID id.RoomID, eventID id.EventID) (*database.Event, error) {
	evt, err := gmx.Client.GetEvent(ctx, roomID, eventID)
	if evt != nil && strings.Contains(evt.DecryptionError, crypto.ErrNoSessionFound.Error()) {
		return gmx.handlePushForEventWithSync(ctx, roomID, eventID)
	}
	return evt, err
}

func (gmx *Gomuks) handlePushForEventWithSync(ctx context.Context, roomID id.RoomID, eventID id.EventID) (*database.Event, error) {
	gmx.notificationSyncLock.Lock()
	defer gmx.notificationSyncLock.Unlock()
	ch, removeWaiter := gmx.Client.WaitForEventDecryption(eventID)
	defer removeWaiter()
	evt, err := gmx.Client.GetEvent(ctx, roomID, eventID)
	if evt != nil && strings.Contains(evt.DecryptionError, crypto.ErrNoSessionFound.Error()) {
		start := time.Now()
		err = gmx.Client.SyncToDeviceQueue(ctx)
		if err != nil {
			return nil, err
		}
		select {
		case <-ch:
		case <-time.After(min(5*time.Second-time.Since(start), 1*time.Second)):
		}
		evt, err = gmx.Client.GetEvent(ctx, roomID, eventID)
	}
	return evt, err
}
