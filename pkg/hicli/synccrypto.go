// Copyright (c) 2026 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"context"
	"encoding/json"
	"fmt"

	"go.mau.fi/util/exerrors"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

func makeEmptyChan() chan struct{} {
	return make(chan struct{})
}

func (h *HiClient) WaitForEventDecryption(evtID id.EventID) (chan struct{}, func()) {
	return h.eventDecryptionWaiters.GetOrSetFactory(evtID, makeEmptyChan), func() {
		ch, ok := h.eventDecryptionWaiters.Pop(evtID)
		if ok {
			close(ch)
		}
	}
}

func (h *HiClient) SyncToDeviceQueue(ctx context.Context) error {
	if h.stopping {
		return fmt.Errorf("client is stopping")
	} else if h.IsSyncing() {
		return nil
	}
	since := h.Account.NextBatch
	if since == "" {
		return fmt.Errorf("no next batch token available")
	}
	everything := []event.Type{{Type: "*"}}
	filter := exerrors.Must(json.Marshal(&mautrix.Filter{
		Presence:    &mautrix.FilterPart{NotTypes: everything},
		AccountData: &mautrix.FilterPart{NotTypes: everything},
		Room: &mautrix.RoomFilter{
			NotRooms: []id.RoomID{"*"},
		},
	}))
	hasMore := true
	for hasMore {
		if h.stopping {
			return fmt.Errorf("client is stopping")
		} else if h.IsSyncing() {
			return nil
		}
		resp, err := h.Client.FullSyncRequest(ctx, mautrix.ReqSync{
			Timeout:     0,
			Since:       since,
			FilterID:    string(filter),
			SetPresence: event.PresenceOffline,
		})
		if err != nil {
			return err
		}
		hasMore = len(resp.ToDevice.Events) > 0
		since = resp.NextBatch
		h.preProcessSyncResponse(ctx, resp)
		go h.asyncPostProcessSyncResponse(ctx, resp)
	}
	h.backgroundMegolmDecrypters.Wait()
	return nil
}
