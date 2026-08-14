package hicli

import (
	"context"
	"fmt"
	"iter"
	"time"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

func (h *HiClient) getInitialSyncRoom(ctx context.Context, room *database.Room) *jsoncmd.SyncRoom {
	syncRoom := &jsoncmd.SyncRoom{
		Meta:   room,
		Events: make([]*database.Event, 0, 2),
		State:  map[event.Type]map[string]database.EventRowID{},
	}
	ad, err := h.DB.AccountData.GetAllRoom(ctx, h.Account.UserID, room.ID)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Stringer("room_id", room.ID).Msg("Failed to get room account data")
		if ctx.Err() != nil {
			return nil
		}
	} else {
		syncRoom.AccountData = make(map[event.Type]*database.AccountData, len(ad))
		for _, data := range ad {
			syncRoom.AccountData[event.Type{Type: data.Type, Class: event.AccountDataEventType}] = data
		}
	}
	if room.PreviewEventRowID != 0 {
		previewEvent, err := h.DB.Event.GetByRowID(ctx, room.PreviewEventRowID)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Stringer("room_id", room.ID).Msg("Failed to get preview event for room")
			if ctx.Err() != nil {
				return nil
			}
		} else if previewEvent != nil {
			h.ReprocessExistingEvent(ctx, previewEvent)
			previewMember, err := h.DB.CurrentState.Get(ctx, room.ID, event.StateMember, previewEvent.Sender.String())
			if err != nil {
				zerolog.Ctx(ctx).Err(err).Stringer("room_id", room.ID).Msg("Failed to get preview member event for room")
			} else if previewMember != nil {
				syncRoom.Events = append(syncRoom.Events, previewMember)
				syncRoom.State[event.StateMember] = map[string]database.EventRowID{
					*previewMember.StateKey: previewMember.RowID,
				}
			}
			if previewEvent.LastEditRowID != nil {
				lastEdit, err := h.DB.Event.GetByRowID(ctx, *previewEvent.LastEditRowID)
				if err != nil {
					zerolog.Ctx(ctx).Err(err).Stringer("room_id", room.ID).Msg("Failed to get last edit for preview event")
				} else if lastEdit != nil {
					h.ReprocessExistingEvent(ctx, lastEdit)
					syncRoom.Events = append(syncRoom.Events, lastEdit)
				}
			}
			syncRoom.Events = append(syncRoom.Events, previewEvent)
		}
	}
	return syncRoom
}

func (h *HiClient) getInitialSyncAccountData(ctx context.Context) (first, last map[event.Type]*database.AccountData) {
	ad, err := h.DB.AccountData.GetAllGlobal(ctx, h.Account.UserID)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("Failed to get global account data")
		return
	}
	first = make(map[event.Type]*database.AccountData, len(ad))
	last = make(map[event.Type]*database.AccountData, 1)
	for _, data := range ad {
		switch data.Type {
		case "im.ponies.emote_rooms", "m.image_pack.rooms":
			last[event.Type{Type: data.Type, Class: event.AccountDataEventType}] = data
		default:
			first[event.Type{Type: data.Type, Class: event.AccountDataEventType}] = data
		}
	}
	return
}

func (h *HiClient) GetCatchupSync(ctx context.Context, since int64) (*jsoncmd.SyncComplete, error) {
	serverTS := time.Now().UnixMilli()

	spaces, err := h.DB.Room.GetAllSpaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get spaces: %w", err)
	}
	ad, err := h.DB.AccountData.GetAllGlobalSince(ctx, h.Account.UserID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get global account data: %w", err)
	}
	payload := &jsoncmd.SyncComplete{
		Rooms:           make(map[id.RoomID]*jsoncmd.SyncRoom, len(spaces)),
		AccountData:     make(map[event.Type]*database.AccountData, len(ad)),
		ServerTimestamp: serverTS,
		Catchup:         true,
	}
	for _, item := range ad {
		payload.AccountData[event.Type{Type: item.Type, Class: event.AccountDataEventType}] = item
	}
	payload.InvitedRooms, err = h.DB.InvitedRoom.GetAllSince(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get invited rooms: %w", err)
	}
	payload.LeftRooms, err = h.DB.Room.GetLeftRoomsSince(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get left rooms: %w", err)
	}
	roomAD, err := h.DB.AccountData.GetAllRoomSince(ctx, h.Account.UserID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get room account data: %w", err)
	}
	roomADByRoom := make(map[id.RoomID]map[event.Type]*database.AccountData)
	for _, item := range roomAD {
		if _, ok := roomADByRoom[item.RoomID]; !ok {
			roomADByRoom[item.RoomID] = make(map[event.Type]*database.AccountData)
		}
		roomADByRoom[item.RoomID][event.Type{Type: item.Type, Class: event.AccountDataEventType}] = item
	}
	rooms, err := h.DB.Room.GetAllChangedSince(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get rooms changed since %d: %w", since, err)
	}
	spaceEdges, err := h.DB.SpaceEdge.GetAll(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get space edges: %w", err)
	}
	payload.SpaceEdges = make(map[id.RoomID][]*database.SpaceEdge, 0)
	for _, room := range rooms {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		payload.Rooms[room.ID] = h.getInitialSyncRoom(ctx, room)
		if room.GetType() == event.RoomTypeSpace {
			edges, ok := spaceEdges[room.ID]
			if !ok {
				edges = []*database.SpaceEdge{}
			}
			payload.SpaceEdges[room.ID] = edges
		}
	}
	if len(payload.SpaceEdges) > 0 || len(payload.LeftRooms) > 0 {
		payload.TopLevelSpaces, err = h.DB.SpaceEdge.GetTopLevelIDs(ctx, h.Account.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get top-level spaces: %w", err)
		}
	}
	for roomID, adRoom := range roomADByRoom {
		_, exists := payload.Rooms[roomID]
		if !exists {
			payload.Rooms[roomID] = &jsoncmd.SyncRoom{
				AccountData: adRoom,
			}
		}
	}
	return payload, nil
}

