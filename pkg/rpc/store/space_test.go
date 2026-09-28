// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package store_test

import (
	"slices"
	"testing"

	"go.mau.fi/util/ptr"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
	"go.mau.fi/gomuks/pkg/rpc/store"
)

func spaceRoom(roomID id.RoomID, name string) *jsoncmd.SyncRoom {
	return &jsoncmd.SyncRoom{Meta: &database.Room{
		ID:              roomID,
		Name:            ptr.Ptr(name),
		CreationContent: &event.CreateEventContent{Type: event.RoomTypeSpace},
	}}
}

func plainRoom(roomID id.RoomID, name string) *jsoncmd.SyncRoom {
	return &jsoncmd.SyncRoom{Meta: &database.Room{
		ID:   roomID,
		Name: ptr.Ptr(name),
	}}
}

const (
	spaceA = id.RoomID("!spaceA:example.org")
	spaceB = id.RoomID("!spaceB:example.org")
	roomX  = id.RoomID("!roomX:example.org")
	roomY  = id.RoomID("!roomY:example.org")
)

func newSyncedStore() *store.GomuksStore {
	st := store.NewStore()
	st.ApplySync(&jsoncmd.SyncComplete{
		Rooms: map[id.RoomID]*jsoncmd.SyncRoom{
			spaceA: spaceRoom(spaceA, "Beta Space"),
			spaceB: spaceRoom(spaceB, "Alpha Space"),
			roomX:  plainRoom(roomX, "Room X"),
			roomY:  plainRoom(roomY, "Room Y"),
		},
		SpaceEdges: map[id.RoomID][]*database.SpaceEdge{
			spaceA: {{ChildID: roomX}},
			spaceB: {{ChildID: roomY}},
		},
	})
	return st
}

func spaceNames(spaces []*store.SpaceEntry) []string {
	names := make([]string, len(spaces))
	for i, sp := range spaces {
		names[i] = sp.Name
	}
	return names
}

func TestGetSpaceList_EmptyBeforeSync(t *testing.T) {
	if got := store.NewStore().GetSpaceList(); len(got) != 0 {
		t.Errorf("expected no spaces before sync, got %d", len(got))
	}
}

func TestGetSpaceList_OnlySpacesSortedByName(t *testing.T) {
	got := spaceNames(newSyncedStore().GetSpaceList())
	want := []string{"Alpha Space", "Beta Space"}
	if !slices.Equal(got, want) {
		t.Errorf("expected spaces %v (regular rooms excluded), got %v", want, got)
	}
}

func TestIsRoomInSpace(t *testing.T) {
	st := newSyncedStore()
	cases := []struct {
		space id.RoomID
		room  id.RoomID
		want  bool
	}{
		{spaceA, roomX, true},
		{spaceA, roomY, false},
		{spaceB, roomY, true},
		{spaceB, roomX, false},
	}
	for _, tc := range cases {
		if got := st.IsRoomInSpace(tc.space, tc.room); got != tc.want {
			t.Errorf("IsRoomInSpace(%s, %s) = %v, want %v", tc.space, tc.room, got, tc.want)
		}
	}
}

func TestGetRoomsInSpace(t *testing.T) {
	st := newSyncedStore()
	if got := st.GetRoomsInSpace(spaceA); !slices.Equal(got, []id.RoomID{roomX}) {
		t.Errorf("expected space A to contain only room X, got %v", got)
	}
	if got := st.GetRoomsInSpace("!nonexistent:example.org"); len(got) != 0 {
		t.Errorf("expected no rooms for unknown space, got %v", got)
	}
}

func TestApplySync_LeavingSpaceDropsItsEdges(t *testing.T) {
	st := newSyncedStore()
	st.ApplySync(&jsoncmd.SyncComplete{LeftRooms: []id.RoomID{spaceA}})

	if st.IsRoomInSpace(spaceA, roomX) {
		t.Error("expected edges of the left space to be dropped")
	}
	if got := spaceNames(st.GetSpaceList()); !slices.Equal(got, []string{"Alpha Space"}) {
		t.Errorf("expected only the remaining space, got %v", got)
	}
}

func TestApplySync_EmptyEdgeListRemovesSpaceChildren(t *testing.T) {
	st := newSyncedStore()
	// The backend resends the full edge list per space, so an empty list means
	// the space has no children anymore.
	st.ApplySync(&jsoncmd.SyncComplete{
		SpaceEdges: map[id.RoomID][]*database.SpaceEdge{spaceA: {}},
	})

	if st.IsRoomInSpace(spaceA, roomX) {
		t.Error("expected an empty edge list to clear the space's children")
	}
}

func TestClear_RemovesSpaces(t *testing.T) {
	st := newSyncedStore()
	st.Clear()

	if got := st.GetSpaceList(); len(got) != 0 {
		t.Errorf("expected no spaces after Clear, got %d", len(got))
	}
	if st.IsRoomInSpace(spaceB, roomY) {
		t.Error("expected space edges to be cleared")
	}
}

func TestApplySync_LeavingChildRoomRemovesEdgeFromParent(t *testing.T) {
	st := newSyncedStore()
	if !st.IsRoomInSpace(spaceA, roomX) {
		t.Fatal("precondition failed: roomX should be in spaceA")
	}

	st.ApplySync(&jsoncmd.SyncComplete{LeftRooms: []id.RoomID{roomX}})

	if st.IsRoomInSpace(spaceA, roomX) {
		t.Error("expected roomX to be removed from spaceA edges after being left")
	}
	if got := st.GetRoomsInSpace(spaceA); len(got) != 0 {
		t.Errorf("expected spaceA to have no rooms, got %v", got)
	}
}

func TestGetSpaceList_UnnamedSpaceFallback(t *testing.T) {
	unnamedSpaceID := id.RoomID("!unnamed:example.org")
	st := store.NewStore()
	st.ApplySync(&jsoncmd.SyncComplete{
		Rooms: map[id.RoomID]*jsoncmd.SyncRoom{
			unnamedSpaceID: {
				Meta: &database.Room{
					ID:              unnamedSpaceID,
					CreationContent: &event.CreateEventContent{Type: event.RoomTypeSpace},
				},
			},
		},
	})

	spaces := st.GetSpaceList()
	if len(spaces) != 1 {
		t.Fatalf("expected 1 space, got %d", len(spaces))
	}
	if spaces[0].Name != string(unnamedSpaceID) {
		t.Errorf("expected fallback name %q, got %q", string(unnamedSpaceID), spaces[0].Name)
	}
}
