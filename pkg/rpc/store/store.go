// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package store

import (
	"encoding/json"
	"slices"
	"strings"
	"sync"
	"time"

	"go.mau.fi/util/ptr"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

type RoomListEntry struct {
	RoomID           id.RoomID
	DMUserID         id.UserID
	SortingTimestamp time.Time
	PreviewEvent     *database.Event
	PreviewSender    *database.Event
	Name             string
	SearchName       string
	Avatar           id.ContentURI
	MarkedUnread     bool
	IsInvite         bool
	database.UnreadCounts
}

type GomuksStore struct {
	jsoncmd.ClientState
	ImageAuthToken string

	lock             sync.RWMutex
	invitedRooms     map[id.RoomID]*InvitedRoom
	rooms            map[id.RoomID]*RoomStore
	roomList         []*RoomListEntry
	ReversedRoomList EventDispatcher[[]*RoomListEntry]
	accountData      map[event.Type]*database.AccountData
	AccountDataSubs  MultiNotifier[event.Type]
	PreferenceCache  EventDispatcher[*Preferences]
	spaceEdges       map[id.RoomID][]*database.SpaceEdge
	SpaceList        EventDispatcher[[]*SpaceEntry]
}

// SpaceEntry is a space shown in the space switcher.
type SpaceEntry struct {
	RoomID id.RoomID
	Name   string
}

func NewStore() *GomuksStore {
	gs := &GomuksStore{
		rooms:        make(map[id.RoomID]*RoomStore),
		invitedRooms: make(map[id.RoomID]*InvitedRoom),
		accountData:  make(map[event.Type]*database.AccountData),
		spaceEdges:   make(map[id.RoomID][]*database.SpaceEdge),
	}
	return gs
}

func roomListEntryChanged(entry *jsoncmd.SyncRoom, oldMeta *database.Room) bool {
	return entry.Meta.SortingTimestamp != oldMeta.SortingTimestamp ||
		entry.Meta.UnreadCounts != oldMeta.UnreadCounts ||
		entry.Meta.MarkedUnread != oldMeta.MarkedUnread ||
		entry.Meta.PreviewEventRowID != oldMeta.PreviewEventRowID ||
		ptr.Val(entry.Meta.Name) != ptr.Val(oldMeta.Name) ||
		ptr.Val(entry.Meta.Avatar) != ptr.Val(oldMeta.Avatar) ||
		slices.ContainsFunc(entry.Timeline, func(tuple database.TimelineRowTuple) bool {
			return tuple.Event == entry.Meta.PreviewEventRowID
		})
}

func (gs *GomuksStore) shouldHideRoom(entry *database.Room) bool {
	switch entry.CreationContent.Type {
	default:
		// The room is not a normal room
		return true
	case "":
	case "support.feline.policy.lists.msc.v1":
	case "org.matrix.msc3417.call":
	}
	if entry.Tombstone.GetReplacementRoom() != "" {
		replStore, ok := gs.rooms[entry.Tombstone.ReplacementRoom]
		if ok && replStore.Meta.Current().CreationContent.GetPredecessor().RoomID == entry.ID {
			// The room is tombstoned and the replacement room is valid.
			return true
		}
	}
	// Otherwise don't hide the room.
	return false
}

func (gs *GomuksStore) makeRoomListEntry(roomStore *RoomStore) *RoomListEntry {
	meta := roomStore.Meta.Current()
	roomStore.Hidden = gs.shouldHideRoom(meta)
	if roomStore.Hidden {
		return nil
	}
	name := ptr.Val(meta.Name)
	if name == "" {
		name = "Unnamed room"
	}
	entry := &RoomListEntry{
		RoomID:           roomStore.ID,
		DMUserID:         ptr.Val(meta.DMUserID),
		SortingTimestamp: meta.SortingTimestamp.Time,
		PreviewEvent:     roomStore.GetEventByRowID(meta.PreviewEventRowID),
		Name:             name,
		SearchName:       toSearchableString(name),
		Avatar:           ptr.Val(meta.Avatar),
		MarkedUnread:     ptr.Val(meta.MarkedUnread),
		UnreadCounts:     meta.UnreadCounts,
	}
	if entry.PreviewEvent != nil {
		entry.PreviewSender = roomStore.GetStateEvent(event.StateMember, entry.PreviewEvent.Sender.String())
	}
	return entry
}

func (gs *GomuksStore) ApplySync(sync *jsoncmd.SyncComplete) {
	gs.lock.Lock()
	defer gs.lock.Unlock()
	resyncRoomList := len(gs.roomList) == 0
	changedRoomListEntries := make(map[id.RoomID]*RoomListEntry)
	spacesChanged := false
	for evtType, ad := range sync.AccountData {
		evtType.Class = event.AccountDataEventType
		if evtType == AccountDataGomuksPreferences {
			parsedPreferences := DefaultPreferences
			_ = json.Unmarshal(ad.Content, &parsedPreferences)
			gs.PreferenceCache.Emit(&parsedPreferences)
		}
		gs.accountData[evtType] = ad
		gs.AccountDataSubs.Notify(evtType)
	}
	for _, data := range sync.InvitedRooms {
		inviteRoomStore := NewInvitedRoom(data, gs)
		gs.invitedRooms[data.ID] = inviteRoomStore
		if !resyncRoomList {
			changedRoomListEntries[data.ID] = inviteRoomStore.RoomListEntry
		}
	}
	for roomID, data := range sync.Rooms {
		data.Meta.EnsureNotNil()
		roomStore, existingRoom := gs.rooms[roomID]
		if !existingRoom {
			roomStore = NewRoomStore(gs, data.Meta)
			gs.rooms[roomID] = roomStore
		}
		entryChanged := !resyncRoomList && (!existingRoom || roomListEntryChanged(data, roomStore.Meta.Current()))
		roomStore.ApplySync(data)
		if entryChanged {
			changedRoomListEntries[roomID] = gs.makeRoomListEntry(roomStore)
		}
		if !existingRoom && !resyncRoomList && data.Meta.CreationContent.GetPredecessor().RoomID != "" {
			changedRoomListEntries[data.Meta.CreationContent.GetPredecessor().RoomID] = nil
		}
	}
	for _, roomID := range sync.LeftRooms {
		delete(gs.rooms, roomID)
		delete(gs.spaceEdges, roomID)
		for parentSpaceID, edges := range gs.spaceEdges {
			filtered := slices.DeleteFunc(edges, func(edge *database.SpaceEdge) bool {
				return edge.ChildID == roomID
			})
			if len(filtered) == 0 {
				delete(gs.spaceEdges, parentSpaceID)
			} else {
				gs.spaceEdges[parentSpaceID] = filtered
			}
		}
		changedRoomListEntries[roomID] = nil
	}

	if len(sync.SpaceEdges) > 0 {
		for spaceID, edges := range sync.SpaceEdges {
			if len(edges) == 0 {
				delete(gs.spaceEdges, spaceID)
			} else {
				gs.spaceEdges[spaceID] = edges
			}
		}
		spacesChanged = true
	}
	var updatedRoomList []*RoomListEntry
	if resyncRoomList {
		updatedRoomList = make([]*RoomListEntry, 0, len(gs.rooms)+len(gs.invitedRooms))
		for _, inviteRoom := range gs.invitedRooms {
			updatedRoomList = append(updatedRoomList, inviteRoom.RoomListEntry)
		}
		for _, roomStore := range gs.rooms {
			entry := gs.makeRoomListEntry(roomStore)
			if entry != nil {
				updatedRoomList = append(updatedRoomList, entry)
			}
		}
		slices.SortFunc(updatedRoomList, func(a, b *RoomListEntry) int {
			return a.SortingTimestamp.Compare(b.SortingTimestamp)
		})
	} else if len(changedRoomListEntries) > 0 {
		updatedRoomList = slices.DeleteFunc(gs.roomList, func(entry *RoomListEntry) bool {
			_, didChange := changedRoomListEntries[entry.RoomID]
			return didChange
		})
		for _, entry := range changedRoomListEntries {
			if entry == nil {
				continue
			}
			if len(updatedRoomList) == 0 || !entry.SortingTimestamp.Before(updatedRoomList[len(updatedRoomList)-1].SortingTimestamp) {
				updatedRoomList = append(updatedRoomList, entry)
			} else if entry.SortingTimestamp.Before(updatedRoomList[0].SortingTimestamp) {
				updatedRoomList = append([]*RoomListEntry{entry}, updatedRoomList...)
			} else {
				var i int
				for i = len(updatedRoomList) - 1; i >= 0; i-- {
					if updatedRoomList[i].SortingTimestamp.Before(entry.SortingTimestamp) {
						i++
						break
					}
				}
				updatedRoomList = slices.Insert(updatedRoomList, i, entry)
			}
		}
	}
	if updatedRoomList != nil {
		gs.roomList = updatedRoomList
		reversed := slices.Clone(updatedRoomList)
		slices.Reverse(reversed)
		gs.ReversedRoomList.Emit(reversed)
	}
	// A room becoming (or ceasing to be) a space changes the switcher contents
	// even when no m.space.child edges moved.
	if spacesChanged || updatedRoomList != nil {
		gs.SpaceList.Emit(gs.buildSpaceList())
	}
}

// buildSpaceList returns the joined spaces, sorted by name. Callers must hold
// at least a read lock.
func (gs *GomuksStore) buildSpaceList() []*SpaceEntry {
	spaces := make([]*SpaceEntry, 0, len(gs.spaceEdges))
	for _, roomStore := range gs.rooms {
		meta := roomStore.Meta.Current()
		if meta.GetType() != event.RoomTypeSpace {
			continue
		}
		name := ptr.Val(meta.Name)
		if name == "" {
			name = string(meta.ID)
		}
		spaces = append(spaces, &SpaceEntry{
			RoomID: meta.ID,
			Name:   name,
		})
	}
	slices.SortFunc(spaces, func(a, b *SpaceEntry) int {
		return strings.Compare(a.Name, b.Name)
	})
	return spaces
}

// GetSpaceList returns the joined spaces, sorted by name.
func (gs *GomuksStore) GetSpaceList() []*SpaceEntry {
	gs.lock.RLock()
	defer gs.lock.RUnlock()
	return gs.buildSpaceList()
}

// IsRoomInSpace reports whether the given room is a direct child of the space.
func (gs *GomuksStore) IsRoomInSpace(spaceID, roomID id.RoomID) bool {
	gs.lock.RLock()
	defer gs.lock.RUnlock()
	return slices.ContainsFunc(gs.spaceEdges[spaceID], func(edge *database.SpaceEdge) bool {
		return edge.ChildID == roomID
	})
}

// GetRoomsInSpace returns the room IDs that are direct children of the space.
func (gs *GomuksStore) GetRoomsInSpace(spaceID id.RoomID) []id.RoomID {
	gs.lock.RLock()
	defer gs.lock.RUnlock()
	edges := gs.spaceEdges[spaceID]
	rooms := make([]id.RoomID, 0, len(edges))
	for _, edge := range edges {
		rooms = append(rooms, edge.ChildID)
	}
	return rooms
}

func (gs *GomuksStore) GetRoom(roomID id.RoomID) *RoomStore {
	gs.lock.RLock()
	defer gs.lock.RUnlock()
	return gs.rooms[roomID]
}

func (gs *GomuksStore) GetInviteRoom(roomID id.RoomID) *InvitedRoom {
	gs.lock.RLock()
	defer gs.lock.RUnlock()
	return gs.invitedRooms[roomID]
}

func (gs *GomuksStore) Clear() {
	gs.lock.Lock()
	defer gs.lock.Unlock()
	clear(gs.rooms)
	clear(gs.invitedRooms)
	clear(gs.accountData)
	clear(gs.spaceEdges)
	gs.PreferenceCache.Emit(nil)
	gs.roomList = nil
	gs.ReversedRoomList.Emit([]*RoomListEntry{})
	gs.SpaceList.Emit([]*SpaceEntry{})
}
