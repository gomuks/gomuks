// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package client

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"slices"
	"strconv"
	"sync"

	"go.mau.fi/util/exsync"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
	"go.mau.fi/gomuks/pkg/rpc"
	"go.mau.fi/gomuks/pkg/rpc/store"
)

type GomuksClient struct {
	jsoncmd.GomuksAPI
	*store.GomuksStore

	InitComplete *exsync.Event
	EventHandler rpc.EventHandler

	SendNotification func(room *store.RoomStore, notif jsoncmd.SyncNotification)

	stateRequestQueue     []database.RoomStateGUID
	stateRequestQueueLock sync.Mutex
}

func NewGomuksClient(baseURL string) (*GomuksClient, error) {
	rpcClient, err := rpc.NewGomuksRPC(baseURL)
	if err != nil {
		return nil, err
	}
	gc := &GomuksClient{
		GomuksAPI:    rpcClient,
		GomuksStore:  store.NewStore(),
		InitComplete: exsync.NewEvent(),
	}
	rpcClient.EventHandler = gc.handleEvent
	return gc, nil
}

func (gc *GomuksClient) handleEvent(ctx context.Context, rawEvt any) {
	switch evt := rawEvt.(type) {
	case *jsoncmd.ClientState:
		gc.GomuksStore.ClientState = *evt
	case *jsoncmd.SyncStatus:
		// TODO
	case *jsoncmd.InitComplete:
		gc.InitComplete.Set()
	case *jsoncmd.SyncComplete:
		gc.GomuksStore.ApplySync(evt)
		for _, room := range evt.Rooms {
			if len(room.Notifications) == 0 {
				continue
			}
			roomStore := gc.GomuksStore.GetRoom(room.Meta.ID)
			for _, notif := range room.Notifications {
				notif.Room = roomStore.Meta.Current()
				notif.Event = roomStore.GetEventByRowID(notif.RowID)
				if notif.Event == nil {
					continue
				}
				gc.SendNotification(roomStore, notif)
			}
		}
	case *jsoncmd.EventsDecrypted:
		callRoomMethod(gc, evt.RoomID, (*store.RoomStore).ApplyDecrypted, evt)
	case *jsoncmd.SendComplete:
		callRoomMethod(gc, evt.Event.RoomID, (*store.RoomStore).ApplySendComplete, evt.Event)
	case *jsoncmd.ImageAuthToken:
		gc.GomuksStore.ImageAuthToken = string(*evt)
	case *jsoncmd.Typing:
		callRoomMethod(gc, evt.RoomID, (*store.RoomStore).ApplyTyping, evt.UserIDs)
	}
	if gc.EventHandler != nil {
		gc.EventHandler(ctx, rawEvt)
	}
}

func callRoomMethod[T any](gc *GomuksClient, roomID id.RoomID, fn func(room *store.RoomStore, val T), val T) {
	room := gc.GomuksStore.GetRoom(roomID)
	if room == nil {
		return
	}
	fn(room, val)
}

func (gc *GomuksClient) RequestEvent(ctx context.Context, room *store.RoomStore, eventID id.EventID) {

}

func (gc *GomuksClient) SendMessage(ctx context.Context, params *jsoncmd.SendMessageParams) error {
	room := gc.GomuksStore.GetRoom(params.RoomID)
	if room == nil {
		return fmt.Errorf("room not found in store")
	}
	dbEvt, err := gc.GomuksAPI.SendMessage(ctx, params)
	if err != nil {
		return err
	} else if dbEvt != nil {
		room.ApplyPending(dbEvt)
	}
	return nil
}

func (gc *GomuksClient) QueueRoomStateRequest(key database.RoomStateGUID) {
	gc.stateRequestQueueLock.Lock()
	defer gc.stateRequestQueueLock.Unlock()
	gc.stateRequestQueue = append(gc.stateRequestQueue, key)
}

func (gc *GomuksClient) FlushRoomStateRequests(ctx context.Context) error {
	gc.stateRequestQueueLock.Lock()
	keys := gc.stateRequestQueue
	gc.stateRequestQueue = nil
	gc.stateRequestQueueLock.Unlock()
	if len(keys) == 0 {
		return nil
	}
	return gc.LoadSpecificRoomState(ctx, keys)
}

func (gc *GomuksClient) LoadSpecificRoomState(ctx context.Context, keys []database.RoomStateGUID) error {
	keys = slices.DeleteFunc(keys, func(guid database.RoomStateGUID) bool {
		room := gc.GomuksStore.GetRoom(guid.RoomID)
		return room == nil || room.GetStateEvent(guid.Type, guid.StateKey) != nil
	})
	if len(keys) == 0 {
		return nil
	}
	resp, err := gc.GomuksAPI.GetSpecificRoomState(ctx, &jsoncmd.GetSpecificRoomStateParams{Keys: keys})
	if err != nil {
		return err
	}
	for _, evt := range resp {
		room := gc.GomuksStore.GetRoom(evt.RoomID)
		if room == nil {
			continue
		}
		room.ApplyState(evt)
	}
	return nil
}

func (gc *GomuksClient) LoadRoomState(ctx context.Context, roomID id.RoomID, includeMembers, refetch bool) error {
	room := gc.GomuksStore.GetRoom(roomID)
	if room == nil {
		return fmt.Errorf("room not found in store")
	}
	room.StateLoadLock.Lock()
	defer room.StateLoadLock.Unlock()
	if !refetch && (room.FullMembersLoaded.Load() || (!includeMembers && room.StateLoaded.Load())) {
		return nil
	}
	resp, err := gc.GomuksAPI.GetRoomState(ctx, &jsoncmd.GetRoomStateParams{
		RoomID:         roomID,
		Refetch:        refetch,
		FetchMembers:   !room.Meta.Current().HasMemberList,
		IncludeMembers: includeMembers,
	})
	if err != nil {
		return err
	}
	room.Meta.Current().HasMemberList = true
	room.ApplyFullState(resp, !includeMembers)
	return nil
}

func (gc *GomuksClient) LoadMoreHistory(ctx context.Context, roomID id.RoomID) error {
	room := gc.GomuksStore.GetRoom(roomID)
	if room == nil {
		return fmt.Errorf("room not found in store")
	} else if !room.Paginating.CompareAndSwap(false, true) {
		return fmt.Errorf("already paginating room")
	}
	defer room.Paginating.Store(false)
	oldestRowID, count := room.GetPaginationParams()
	resp, err := gc.GomuksAPI.Paginate(ctx, &jsoncmd.PaginateParams{
		RoomID:        room.ID,
		MaxTimelineID: oldestRowID,
		Limit:         count,
		Reset:         false,
	})
	if err != nil {
		return err
	}
	room.ApplyPagination(resp)
	return nil
}

func (gc *GomuksClient) GetDownloadURL(mxc id.ContentURI, encrypted, preauthed bool) string {
	query := url.Values{
		"encrypted": {strconv.FormatBool(encrypted)},
	}
	if preauthed {
		query.Set("image_auth", gc.GomuksStore.ImageAuthToken)
	}
	// TODO fix downloads with non-remote backend
	return gc.GomuksAPI.(*rpc.GomuksRPC).BuildURLWithQuery(rpc.GomuksURLPath{"media", mxc.Homeserver, mxc.FileID}, query)
}

func (gc *GomuksClient) Download(mxc id.ContentURI, encrypted bool) ([]byte, error) {
	// TODO fix downloads with non-remote backend
	resp, err := gc.GomuksAPI.(*rpc.GomuksRPC).DownloadMedia(context.TODO(), rpc.DownloadMediaParams{
		MXC:       mxc,
		Encrypted: encrypted,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
