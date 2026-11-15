// gomuks - A terminal Matrix client written in Go.
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

package tui

import (
	"testing"

	"go.mau.fi/util/ptr"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
	"go.mau.fi/gomuks/pkg/rpc/client"
	"go.mau.fi/gomuks/pkg/rpc/store"
)

func TestRoomList_SpaceFiltering(t *testing.T) {
	st := store.NewStore()

	space1 := id.RoomID("!space1:example.org")
	space2 := id.RoomID("!space2:example.org")
	roomA := id.RoomID("!roomA:example.org")
	roomB := id.RoomID("!roomB:example.org")
	roomC := id.RoomID("!roomC:example.org")

	st.ApplySync(&jsoncmd.SyncComplete{
		Rooms: map[id.RoomID]*jsoncmd.SyncRoom{
			space1: {Meta: &database.Room{ID: space1, Name: ptr.Ptr("Space 1"), CreationContent: &event.CreateEventContent{Type: event.RoomTypeSpace}}},
			space2: {Meta: &database.Room{ID: space2, Name: ptr.Ptr("Space 2"), CreationContent: &event.CreateEventContent{Type: event.RoomTypeSpace}}},
			roomA:  {Meta: &database.Room{ID: roomA, Name: ptr.Ptr("Room A")}},
			roomB:  {Meta: &database.Room{ID: roomB, Name: ptr.Ptr("Room B")}},
			roomC:  {Meta: &database.Room{ID: roomC, Name: ptr.Ptr("Room C")}},
		},
		SpaceEdges: map[id.RoomID][]*database.SpaceEdge{
			space1: {{ChildID: roomA}, {ChildID: roomB}},
			space2: {{ChildID: roomC}},
		},
	})

	gmx := &client.GomuksClient{
		GomuksStore: st,
	}
	mainView := &MainView{
		matrix: gmx,
	}
	roomList := NewRoomList(mainView)

	// Initially All Rooms: contains all regular non-space rooms
	roomList.SetActiveSpace("")
	if len(roomList.rooms) != 3 {
		t.Fatalf("expected 3 non-space rooms in all rooms mode, got %d", len(roomList.rooms))
	}

	// Filter by space1
	roomList.SetActiveSpace(space1)
	if len(roomList.rooms) != 2 {
		t.Fatalf("expected 2 rooms in space 1, got %d", len(roomList.rooms))
	}

	// Filter by space2
	roomList.SetActiveSpace(space2)
	if len(roomList.rooms) != 1 {
		t.Fatalf("expected 1 room in space 2, got %d", len(roomList.rooms))
	}
	if roomList.rooms[0].RoomID != roomC {
		t.Errorf("expected roomC in space 2, got %s", roomList.rooms[0].RoomID)
	}

	// Fallback to all rooms
	roomList.SetActiveSpace("")
	if len(roomList.rooms) != 3 {
		t.Fatalf("expected 3 rooms after clearing space filter, got %d", len(roomList.rooms))
	}
}