func (h *HiClient) GetInitialSync(ctx context.Context, batchSize int, catchupSince int64) iter.Seq[*jsoncmd.SyncComplete] {
	return func(yield func(*jsoncmd.SyncComplete) bool) {
		if catchupSince > 0 {
			payload, err := h.GetCatchupSync(ctx, catchupSince)
			if err == nil {
				yield(payload)
			} else if ctx.Err() == nil {
				zerolog.Ctx(ctx).Err(err).Msg("Failed to generate catchup sync")
			}
			return
		}
		serverTS := time.Now().UnixMilli()
		maxTS := time.Now().Add(1 * time.Hour)
		firstAccountData, lastAccountData := h.getInitialSyncAccountData(ctx)
		{
			spaces, err := h.DB.Room.GetAllSpaces(ctx)
			if err != nil {
				if ctx.Err() == nil {
					zerolog.Ctx(ctx).Err(err).Msg("Failed to get initial spaces to send to client")
				}
				return
			}
			payload := jsoncmd.SyncComplete{
				Rooms:           make(map[id.RoomID]*jsoncmd.SyncRoom, len(spaces)),
				AccountData:     firstAccountData,
				ServerTimestamp: serverTS,
			}
			payload.TopLevelSpaces, err = h.DB.SpaceEdge.GetTopLevelIDs(ctx, h.Account.UserID)
			if err != nil {
				if ctx.Err() == nil {
					zerolog.Ctx(ctx).Err(err).Msg("Failed to get top-level space IDs to send to client")
				}
				return
			}
			payload.SpaceEdges, err = h.DB.SpaceEdge.GetAll(ctx, "")
			if err != nil {
				if ctx.Err() == nil {
					zerolog.Ctx(ctx).Err(err).Msg("Failed to get space edges to send to client")
				}
				return
			}
			for _, room := range spaces {
				payload.Rooms[room.ID] = h.getInitialSyncRoom(ctx, room)
				if ctx.Err() != nil {
					return
				}
				_, hasEdges := payload.SpaceEdges[room.ID]
				if !hasEdges {
					payload.SpaceEdges[room.ID] = []*database.SpaceEdge{}
				}
			}
			payload.InvitedRooms, err = h.DB.InvitedRoom.GetAll(ctx)
			if err != nil {
				if ctx.Err() == nil {
					zerolog.Ctx(ctx).Err(err).Msg("Failed to get invited rooms to send to client")
				}
				return
			}
			payload.ClearState = true
			if !yield(&payload) {
				return
			}
		}
		for i := 0; ; i++ {
			rooms, err := h.DB.Room.GetBySortTS(ctx, maxTS, batchSize)
			if err != nil {
				if ctx.Err() == nil {
					zerolog.Ctx(ctx).Err(err).Msg("Failed to get initial rooms to send to client")
				}
				return
			}
			payload := jsoncmd.SyncComplete{
				Rooms: make(map[id.RoomID]*jsoncmd.SyncRoom, len(rooms)),

				ServerTimestamp: serverTS,
			}
			for roomIdx, room := range rooms {
				if len(rooms) >= batchSize && room.SortingTimestamp == rooms[len(rooms)-1].SortingTimestamp {
					if roomIdx == 0 {
						batchSize *= 2
					}
					break
				}
				maxTS = room.SortingTimestamp.Time
				payload.Rooms[room.ID] = h.getInitialSyncRoom(ctx, room)
				if ctx.Err() != nil {
					return
				}
			}
			if !yield(&payload) {
				return
			} else if len(rooms) < batchSize {
				break
			}
		}
		// This is last so that the frontend would know about all rooms before trying to fetch custom emoji packs
		yield(&jsoncmd.SyncComplete{AccountData: lastAccountData, ServerTimestamp: serverTS})
	}
}
