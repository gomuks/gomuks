package ui

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"sync"

	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"

	"go.mau.fi/mauview"

	"go.mau.fi/gomuks/tui/abstract"
	"go.mau.fi/gomuks/tui/ui/components"
)

type MainView struct {
	*mauview.Grid
	app abstract.App
	ctx context.Context

	RoomList    *components.RoomList
	Timelines   map[id.RoomID]*components.TimelineComponent
	MemberLists map[id.RoomID]*components.MemberList
	syncLock    sync.Mutex

	memberListElement *components.MemberList
	timelineElement   *components.TimelineComponent
	composerElement   *components.Composer
}

func (m *MainView) OnSync(resp *jsoncmd.SyncComplete) {
	logger := m.app.Gmx().Log.With().Str("component", "ui.sync").Logger()
	m.syncLock.Lock()
	defer m.syncLock.Unlock()
	for _, leftRoomID := range resp.LeftRooms {
		// Remove data for rooms we left
		logger.Debug().Stringer("room_id", leftRoomID).Msg("Removing left room from room list")
		delete(m.MemberLists, leftRoomID)
		delete(m.RoomList.Elements, leftRoomID)
	}
	// Add invited rooms to the top of the room list
	for _, room := range resp.InvitedRooms {
		//m.RoomList.AddRoom(room.ID)
		// bad!
		m.RoomList.AddRoom(m.ctx, room.ID, &jsoncmd.SyncRoom{})
		logger.Debug().Interface("room", room).Msg("Adding invited room to room list")
	}

	// Process joined rooms
	for roomID, room := range resp.Rooms {
		existingRoom := m.RoomList.Elements[roomID]
		if existingRoom != nil {
			logger.Debug().Interface("room", room).Msg("Updating existing room in room list")
			if room.Meta != nil && room.Meta.Name != nil && *room.Meta.Name != "" {
				// Update existing room name
				existingRoom.SetText(*room.Meta.Name)
			}
		} else {
			// Add new room
			logger.Debug().Interface("room", room).Msg("Adding new room to room list")
			m.RoomList.AddRoom(m.ctx, roomID, room)
		}

		timeline, exists := m.Timelines[roomID]
		if !exists {
			logger.Debug().Stringer("room_id", roomID).Msg("Creating new timeline for room")
			timeline = components.NewTimeline(m.ctx, m.app)
			m.Timelines[roomID] = timeline
		}
		if room.Events != nil {
			logger.Debug().Stringer("room_id", roomID).Msgf("Adding %d events to timeline", len(room.Events))
			for _, evt := range room.Events {
				logger.Debug().Interface("event", evt).Stringer("room_id", roomID).Msg("Adding event to timeline")
				timeline.AddEvent(evt)
			}
		}
	}
}

func (m *MainView) OnRoomSelected(ctx context.Context, old, new id.RoomID) {
	if old == new {
		m.app.Gmx().Log.Debug().Msgf("ignoring room switch from %s to itself", old)
		return
	}
	memberlist, ok := m.MemberLists[new]
	if !ok {
		m.app.Gmx().Log.Debug().Msgf("creating new member list for room %s", new)
		memberlist = components.NewMemberList(m.ctx, m.app, []id.UserID{}, nil)
		m.MemberLists[new] = memberlist
	}
	m.app.Gmx().Log.Debug().Msgf("switching to room view for %s from %s", old, new)
	evts, err := m.app.Rpc().GetRoomState(m.ctx, &jsoncmd.GetRoomStateParams{RoomID: new, IncludeMembers: true})
	if err == nil {
		var powerLevels event.PowerLevelsEventContent
		for _, evt := range evts {
			if evt.Type == "m.room.power_levels" && evt.StateKey != nil {
				if err = json.Unmarshal(evt.Content, &powerLevels); err != nil {
					m.app.Gmx().Log.Error().Err(err).Msgf("failed to parse power levels for room %s", new)
				}
			}
			if evt.Type == "m.room.member" && evt.StateKey != nil {
				var content event.MemberEventContent
				if err = json.Unmarshal(evt.Content, &content); err != nil {
					continue
				}
				if content.Membership == "join" {
					memberlist.Members = append(memberlist.Members, id.UserID(*evt.StateKey))
					m.app.Gmx().Log.Debug().Msgf("joined member %s", *evt.StateKey)
				}
			}
		}
		slices.SortStableFunc(memberlist.Members, func(a, b id.UserID) int {
			aPL := powerLevels.GetUserLevel(a)
			bPL := powerLevels.GetUserLevel(b)
			if aPL == bPL {
				return strings.Compare(a.String(), b.String())
			}
			return aPL - bPL
		})
	}
	m.RemoveComponent(m.memberListElement)
	m.memberListElement = memberlist
	m.memberListElement.Render()
	m.AddComponent(m.memberListElement, 2, 0, 1, 1)

	timeline := m.Timelines[new]
	if timeline == nil {
		m.app.Gmx().Log.Debug().Msgf("creating new timeline for room %s", new)
		timeline = components.NewTimeline(m.ctx, m.app)
		m.Timelines[new] = timeline
		//m.AddComponent(timeline, 1, 0, 1, 1)
	}
	// Fetch history for the new timeline
	history, err := m.app.Rpc().Paginate(ctx, &jsoncmd.PaginateParams{
		RoomID: new,
		Limit:  50,
		Reset:  true,
	})
	if err != nil {
		m.app.Gmx().Log.Error().Err(err).Msgf("failed to fetch history for room %s", new)
	} else {
		m.app.Gmx().Log.Debug().Msgf("adding %d events to timeline for room %s", len(history.Events), new)
		slices.Reverse(history.Events)
		for _, evt := range history.Events {
			m.app.Gmx().Log.Debug().Interface("event", evt).Stringer("room_id", new).Msg("Adding event to timeline")
			timeline.AddEvent(evt)
		}
	}
	m.app.Gmx().Log.Debug().Msgf("Removing timeline for room %s", old)
	m.RemoveComponent(m.timelineElement)
	m.timelineElement = timeline
	m.app.Gmx().Log.Debug().Msgf("Timeline for %s from %s", old, new)
	m.AddComponent(m.timelineElement, 1, 0, 1, 1)
	m.app.App().Redraw()
}

func NewMainView(ctx context.Context, app abstract.App) *MainView {
	//screenW, screenH := app.App().Screen().Size()
	var (
		screenW, screenH = 253, 65
	)
	m := &MainView{
		Grid:              mauview.NewGrid().SetColumns([]int{30, screenW - 60, 30}).SetRows([]int{screenH - 3, 3}),
		app:               app,
		ctx:               ctx,
		MemberLists:       make(map[id.RoomID]*components.MemberList),
		memberListElement: components.NewMemberList(ctx, app, []id.UserID{}, nil),
		Timelines:         make(map[id.RoomID]*components.TimelineComponent),
		timelineElement:   components.NewTimeline(ctx, app),
		composerElement:   components.NewComposer(ctx, app),
	}
	m.MemberLists[""] = m.memberListElement

	m.RoomList = components.NewRoomList(ctx, app, m.OnRoomSelected)

	m.AddComponent(m.RoomList, 0, 0, 1, 2)
	m.AddComponent(m.timelineElement, 1, 0, 1, 1)
	m.AddComponent(m.memberListElement, 2, 0, 1, 2)
	m.AddComponent(m.composerElement, 1, 1, 1, 1)
	return m
}
